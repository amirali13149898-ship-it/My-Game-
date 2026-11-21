package store

import (
	"database/sql"
	"errors"
	"strings"
	"time"
)

// ---------- clans ----------

var (
	ErrNameTaken = errors.New("این اسم قبلاً گرفته شده")
	ErrClanFull  = errors.New("ظرفیت کلن پر است")
	ErrNoClan    = errors.New("کلن پیدا نشد")
)

const (
	ClanMaxMembers = 30
	ClanMinName    = 3
	ClanMaxName    = 20
)

type Clan struct {
	ID         int64
	Name       string
	LeaderID   int64
	Members    int
	TotalLevel int
	Kills      int
}

type Member struct {
	ID    int64
	Name  string
	Level int
	Kills int
}

const clanSelect = `SELECT c.id, c.name, c.leader_id, COUNT(p.user_id), CAST(COALESCE(SUM(p.level),0) AS BIGINT), CAST(COALESCE(SUM(p.kills),0) AS BIGINT)
	FROM clans c LEFT JOIN players p ON p.clan_id = c.id`

func (d *DB) scanClans(query string, args ...any) ([]Clan, error) {
	rows, err := d.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Clan
	for rows.Next() {
		var c Clan
		if err := rows.Scan(&c.ID, &c.Name, &c.LeaderID, &c.Members, &c.TotalLevel, &c.Kills); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (d *DB) Clan(id int64) (*Clan, error) {
	cs, err := d.scanClans(clanSelect+` WHERE c.id=? GROUP BY c.id`, id)
	if err != nil || len(cs) == 0 {
		return nil, err
	}
	return &cs[0], nil
}

// ListClans returns clans ordered by total level.
func (d *DB) ListClans(limit, offset int) ([]Clan, error) {
	return d.scanClans(clanSelect+` GROUP BY c.id ORDER BY 5 DESC, c.id LIMIT ? OFFSET ?`, limit, offset)
}

func (d *DB) ClanCount() (n int, err error) {
	err = d.QueryRow(`SELECT COUNT(*) FROM clans`).Scan(&n)
	return
}

func (d *DB) ClanMembers(id int64) ([]Member, error) {
	rows, err := d.Query(`SELECT user_id, name, level, kills FROM players WHERE clan_id=? ORDER BY level DESC, kills DESC`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Member
	for rows.Next() {
		var m Member
		if err := rows.Scan(&m.ID, &m.Name, &m.Level, &m.Kills); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// CreateClan creates a clan and makes the leader its first member.
func (d *DB) CreateClan(name string, leader int64, now time.Time) (int64, error) {
	name = strings.TrimSpace(name)
	var n int
	if err := d.QueryRow(`SELECT COUNT(*) FROM clans WHERE lower(name)=lower(?)`, name).Scan(&n); err != nil {
		return 0, err
	}
	if n > 0 {
		return 0, ErrNameTaken
	}
	tx, err := d.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var id int64
	if err := tx.QueryRow(`INSERT INTO clans(name, leader_id, created_at) VALUES(?,?,?) RETURNING id`,
		name, leader, now.UnixMilli()).Scan(&id); err != nil {
		return 0, err
	}
	if _, err := tx.Exec(`UPDATE players SET clan_id=? WHERE user_id=?`, id, leader); err != nil {
		return 0, err
	}
	return id, tx.Commit()
}

func (d *DB) JoinClan(user, clan int64) error {
	c, err := d.Clan(clan)
	if err != nil {
		return err
	}
	if c == nil {
		return ErrNoClan
	}
	if c.Members >= ClanMaxMembers {
		return ErrClanFull
	}
	_, err = d.Exec(`UPDATE players SET clan_id=? WHERE user_id=? AND clan_id=0`, clan, user)
	return err
}

func (d *DB) LeaveClan(user int64) error {
	_, err := d.Exec(`UPDATE players SET clan_id=0 WHERE user_id=?`, user)
	return err
}

// DeleteClan removes the clan and returns the ids of its former members.
func (d *DB) DeleteClan(id int64) ([]int64, error) {
	ms, err := d.ClanMembers(id)
	if err != nil {
		return nil, err
	}
	tx, err := d.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`UPDATE players SET clan_id=0 WHERE clan_id=?`, id); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(`DELETE FROM clans WHERE id=?`, id); err != nil {
		return nil, err
	}
	ids := make([]int64, len(ms))
	for i, m := range ms {
		ids[i] = m.ID
	}
	return ids, tx.Commit()
}

// ---------- leaderboards ----------

type RankRow struct {
	ID    int64
	Name  string
	Level int
	XP    int64
	Kills int
	Gold  int64
}

var rankOrder = map[string]string{
	"level": "level DESC, xp DESC, user_id",
	"kills": "kills DESC, level DESC, user_id",
	"gold":  "gold DESC, level DESC, user_id",
}

func (d *DB) TopPlayers(kind string, limit int) ([]RankRow, error) {
	order, ok := rankOrder[kind]
	if !ok {
		order = rankOrder["level"]
	}
	rows, err := d.Query(`SELECT user_id, name, level, xp, kills, gold FROM players ORDER BY `+order+` LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []RankRow
	for rows.Next() {
		var r RankRow
		if err := rows.Scan(&r.ID, &r.Name, &r.Level, &r.XP, &r.Kills, &r.Gold); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// PlayerRank is the 1-based position of a player in a board.
func (d *DB) PlayerRank(kind string, me RankRow) (rank int, err error) {
	var q string
	var args []any
	switch kind {
	case "kills":
		q, args = `kills > ? OR (kills = ? AND level > ?) OR (kills = ? AND level = ? AND user_id < ?)`,
			[]any{me.Kills, me.Kills, me.Level, me.Kills, me.Level, me.ID}
	case "gold":
		q, args = `gold > ? OR (gold = ? AND level > ?) OR (gold = ? AND level = ? AND user_id < ?)`,
			[]any{me.Gold, me.Gold, me.Level, me.Gold, me.Level, me.ID}
	default:
		q, args = `level > ? OR (level = ? AND xp > ?) OR (level = ? AND xp = ? AND user_id < ?)`,
			[]any{me.Level, me.Level, me.XP, me.Level, me.XP, me.ID}
	}
	err = d.QueryRow(`SELECT COUNT(*) + 1 FROM players WHERE `+q, args...).Scan(&rank)
	return
}

// ---------- photos ----------

// Photo returns the cached Telegram file_id of a picture key and the content
// hash it was uploaded from (empty when set by the admin from Telegram).
func (d *DB) Photo(key string) (fileID, hash string, ok bool, err error) {
	err = d.QueryRow(`SELECT file_id, hash FROM photos WHERE key=?`, key).Scan(&fileID, &hash)
	if err == sql.ErrNoRows {
		return "", "", false, nil
	}
	return fileID, hash, err == nil, err
}

func (d *DB) SetPhoto(key, fileID, hash string) error {
	_, err := d.Exec(`INSERT INTO photos(key,file_id,hash) VALUES(?,?,?)
		ON CONFLICT(key) DO UPDATE SET file_id=excluded.file_id, hash=excluded.hash`, key, fileID, hash)
	return err
}

func (d *DB) PhotoKeys() ([]string, error) {
	rows, err := d.Query(`SELECT key FROM photos ORDER BY key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ks []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		ks = append(ks, k)
	}
	return ks, rows.Err()
}
