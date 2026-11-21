package store

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// The store speaks two SQL dialects:
//   - SQLite (a local file, driver "sqlite") — default, for local runs;
//   - PostgreSQL (e.g. Supabase, driver "pgx") — selected by a postgres:// URL.
// All queries are written once with "?" placeholders; for Postgres they are
// rewritten to $1, $2, ... by rebind.

var ErrNoBackup = errors.New("backups are handled by the database provider")

type DB struct {
	*sql.DB
	pg bool
}

// rebind converts ? placeholders to $n for Postgres.
func rebind(q string) string {
	var sb strings.Builder
	n := 0
	for i := 0; i < len(q); i++ {
		if q[i] == '?' {
			n++
			sb.WriteByte('$')
			sb.WriteString(strconv.Itoa(n))
		} else {
			sb.WriteByte(q[i])
		}
	}
	return sb.String()
}

func (d *DB) rb(q string) string {
	if d.pg {
		return rebind(q)
	}
	return q
}

func (d *DB) Exec(q string, a ...any) (sql.Result, error) { return d.DB.Exec(d.rb(q), a...) }
func (d *DB) Query(q string, a ...any) (*sql.Rows, error) { return d.DB.Query(d.rb(q), a...) }
func (d *DB) QueryRow(q string, a ...any) *sql.Row        { return d.DB.QueryRow(d.rb(q), a...) }

// Tx is a transaction with the same placeholder rewriting.
type Tx struct {
	*sql.Tx
	d *DB
}

func (d *DB) Begin() (Tx, error) {
	t, err := d.DB.Begin()
	return Tx{t, d}, err
}

func (t Tx) Exec(q string, a ...any) (sql.Result, error) { return t.Tx.Exec(t.d.rb(q), a...) }
func (t Tx) QueryRow(q string, a ...any) *sql.Row        { return t.Tx.QueryRow(t.d.rb(q), a...) }

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

// IsPostgresURL reports whether target is a Postgres connection URL.
func IsPostgresURL(target string) bool {
	return strings.HasPrefix(target, "postgres://") || strings.HasPrefix(target, "postgresql://")
}

// Open opens and migrates the database. target is a postgres:// URL
// (Supabase) or the path of a SQLite file. sqliteDriver / pgDriver are the
// registered database/sql driver names ("sqlite", "pgx").
func Open(target, sqliteDriver, pgDriver string) (*DB, error) {
	if IsPostgresURL(target) {
		return openPostgres(target, pgDriver)
	}
	return openSQLite(target, sqliteDriver)
}

// ---------- SQLite ----------

func openSQLite(path, driver string) (*DB, error) {
	dsn := "file:" + path + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)"
	sdb, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, err
	}
	sdb.SetMaxOpenConns(1) // single writer; the bot handles updates sequentially
	if err := execAll(sdb, sqliteSchema); err != nil {
		sdb.Close()
		return nil, fmt.Errorf("schema: %w", err)
	}
	d := &DB{DB: sdb}
	var v int
	if err := sdb.QueryRow(`PRAGMA user_version`).Scan(&v); err != nil {
		sdb.Close()
		return nil, err
	}
	if err := d.applyMigrations(sqliteMigrations, v, func(i int) string {
		return fmt.Sprintf(`PRAGMA user_version = %d`, i)
	}); err != nil {
		sdb.Close()
		return nil, err
	}
	return d, nil
}

// sqliteMigrations run in order; PRAGMA user_version records how many were applied.
var sqliteMigrations = []string{
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

// ---------- PostgreSQL (Supabase) ----------

func openPostgres(target, driver string) (*DB, error) {
	dsn, err := pgDSN(target)
	if err != nil {
		return nil, err
	}
	sdb, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, err
	}
	// updates are handled one at a time, so two connections are plenty and
	// keep us far below the free-tier connection limits
	sdb.SetMaxOpenConns(2)
	sdb.SetMaxIdleConns(1)
	sdb.SetConnMaxLifetime(30 * time.Minute)
	sdb.SetConnMaxIdleTime(5 * time.Minute)
	if err := sdb.Ping(); err != nil {
		sdb.Close()
		return nil, fmt.Errorf("connect to postgres: %w", scrubDSN(err, target))
	}
	d := &DB{DB: sdb, pg: true}
	if _, err := sdb.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations(version INTEGER NOT NULL)`); err != nil {
		sdb.Close()
		return nil, err
	}
	var v int
	if err := sdb.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&v); err != nil {
		sdb.Close()
		return nil, err
	}
	if err := d.applyMigrations(pgMigrations, v, func(i int) string {
		return fmt.Sprintf(`INSERT INTO schema_migrations(version) VALUES(%d)`, i)
	}); err != nil {
		sdb.Close()
		return nil, err
	}
	return d, nil
}

// pgDSN makes the URL safe for Supabase's poolers: pgbouncer in transaction
// mode (port 6543) does not support prepared statements, so use the simple
// query protocol unless the URL already chooses a mode.
func pgDSN(target string) (string, error) {
	u, err := url.Parse(target)
	if err != nil {
		return "", errors.New("invalid DATABASE_URL")
	}
	q := u.Query()
	if q.Get("default_query_exec_mode") == "" {
		q.Set("default_query_exec_mode", "simple_protocol")
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func scrubDSN(err error, target string) error {
	msg := err.Error()
	if u, perr := url.Parse(target); perr == nil && u.User != nil {
		if pw, ok := u.User.Password(); ok && pw != "" {
			msg = strings.ReplaceAll(msg, pw, "<password>")
		}
	}
	return errors.New(msg)
}

// Same tables as SQLite, with 64-bit ids/times (Telegram ids exceed int32).
var pgMigrations = []string{
	// v1: base game, inventory, equipment, clans
	`CREATE TABLE IF NOT EXISTS players(
		user_id    BIGINT PRIMARY KEY,
		name       TEXT NOT NULL,
		race       INTEGER NOT NULL,
		female     INTEGER NOT NULL,
		elem1      INTEGER NOT NULL,
		elem2      INTEGER NOT NULL DEFAULT -1,
		level      INTEGER NOT NULL DEFAULT 1,
		xp         BIGINT  NOT NULL DEFAULT 0,
		gold       BIGINT  NOT NULL DEFAULT 0,
		points     INTEGER NOT NULL DEFAULT 5,
		alloc      TEXT    NOT NULL DEFAULT '0,0,0,0,0,0,0,0,0',
		bought     INTEGER NOT NULL DEFAULT 0,
		race_tier  INTEGER NOT NULL DEFAULT 0,
		hp         DOUBLE PRECISION NOT NULL,
		mana       DOUBLE PRECISION NOT NULL,
		updated_at BIGINT  NOT NULL,
		dead_until BIGINT  NOT NULL DEFAULT 0,
		created_at BIGINT  NOT NULL,
		kills      INTEGER NOT NULL DEFAULT 0,
		clan_id    BIGINT  NOT NULL DEFAULT 0);
	 CREATE TABLE IF NOT EXISTS spell_uses(
		user_id BIGINT NOT NULL, ability TEXT NOT NULL, uses INTEGER NOT NULL,
		PRIMARY KEY(user_id, ability));
	 CREATE TABLE IF NOT EXISTS texts(key TEXT PRIMARY KEY, value TEXT NOT NULL);
	 CREATE TABLE IF NOT EXISTS inventory(
		user_id BIGINT NOT NULL, item TEXT NOT NULL, qty INTEGER NOT NULL,
		PRIMARY KEY(user_id, item));
	 CREATE TABLE IF NOT EXISTS equipped(
		user_id BIGINT NOT NULL, slot INTEGER NOT NULL, item TEXT NOT NULL,
		PRIMARY KEY(user_id, slot));
	 CREATE TABLE IF NOT EXISTS clans(
		id BIGSERIAL PRIMARY KEY,
		name TEXT NOT NULL,
		leader_id BIGINT NOT NULL,
		created_at BIGINT NOT NULL);
	 CREATE UNIQUE INDEX IF NOT EXISTS clans_name_ci ON clans(lower(name));
	 CREATE INDEX IF NOT EXISTS players_clan ON players(clan_id)`,
	// v2: stat resets, photo cache
	`ALTER TABLE players ADD COLUMN IF NOT EXISTS resets INTEGER NOT NULL DEFAULT 0;
	 CREATE TABLE IF NOT EXISTS photos(
		key TEXT PRIMARY KEY, file_id TEXT NOT NULL, hash TEXT NOT NULL DEFAULT '')`,
	// v3: Supabase exposes the public schema through its Data API. RLS without
	// any policy denies that API completely, while the bot (connecting as the
	// table owner / postgres role) is not affected.
	`ALTER TABLE players ENABLE ROW LEVEL SECURITY;
	 ALTER TABLE spell_uses ENABLE ROW LEVEL SECURITY;
	 ALTER TABLE texts ENABLE ROW LEVEL SECURITY;
	 ALTER TABLE inventory ENABLE ROW LEVEL SECURITY;
	 ALTER TABLE equipped ENABLE ROW LEVEL SECURITY;
	 ALTER TABLE clans ENABLE ROW LEVEL SECURITY;
	 ALTER TABLE photos ENABLE ROW LEVEL SECURITY;
	 ALTER TABLE schema_migrations ENABLE ROW LEVEL SECURITY`,
}

// applyMigrations runs migrations[done:] each in its own transaction and
// records the version with record(i).
func (d *DB) applyMigrations(migs []string, done int, record func(version int) string) error {
	for i := done; i < len(migs); i++ {
		tx, err := d.Begin()
		if err != nil {
			return err
		}
		if err := execAll(tx, migs[i]); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %d: %w", i+1, err)
		}
		if _, err := tx.Exec(record(i + 1)); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}
