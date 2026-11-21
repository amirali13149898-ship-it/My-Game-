// Command ardian runs the Ardian Telegram game bot.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // Postgres driver ("pgx") for Supabase
	_ "modernc.org/sqlite"             // pure-Go SQLite driver (no CGO)

	"ardian/internal/bot"
	"ardian/internal/store"
)

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int64) int64 {
	if v := os.Getenv(key); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			log.Fatalf("%s must be a number, got %q", key, v)
		}
		return n
	}
	return def
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		log.Fatal("BOT_TOKEN is required")
	}
	// DATABASE_URL (postgres://..., e.g. Supabase) wins; otherwise a local SQLite file.
	dbPath := env("DB_PATH", "ardian.db")
	target := os.Getenv("DATABASE_URL")
	if target == "" {
		target = dbPath
		if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
			log.Fatalf("db dir: %v", err)
		}
		log.Printf("database: SQLite file %s", dbPath)
	} else {
		log.Printf("database: PostgreSQL (DATABASE_URL)")
	}
	db, err := store.Open(target, "sqlite", "pgx")
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	b := bot.New(bot.Config{
		Token:       token,
		AdminID:     envInt("ADMIN_ID", 0),
		BackupDir:   env("BACKUP_DIR", filepath.Join(filepath.Dir(dbPath), "backups")),
		BackupEvery: time.Duration(envInt("BACKUP_EVERY_HOURS", 6)) * time.Hour,
		BackupKeep:  int(envInt("BACKUP_KEEP", 10)),
		AssetsDir:   env("ASSETS_DIR", "assets"),
	}, db)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() { // a long-poll request cannot be interrupted; exit shortly after the signal
		<-ctx.Done()
		time.Sleep(2 * time.Second)
		db.Close()
		os.Exit(0)
	}()
	if err := b.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
