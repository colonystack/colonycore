// Package datasetapi defines dataset templating primitives, descriptors,
// execution types, and plugin binding contracts.
package datasetapi

import (
	"colonycore/pkg/domain"
	"context"
	"time"
)

// Dialect identifies the query language a template uses.
type Dialect string

const (
	// DialectSQL is the standard SQL dialect.
	DialectSQL Dialect = "sql"
	// DialectDSL is the custom domain-specific language dialect.
	DialectDSL Dialect = "dsl"
)

// Format identifies an output encoding for a dataset result.
type Format string

const (
	// FormatJSON is JSON document output.
	FormatJSON Format = "json"
	// FormatCSV is comma separated values output.
	FormatCSV Format = "csv"
	// FormatParquet is columnar Parquet output.
	FormatParquet Format = "parquet"
	// FormatPNG is PNG image output.
	FormatPNG Format = "png"
	// FormatHTML is rendered HTML output.
	FormatHTML Format = "html"
)

// Scope defines requestor identity and authorization context.
type Scope struct {
	Requestor   string   `json:"requestor"`
	Roles       []string `json:"roles,omitempty"`
	ProjectIDs  []string `json:"project_ids,omitempty"`
	ProtocolIDs []string `json:"protocol_ids,omitempty"`
}

// Parameter declares a runtime-supplied template parameter.
type Parameter struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Required    bool     `json:"required"`
	Description string   `json:"description,omitempty"`
	Unit        string   `json:"unit,omitempty"`
	Enum        []string `json:"enum,omitempty"`
	Example     any      `json:"example,omitempty"`
	Default     any      `json:"default,omitempty"`
}

// Column describes a column returned by a dataset query.
type Column struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Unit        string `json:"unit,omitempty"`
	Description string `json:"description,omitempty"`
	Format      string `json:"format,omitempty"`
}

// Metadata provides descriptive and operational metadata for a template.
type Metadata struct {
	Source          string            `json:"source,omitempty"`
	Documentation   string            `json:"documentation,omitempty"`
	RefreshInterval string            `json:"refresh_interval,omitempty"`
	Tags            []string          `json:"tags,omitempty"`
	Annotations     map[string]string `json:"annotations,omitempty"`
}

// Environment bundles dependencies needed for binding template runners.
type Environment struct {
	Store domain.PersistentStore
	Now   func() time.Time
}

// Template is the in-memory representation of a dataset template.
type Template struct {
	Key           string
	Version       string
	Title         string
	Description   string
	Dialect       Dialect
	Query         string
	Parameters    []Parameter
	Columns       []Column
	Metadata      Metadata
	OutputFormats []Format
	Binder        Binder
}

// TemplateDescriptor is a serializable view of a Template for clients.
type TemplateDescriptor struct {
	Plugin        string      `json:"plugin"`
	Key           string      `json:"key"`
	Version       string      `json:"version"`
	Title         string      `json:"title"`
	Description   string      `json:"description"`
	Dialect       Dialect     `json:"dialect"`
	Query         string      `json:"query"`
	Parameters    []Parameter `json:"parameters"`
	Columns       []Column    `json:"columns"`
	Metadata      Metadata    `json:"metadata"`
	OutputFormats []Format    `json:"output_formats"`
	Slug          string      `json:"slug"`
}

// RunRequest describes a dataset execution request.
type RunRequest struct {
	Template   TemplateDescriptor
	Parameters map[string]any
	Scope      Scope
}

// RunResult is the materialized outcome of a dataset execution.
type RunResult struct {
	Schema      []Column         `json:"schema"`
	Rows        []map[string]any `json:"rows"`
	Metadata    map[string]any   `json:"metadata,omitempty"`
	GeneratedAt time.Time        `json:"generated_at"`
	Format      Format           `json:"format"`
}

// Runner executes a dataset with provided parameters and scope.
type Runner func(context.Context, RunRequest) (RunResult, error)

// Binder produces a Runner from an Environment.
type Binder func(Environment) (Runner, error)
