// Package datasetapi defines dataset templating primitives, descriptors,
// execution types, and plugin binding contracts.
package datasetapi

import (
	"context"
	"time"
)

// Dialect identifies the query language a template uses.
type Dialect string

// DialectProvider defines the interface for providing dialect constants.
type DialectProvider interface {
	// SQL returns the standard SQL dialect identifier.
	SQL() Dialect
	// DSL returns the custom domain-specific language dialect identifier.
	DSL() Dialect
}

// DefaultDialectProvider provides the default dialect implementation.
type DefaultDialectProvider struct{}

// SQL returns the standard SQL dialect.
func (DefaultDialectProvider) SQL() Dialect {
	return "sql"
}

// DSL returns the custom domain-specific language dialect.
func (DefaultDialectProvider) DSL() Dialect {
	return "dsl"
}

// GetDialectProvider returns the default dialect provider instance.
func GetDialectProvider() DialectProvider {
	return DefaultDialectProvider{}
}

// Format identifies an output encoding for a dataset result.
type Format string

// FormatProvider defines the interface for providing format constants.
type FormatProvider interface {
	// JSON returns the JSON document output format.
	JSON() Format
	// CSV returns the comma separated values output format.
	CSV() Format
	// Parquet returns the columnar Parquet output format.
	Parquet() Format
	// PNG returns the PNG image output format.
	PNG() Format
	// HTML returns the rendered HTML output format.
	HTML() Format
}

// DefaultFormatProvider provides the default format implementation.
type DefaultFormatProvider struct{}

// JSON returns the JSON document output format.
func (DefaultFormatProvider) JSON() Format {
	return "json"
}

// CSV returns the comma separated values output format.
func (DefaultFormatProvider) CSV() Format {
	return "csv"
}

// Parquet returns the columnar Parquet output format.
func (DefaultFormatProvider) Parquet() Format {
	return "parquet"
}

// PNG returns the PNG image output format.
func (DefaultFormatProvider) PNG() Format {
	return "png"
}

// HTML returns the rendered HTML output format.
func (DefaultFormatProvider) HTML() Format {
	return "html"
}

// GetFormatProvider returns the default format provider instance.
func GetFormatProvider() FormatProvider {
	return DefaultFormatProvider{}
}

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
	Source           string            `json:"source,omitempty"`
	Documentation    string            `json:"documentation,omitempty"`
	RefreshInterval  string            `json:"refresh_interval,omitempty"`
	Tags             []string          `json:"tags,omitempty"`
	Annotations      map[string]string `json:"annotations,omitempty"`
	EntityModelMajor *int              `json:"entity_model_major,omitempty"`
}

// Environment bundles dependencies needed for binding template runners.
type Environment struct {
	Store PersistentStore
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

// TemplateDescriptor is a serialization-focused projection of a dataset template.
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

// ParameterError captures validation failures reported during parameter coercion
// and validation when invoking dataset templates.
type ParameterError struct {
	Name    string `json:"name"`
	Message string `json:"message"`
}

// EntityRef identifies a domain entity related to a dataset resource.
type EntityRef struct {
	Entity string `json:"entity"`
	ID     string `json:"id"`
}

// Row is a serialization-friendly dataset row representation.
type Row map[string]any

// RunResult is the materialized outcome of a dataset execution.
type RunResult struct {
	Schema      []Column       `json:"schema"`
	Rows        []Row          `json:"rows"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	GeneratedAt time.Time      `json:"generated_at"`
	Format      Format         `json:"format"`
}

// Runner executes a dataset with provided parameters and scope.
type Runner func(context.Context, RunRequest) (RunResult, error)

// Binder produces a Runner from an Environment.
type Binder func(Environment) (Runner, error)

// TemplateRuntime exposes host-managed capabilities for executing dataset templates.
// Implementations are provided by the colonycore service layer and adapt plugin-
// supplied templates to runtime dependencies and validation semantics.
type TemplateRuntime interface {
	Descriptor() TemplateDescriptor
	SupportsFormat(format Format) bool
	ValidateParameters(params map[string]any) (map[string]any, []ParameterError)
	Run(ctx context.Context, params map[string]any, scope Scope, format Format) (RunResult, []ParameterError, error)
}
