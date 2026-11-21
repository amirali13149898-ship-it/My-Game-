package store

import (
	"strings"
	"testing"
)

func TestRebind(t *testing.T) {
	got := rebind(`UPDATE t SET a=?, b=? WHERE id=? AND c='x'`)
	if got != `UPDATE t SET a=$1, b=$2 WHERE id=$3 AND c='x'` {
		t.Fatal(got)
	}
	if rebind("SELECT 1") != "SELECT 1" {
		t.Fatal("no-op rebind changed the query")
	}
}

func TestPostgresURL(t *testing.T) {
	for _, u := range []string{"postgres://u:p@h:5432/db", "postgresql://u@h/db"} {
		if !IsPostgresURL(u) {
			t.Errorf("%s not detected", u)
		}
	}
	if IsPostgresURL("/data/ardian.db") || IsPostgresURL("ardian.db") {
		t.Error("sqlite path detected as postgres")
	}
	dsn, err := pgDSN("postgresql://postgres.ref:p%40ss@aws-0-eu.pooler.supabase.com:6543/postgres?sslmode=require")
	if err != nil || !strings.Contains(dsn, "default_query_exec_mode=simple_protocol") || !strings.Contains(dsn, "sslmode=require") {
		t.Fatalf("dsn %q err %v", dsn, err)
	}
	if d2, _ := pgDSN("postgres://h/db?default_query_exec_mode=exec"); strings.Contains(d2, "simple_protocol") {
		t.Fatal("user-chosen exec mode must be kept")
	}
	e := scrubDSN(errString("dial failed for hunter2"), "postgres://u:hunter2@h/db")
	if strings.Contains(e.Error(), "hunter2") {
		t.Fatal("password leaked into error")
	}
}

type errString string

func (e errString) Error() string { return string(e) }
