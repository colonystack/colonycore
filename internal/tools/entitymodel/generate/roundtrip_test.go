package main

import (
	"bufio"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"colonycore/internal/entitymodel/sqlbundle"
	_ "modernc.org/sqlite"
)

func TestEntityModelRoundTripSmoke(t *testing.T) {
	schemaPath := repoPath("docs/schema/entity-model.json")
	doc, err := loadSchema(schemaPath)
	if err != nil {
		t.Fatalf("load schema: %v", err)
	}

	pgDDL, sqliteDDL, err := generateSQL(doc)
	if err != nil {
		t.Fatalf("generate SQL: %v", err)
	}

	mustMatchFile(t, repoPath("docs/schema/sql/postgres.sql"), pgDDL)
	mustMatchFile(t, repoPath("docs/schema/sql/sqlite.sql"), sqliteDDL)

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	for _, stmt := range sqlbundle.SplitStatements(string(sqliteDDL)) {
		if strings.TrimSpace(stmt) == "" {
			continue
		}
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("apply ddl: %v", err)
		}
	}

	requiredTables := extractTableNames(string(sqliteDDL))
	if len(requiredTables) == 0 {
		t.Fatalf("expected tables parsed from DDL")
	}
	for _, tbl := range requiredTables {
		var name string
		if err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, tbl).Scan(&name); err != nil {
			t.Fatalf("missing table %s: %v", tbl, err)
		}
	}

	erd, err := os.ReadFile(repoPath("docs/annex/entity-model-erd.dot")) //nolint:gosec // ERD path is generated within the repo
	if err != nil {
		t.Fatalf("read erd: %v", err)
	}
	erdContent := string(erd)
	for _, tbl := range requiredTables {
		needle := "<B>" + tbl + "</B>"
		if !strings.Contains(erdContent, needle) {
			t.Fatalf("ERD missing table %s", tbl)
		}
	}
}

func mustMatchFile(t *testing.T, path string, data []byte) {
	t.Helper()
	disk, err := os.ReadFile(path) //nolint:gosec // corpus lives inside the repository
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(disk) != string(data) {
		t.Fatalf("%s out of sync; run make entity-model-generate", path)
	}
}

func extractTableNames(ddl string) []string {
	scanner := bufio.NewScanner(strings.NewReader(ddl))
	var tables []string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "CREATE TABLE IF NOT EXISTS") {
			parts := strings.Fields(line)
			for i, token := range parts {
				if token == "(" && i > 0 {
					name := parts[i-1]
					name = strings.Trim(name, "`")
					tables = append(tables, name)
					break
				}
			}
		}
	}
	return tables
}

func repoPath(rel string) string {
	return filepath.Join("..", "..", "..", "..", rel)
}
