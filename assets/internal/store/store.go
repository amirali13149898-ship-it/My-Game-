// Package store persists players in SQLite. It only uses database/sql; the
// driver (modernc.org/sqlite, pure Go, no CGO) is imported by cmd/ardian.
package store

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"ardian/internal/game"
)

const schema = `
CREATE TABLE IF NOT EXISTS players(
	user_id    INTEGER PRIMARY KEY,
	name       TEXT    NOT NULL,
	race       INTEGER NOT NULL,
	female     INTEGER NOT NULL,
	elem1      INTEGER NOT NULL,
	elem2      INTEGER NOT NULL DEFAULT -1,
	level      INTEGER NOT NULL DEFAULT 1,
	xp         INTEGER NOT NULL DEFAULT 0,
	gold       INTEGER NOT NULL DEFAULT 0,
	points     INTEGER NOT NULL DEFAULT 5,
	alloc      TEXT    NOT NULL DEFAULT '0,0,0,0,0,0,0,0,0',
	bought     INTEGER NOT NULL DEFAULT 0,
	race_tier  INTEGER NOT NULL DEFAULT 0,
	hp         REAL    NOT NULL,
	mana       REAL    NOT NULL,
	updated_at INTEGER NOT NULL,
	dead_until INTEGER NOT NULL DEFAULT 0,
	created_at INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS spell_uses(
	user_id INTEGER NOT NULL,
	ability TEXT    NOT NULL,
	uses    INTEGER NOT NULL,
	PRIMARY KEY(user_id, ability)
);
CREATE TABLE IF NOT EXISTS texts(
	key   TEXT PRIMARY KEY,
	value TEXT NOT NULL
);
`

// migrations run in order; PRAGMA user_version records how many were applied.
var migrations = []string{
	// v1: kills, clan membership, inventory, equipment, clans
	`ALTER TABLE players ADD COLUMN kills INTEGER NOT NULL DEFAULT 0;
	 ALTER TABLE players ADD COLUMN clan_id INTEGER NOT NULL DEFAULT 0;
	 CREATE TABLE IF NOT EXISTS inventory(
		user_id INTEGER NOT NULL, item TEXT NOT NULL, qty INTEGER NOT NULL,
		PRIMARY KEY(user_id, item));
	 CREATE TABLE IF NOT EXISTS equipped(
		user_id INTEGER NOT NULL, slot INTEGER NOT NULL, item TEXT NOT NULL,
		PRIMARY KEY(user_id, slot));
	 CREATE TABLE IF NOT EXISTS clans(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE COLLATE NOCASE,
		leader_id INTEGER NOT NULL,
		created_at INTEGER NOT NULL);
	 CREATE INDEX IF NOT EXISTS players_clan ON players(clan_id);`,
	// v2: stat resets, photo cache (key → Telegram file_id + content hash)
	`ALTER TABLE players ADD COLUMN resets INTEGER NOT NULL DEFAULT 0;
	 CREATE TABLE IF NOT EXISTS photos(
		key TEXT PRIMARY KEY, file_id TEXT NOT NULL, hash TEXT NOT NULL DEFAULT '');`,
}

func migrate(db *sql.DB) error {
	var v int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&v); err != nil {
		return err
	}
	for i := v; i < len(migrations); i++ {
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		if err := execAll(tx, migrations[i]); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %d: %w", i+1, err)
		}
		if _, err := tx.Exec(fmt.Sprintf(`PRAGMA user_version = %d`, i+1)); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

type execer interface {
	Exec(query string, args ...any) (sql.Result, error)
}

// execAll runs a multi-statement script one statement at a time (does not
// depend on the driver supporting several statements per Exec).
func execAll(x execer, script string) error {
	for _, stmt := range strings.Split(script, ";") {
		if strings.TrimSpace(stmt) == "" {
			continue
		}
		if _, err := x.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

type DB struct{ *sql.DB }

// Open opens (and migrates) the database file with the given driver name.
func Open(driver, path string) (*DB, error) {
	dsn := "file:" + path + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)"
	sdb, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, err
	}
	sdb.SetMaxOpenConns(1) // single writer; the bot handles updates sequentially
	if err := execAll(sdb, schema); err != nil {
		sdb.Close()
		return nil, fmt.Errorf("schema: %w", err)
	}
	if err := migrate(sdb); err != nil {
		sdb.Close()
		return nil, err
	}
	return &DB{sdb}, nil
}

func ms(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UnixMilli()
}

func fromMs(v int64) time.Time {
	if v == 0 {
		return time.Time{}
	}
	return time.UnixMilli(v)
}

func encodeAlloc(a [game.NumStats]int) string {
	p := make([]string, len(a))
	for i, v := range a {
		p[i] = strconv.Itoa(v)
	}
	return strings.Join(p, ",")
}

func decodeAlloc(s string) (a [game.NumStats]int) {
	for i, p := range strings.Split(s, ",") {
		if i >= len(a) {
			break
		}
		a[i], _ = strconv.Atoi(p)
	}
	return
}

// CreateFighter inserts a brand-new player.
func (d *DB) CreateFighter(f *game.Fighter, now time.Time) error {
	e2 := -1
	if len(f.Elems) > 1 {
		e2 = int(f.Elems[1])
	}
	_, err := d.Exec(`INSERT INTO players(user_id,name,race,female,elem1,elem2,level,xp,gold,points,alloc,
		bought,race_tier,hp,mana,updated_at,dead_until,created_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		f.ID, f.Name, int(f.Race), b2i(f.Female), int(f.Elems[0]), e2, f.Level, f.XP, f.Gold, f.Points,
		encodeAlloc(f.Alloc), f.Bought, f.RaceTier, f.HP, f.Mana, ms(f.Updated), ms(f.DeadUntil), ms(now))
	return err
}

// LoadFighter returns (nil, nil) when the user has no character.
func (d *DB) LoadFighter(id int64) (*game.Fighter, error) {
	var (
		f                    = &game.Fighter{ID: id}
		race, female, e1, e2 int
		alloc                string
		updated, dead        int64
	)
	err := d.QueryRow(`SELECT name,race,female,elem1,elem2,level,xp,gold,points,alloc,bought,race_tier,
		hp,mana,updated_at,dead_until,kills,clan_id,resets FROM players WHERE user_id=?`, id).
		Scan(&f.Name, &race, &female, &e1, &e2, &f.Level, &f.XP, &f.Gold, &f.Points, &alloc, &f.Bought,
			&f.RaceTier, &f.HP, &f.Mana, &updated, &dead, &f.Kills, &f.ClanID, &f.Resets)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	f.Race, f.Female = game.Race(race), female == 1
	f.Elems = []game.Element{game.Element(e1)}
	if e2 >= 0 {
		f.Elems = append(f.Elems, game.Element(e2))
	}
	f.Alloc = decodeAlloc(alloc)
	f.Updated, f.DeadUntil = fromMs(updated), fromMs(dead)
	f.Init()

	rows, err := d.Query(`SELECT ability, uses FROM spell_uses WHERE user_id=?`, id)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var name string
		var n int
		if err := rows.Scan(&name, &n); err != nil {
			rows.Close()
			return nil, err
		}
		f.Uses[name] = n
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := d.loadGear(f); err != nil {
		return nil, err
	}
	return f, nil
}

func (d *DB) loadGear(f *game.Fighter) error {
	rows, err := d.Query(`SELECT item, qty FROM inventory WHERE user_id=?`, f.ID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id string
		var n int
		if err := rows.Scan(&id, &n); err != nil {
			rows.Close()
			return err
		}
		f.Inv[id] = n
	}
	rows.Close()
	rows, err = d.Query(`SELECT slot, item FROM equipped WHERE user_id=?`, f.ID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var slot int
		var id string
		if err := rows.Scan(&slot, &id); err != nil {
			rows.Close()
			return err
		}
		if slot >= 0 && slot < game.NumSlots {
			f.Equipped[slot] = id
		}
	}
	rows.Close()
	f.RecalcGear()
	return nil
}

// SaveGear rewrites the inventory and equipment rows of a player atomically.
func (d *DB) SaveGear(f *game.Fighter) error {
	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM inventory WHERE user_id=?`, f.ID); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM equipped WHERE user_id=?`, f.ID); err != nil {
		return err
	}
	for id, n := range f.Inv {
		if n > 0 {
			if _, err := tx.Exec(`INSERT INTO inventory(user_id,item,qty) VALUES(?,?,?)`, f.ID, id, n); err != nil {
				return err
			}
		}
	}
	for slot, id := range f.Equipped {
		if id != "" {
			if _, err := tx.Exec(`INSERT INTO equipped(user_id,slot,item) VALUES(?,?,?)`, f.ID, slot, id); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

// SaveFighter writes the persistent part of the fighter state.
func (d *DB) SaveFighter(f *game.Fighter) error {
	_, err := d.Exec(`UPDATE players SET name=?,level=?,xp=?,gold=?,points=?,alloc=?,bought=?,race_tier=?,
		hp=?,mana=?,updated_at=?,dead_until=?,kills=?,resets=? WHERE user_id=?`,
		f.Name, f.Level, f.XP, f.Gold, f.Points, encodeAlloc(f.Alloc), f.Bought, f.RaceTier,
		f.HP, f.Mana, ms(f.Updated), ms(f.DeadUntil), f.Kills, f.Resets, f.ID)
	return err
}

func (d *DB) SaveUses(id int64, ability string, uses int) error {
	_, err := d.Exec(`INSERT INTO spell_uses(user_id,ability,uses) VALUES(?,?,?)
		ON CONFLICT(user_id,ability) DO UPDATE SET uses=excluded.uses`, id, ability, uses)
	return err
}

func (d *DB) Text(key string) (string, bool, error) {
	var v string
	err := d.QueryRow(`SELECT value FROM texts WHERE key=?`, key).Scan(&v)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	return v, err == nil, err
}

func (d *DB) SetText(key, value string) error {
	_, err := d.Exec(`INSERT INTO texts(key,value) VALUES(?,?)
		ON CONFLICT(key) DO UPDATE SET value=excluded.value`, key, value)
	return err
}

func (d *DB) PlayerCount() (n int, err error) {
	err = d.QueryRow(`SELECT COUNT(*) FROM players`).Scan(&n)
	return
}

// Backup writes a consistent snapshot of the database to dest.
func (d *DB) Backup(dest string) error {
	_, err := d.Exec(`VACUUM INTO ?`, dest)
	return err
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}
