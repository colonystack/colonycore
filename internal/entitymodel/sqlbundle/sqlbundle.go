// Package sqlbundle exposes generated entity-model DDL bundles for adapters.
package sqlbundle

import (
	"bufio"
	"strings"

	sqldocs "colonycore/docs/schema/sql"
)

// SQLite returns the generated SQLite DDL for the entity model.
func SQLite() string {
	return sqldocs.SQLite
}

// Postgres returns the generated Postgres DDL for the entity model.
func Postgres() string {
	return sqldocs.Postgres
}

// SplitStatements splits a semicolon-terminated DDL script into executable statements.
// It drops blank lines and single-line comments that start with "--".
func SplitStatements(ddl string) []string {
	scanner := bufio.NewScanner(strings.NewReader(ddl))
	var stmts []string
	var current strings.Builder

	flush := func() {
		stmt := strings.TrimSpace(current.String())
		if stmt != "" {
			stmts = append(stmts, stmt)
		}
		current.Reset()
	}

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "--") {
			continue
		}
		current.WriteString(line)
		current.WriteByte('\n')
		if strings.HasSuffix(trimmed, ";") {
			flush()
		}
	}

	if tail := strings.TrimSpace(current.String()); tail != "" {
		stmts = append(stmts, tail)
	}

	return stmts
}
