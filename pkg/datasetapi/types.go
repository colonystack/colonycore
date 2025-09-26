package datasetapi

import (
	"context"
	"time"

	"colonycore/pkg/domain"
)

type Dialect string

const (
	DialectSQL Dialect = "sql"
	DialectDSL Dialect = "dsl"
)

type Format string

const (
	FormatJSON    Format = "json"
	FormatCSV     Format = "csv"
	FormatParquet Format = "parquet"
	FormatPNG     Format = "png"
	FormatHTML    Format = "html"
)

type Scope struct {
	Requestor   string   `json:"requestor"`
	Roles       []string `json:"roles,omitempty"`
	ProjectIDs  []string `json:"project_ids,omitempty"`
	ProtocolIDs []string `json:"protocol_ids,omitempty"`
}

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

type Column struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Unit        string `json:"unit,omitempty"`
	Description string `json:"description,omitempty"`
	Format      string `json:"format,omitempty"`
}

type Metadata struct {
	Source          string            `json:"source,omitempty"`
	Documentation   string            `json:"documentation,omitempty"`
	RefreshInterval string            `json:"refresh_interval,omitempty"`
	Tags            []string          `json:"tags,omitempty"`
	Annotations     map[string]string `json:"annotations,omitempty"`
}

type Environment struct {
	Store domain.PersistentStore
	Now   func() time.Time
}

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

type RunRequest struct {
	Template   TemplateDescriptor
	Parameters map[string]any
	Scope      Scope
}

type RunResult struct {
	Schema      []Column         `json:"schema"`
	Rows        []map[string]any `json:"rows"`
	Metadata    map[string]any   `json:"metadata,omitempty"`
	GeneratedAt time.Time        `json:"generated_at"`
	Format      Format           `json:"format"`
}

type Runner func(context.Context, RunRequest) (RunResult, error)

type Binder func(Environment) (Runner, error)
