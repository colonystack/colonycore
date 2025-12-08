// Package sqldocs exposes entity-model SQL bundles directly from the docs tree.
package sqldocs

import _ "embed"

// SQLite contains the generated entity-model SQLite DDL bundle.
//
//go:embed sqlite.sql
var SQLite string

// Postgres contains the generated entity-model Postgres DDL bundle.
//
//go:embed postgres.sql
var Postgres string
