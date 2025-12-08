package sqlbundle

import (
	"strings"
	"testing"
)

func TestSplitStatements(t *testing.T) {
	stmts := SplitStatements(SQLite())
	if len(stmts) == 0 {
		t.Fatal("expected sqlite DDL to produce statements")
	}
	for _, stmt := range stmts {
		if strings.HasPrefix(strings.TrimSpace(stmt), "--") {
			t.Fatalf("statement unexpectedly starts with comment: %q", stmt)
		}
		if !strings.HasSuffix(strings.TrimSpace(stmt), ";") {
			t.Fatalf("statement missing semicolon terminator: %q", stmt)
		}
	}
}

func TestPostgresBundle(t *testing.T) {
	if !strings.Contains(Postgres(), "CREATE TABLE") {
		t.Fatal("expected postgres DDL to contain CREATE TABLE")
	}
}
