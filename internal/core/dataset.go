package core

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// DatasetDialect indicates the query language used by a dataset template.
type DatasetDialect string

const (
	// DatasetDialectSQL indicates templates expressed in SQL.
	DatasetDialectSQL DatasetDialect = "sql"
	// DatasetDialectDSL indicates templates expressed in the platform reporting DSL.
	DatasetDialectDSL DatasetDialect = "dsl"
)

// DatasetFormat enumerates supported output formats for datasets.
type DatasetFormat string

const (
	FormatJSON    DatasetFormat = "json"
	FormatCSV     DatasetFormat = "csv"
	FormatParquet DatasetFormat = "parquet"
	FormatPNG     DatasetFormat = "png"
	FormatHTML    DatasetFormat = "html"
)

// DatasetScope captures RBAC-derived filters applied to dataset execution.
type DatasetScope struct {
	Requestor   string   `json:"requestor"`
	Roles       []string `json:"roles,omitempty"`
	ProjectIDs  []string `json:"project_ids,omitempty"`
	ProtocolIDs []string `json:"protocol_ids,omitempty"`
}

// DatasetParameter declares an input parameter for a dataset template.
type DatasetParameter struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Required    bool     `json:"required"`
	Description string   `json:"description,omitempty"`
	Unit        string   `json:"unit,omitempty"`
	Enum        []string `json:"enum,omitempty"`
	Example     any      `json:"example,omitempty"`
	Default     any      `json:"default,omitempty"`
}

// DatasetColumn describes an output column for a dataset.
type DatasetColumn struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Unit        string `json:"unit,omitempty"`
	Description string `json:"description,omitempty"`
	Format      string `json:"format,omitempty"`
}

// DatasetTemplateMetadata stores additional descriptive attributes for a template.
type DatasetTemplateMetadata struct {
	Source          string            `json:"source,omitempty"`
	Documentation   string            `json:"documentation,omitempty"`
	RefreshInterval string            `json:"refresh_interval,omitempty"`
	Tags            []string          `json:"tags,omitempty"`
	Annotations     map[string]string `json:"annotations,omitempty"`
}

// DatasetBinder constructs a runner bound to runtime dependencies.
type DatasetBinder func(DatasetEnvironment) (DatasetRunner, error)

// DatasetRunner executes a dataset with validated parameters and scope.
type DatasetRunner func(context.Context, DatasetRunRequest) (DatasetRunResult, error)

// DatasetEnvironment provides runtime dependencies to binders.
type DatasetEnvironment struct {
	Store PersistentStore
	Now   func() time.Time
}

// DatasetRunRequest bundles invocation data for dataset runners.
type DatasetRunRequest struct {
	Template   DatasetTemplateDescriptor
	Parameters map[string]any
	Scope      DatasetScope
}

// DatasetRunResult represents canonical dataset output.
type DatasetRunResult struct {
	Schema      []DatasetColumn  `json:"schema"`
	Rows        []map[string]any `json:"rows"`
	Metadata    map[string]any   `json:"metadata,omitempty"`
	GeneratedAt time.Time        `json:"generated_at"`
	Format      DatasetFormat    `json:"format"`
}

// DatasetParameterError captures validation failures.
type DatasetParameterError struct {
	Name    string `json:"name"`
	Message string `json:"message"`
}

// DatasetTemplateDescriptor exposes immutable template metadata.
type DatasetTemplateDescriptor struct {
	Plugin        string                  `json:"plugin"`
	Key           string                  `json:"key"`
	Version       string                  `json:"version"`
	Title         string                  `json:"title"`
	Description   string                  `json:"description"`
	Dialect       DatasetDialect          `json:"dialect"`
	Query         string                  `json:"query"`
	Parameters    []DatasetParameter      `json:"parameters"`
	Columns       []DatasetColumn         `json:"columns"`
	Metadata      DatasetTemplateMetadata `json:"metadata"`
	OutputFormats []DatasetFormat         `json:"output_formats"`
	Slug          string                  `json:"slug"`
}

// DatasetTemplate captures plugin-provided dataset manifests and runtime binders.
type DatasetTemplate struct {
	Plugin        string
	Key           string
	Version       string
	Title         string
	Description   string
	Dialect       DatasetDialect
	Query         string
	Parameters    []DatasetParameter
	Columns       []DatasetColumn
	Metadata      DatasetTemplateMetadata
	OutputFormats []DatasetFormat
	Binder        DatasetBinder

	runner DatasetRunner
}

// Descriptor produces a descriptor snapshot for the template.
func (t DatasetTemplate) Descriptor() DatasetTemplateDescriptor {
	return DatasetTemplateDescriptor{
		Plugin:        t.Plugin,
		Key:           t.Key,
		Version:       t.Version,
		Title:         t.Title,
		Description:   t.Description,
		Dialect:       t.Dialect,
		Query:         t.Query,
		Parameters:    cloneParameters(t.Parameters),
		Columns:       cloneColumns(t.Columns),
		Metadata:      cloneMetadata(t.Metadata),
		OutputFormats: append([]DatasetFormat(nil), t.OutputFormats...),
		Slug:          t.slug(),
	}
}

func cloneParameters(in []DatasetParameter) []DatasetParameter {
	if len(in) == 0 {
		return nil
	}
	out := make([]DatasetParameter, len(in))
	copy(out, in)
	for i := range out {
		if len(out[i].Enum) > 0 {
			out[i].Enum = append([]string(nil), out[i].Enum...)
		}
	}
	return out
}

func cloneColumns(in []DatasetColumn) []DatasetColumn {
	if len(in) == 0 {
		return nil
	}
	out := make([]DatasetColumn, len(in))
	copy(out, in)
	return out
}

func cloneMetadata(in DatasetTemplateMetadata) DatasetTemplateMetadata {
	out := in
	if len(in.Tags) > 0 {
		out.Tags = append([]string(nil), in.Tags...)
	}
	if len(in.Annotations) > 0 {
		out.Annotations = make(map[string]string, len(in.Annotations))
		for k, v := range in.Annotations {
			out.Annotations[k] = v
		}
	}
	return out
}

// slug returns the canonical identifier for the template.
func (t DatasetTemplate) slug() string {
	if t.Plugin == "" {
		return fmt.Sprintf("%s@%s", t.Key, t.Version)
	}
	return fmt.Sprintf("%s/%s@%s", t.Plugin, t.Key, t.Version)
}

// validate ensures required fields are present and structurally sound.
func (t DatasetTemplate) validate() error {
	if strings.TrimSpace(t.Key) == "" {
		return errors.New("dataset template key required")
	}
	if strings.TrimSpace(t.Version) == "" {
		return errors.New("dataset template version required")
	}
	if strings.TrimSpace(t.Title) == "" {
		return errors.New("dataset template title required")
	}
	if t.Dialect != DatasetDialectSQL && t.Dialect != DatasetDialectDSL {
		return fmt.Errorf("unsupported dataset dialect %q", t.Dialect)
	}
	if strings.TrimSpace(t.Query) == "" {
		return errors.New("dataset template query required")
	}
	if len(t.Columns) == 0 {
		return errors.New("dataset template requires at least one column")
	}
	if len(t.OutputFormats) == 0 {
		return errors.New("dataset template must declare output formats")
	}
	if t.Binder == nil {
		return errors.New("dataset template binder required")
	}
	return nil
}

// bind attaches a runtime runner using the provided environment.
func (t *DatasetTemplate) bind(env DatasetEnvironment) error {
	if t == nil {
		return errors.New("dataset template nil")
	}
	if t.Binder == nil {
		return errors.New("dataset template binder missing")
	}
	runner, err := t.Binder(env)
	if err != nil {
		return err
	}
	if runner == nil {
		return errors.New("dataset template binder returned nil runner")
	}
	t.runner = runner
	return nil
}

// Run executes the dataset template using the bound runner after validating parameters.
func (t DatasetTemplate) Run(ctx context.Context, params map[string]any, scope DatasetScope, format DatasetFormat) (DatasetRunResult, []DatasetParameterError, error) {
	if t.runner == nil {
		return DatasetRunResult{}, nil, errors.New("dataset template not bound")
	}
	cleaned, errs := validateParameters(t.Parameters, params)
	if len(errs) > 0 {
		return DatasetRunResult{}, errs, nil
	}
	result, err := t.runner(ctx, DatasetRunRequest{
		Template:   t.Descriptor(),
		Parameters: cleaned,
		Scope:      scope,
	})
	if err != nil {
		return DatasetRunResult{}, nil, err
	}
	if len(result.Schema) == 0 {
		result.Schema = cloneColumns(t.Columns)
	}
	result.GeneratedAt = result.GeneratedAt.UTC()
	result.Format = format
	return result, nil, nil
}

// ValidateParameters validates parameters without executing the runner.
func (t DatasetTemplate) ValidateParameters(params map[string]any) (map[string]any, []DatasetParameterError) {
	return validateParameters(t.Parameters, params)
}

// SupportsFormat reports whether the template declares the requested format.
func (t DatasetTemplate) SupportsFormat(format DatasetFormat) bool {
	for _, candidate := range t.OutputFormats {
		if candidate == format {
			return true
		}
	}
	return false
}

func validateParameters(definitions []DatasetParameter, supplied map[string]any) (map[string]any, []DatasetParameterError) {
	cleaned := make(map[string]any)
	var errs []DatasetParameterError
	provided := make(map[string]struct{}, len(supplied))
	for k := range supplied {
		provided[strings.ToLower(k)] = struct{}{}
	}
	for _, param := range definitions {
		key := strings.ToLower(param.Name)
		val, ok := findParamValue(param.Name, supplied)
		if !ok {
			if param.Required {
				errs = append(errs, DatasetParameterError{Name: param.Name, Message: "required parameter missing"})
				continue
			}
			if param.Default != nil {
				cleaned[param.Name] = param.Default
			}
			continue
		}
		coerced, err := coerceParameter(param, val)
		if err != nil {
			errs = append(errs, DatasetParameterError{Name: param.Name, Message: err.Error()})
			continue
		}
		cleaned[param.Name] = coerced
		delete(provided, key)
	}
	for leftovers := range provided {
		errs = append(errs, DatasetParameterError{Name: leftovers, Message: "parameter not declared"})
	}
	if len(errs) > 0 {
		sort.Slice(errs, func(i, j int) bool { return errs[i].Name < errs[j].Name })
	}
	return cleaned, errs
}

func findParamValue(name string, supplied map[string]any) (any, bool) {
	if supplied == nil {
		return nil, false
	}
	if val, ok := supplied[name]; ok {
		return val, true
	}
	lower := strings.ToLower(name)
	for k, v := range supplied {
		if strings.ToLower(k) == lower {
			return v, true
		}
	}
	return nil, false
}

func coerceParameter(param DatasetParameter, raw any) (any, error) {
	if raw == nil {
		return nil, fmt.Errorf("parameter %s cannot be null", param.Name)
	}
	switch param.Type {
	case "string":
		switch v := raw.(type) {
		case string:
			if len(param.Enum) > 0 && !containsString(param.Enum, v) {
				return nil, enumError(param.Enum)
			}
			return v, nil
		case fmt.Stringer:
			val := v.String()
			if len(param.Enum) > 0 && !containsString(param.Enum, val) {
				return nil, enumError(param.Enum)
			}
			return val, nil
		default:
			return nil, fmt.Errorf("parameter %s expects string", param.Name)
		}
	case "integer":
		switch v := raw.(type) {
		case int:
			return v, nil
		case int64:
			return int(v), nil
		case float64:
			if v != float64(int(v)) {
				return nil, fmt.Errorf("parameter %s expects integer", param.Name)
			}
			return int(v), nil
		case string:
			parsed, err := strconv.Atoi(v)
			if err != nil {
				return nil, fmt.Errorf("parameter %s expects integer", param.Name)
			}
			return parsed, nil
		default:
			return nil, fmt.Errorf("parameter %s expects integer", param.Name)
		}
	case "number":
		switch v := raw.(type) {
		case float32:
			return float64(v), nil
		case float64:
			return v, nil
		case int:
			return float64(v), nil
		case int64:
			return float64(v), nil
		case string:
			parsed, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return nil, fmt.Errorf("parameter %s expects number", param.Name)
			}
			return parsed, nil
		default:
			return nil, fmt.Errorf("parameter %s expects number", param.Name)
		}
	case "boolean":
		switch v := raw.(type) {
		case bool:
			return v, nil
		case string:
			parsed, err := strconv.ParseBool(v)
			if err != nil {
				return nil, fmt.Errorf("parameter %s expects boolean", param.Name)
			}
			return parsed, nil
		default:
			return nil, fmt.Errorf("parameter %s expects boolean", param.Name)
		}
	case "timestamp":
		switch v := raw.(type) {
		case time.Time:
			return v.UTC(), nil
		case string:
			parsed, err := time.Parse(time.RFC3339, v)
			if err != nil {
				return nil, fmt.Errorf("parameter %s expects RFC3339 timestamp", param.Name)
			}
			return parsed.UTC(), nil
		default:
			return nil, fmt.Errorf("parameter %s expects timestamp", param.Name)
		}
	default:
		return nil, fmt.Errorf("unsupported parameter type %q", param.Type)
	}
}

func containsString(list []string, target string) bool {
	for _, candidate := range list {
		if candidate == target {
			return true
		}
	}
	return false
}

func enumError(options []string) error {
	if len(options) == 0 {
		return errors.New("invalid enumeration")
	}
	return fmt.Errorf("value must be one of: %s", strings.Join(options, ", "))
}

// DatasetTemplateCollection sorts descriptors for stable responses.
type DatasetTemplateCollection []DatasetTemplateDescriptor

func (c DatasetTemplateCollection) Len() int      { return len(c) }
func (c DatasetTemplateCollection) Swap(i, j int) { c[i], c[j] = c[j], c[i] }
func (c DatasetTemplateCollection) Less(i, j int) bool {
	if c[i].Plugin == c[j].Plugin {
		if c[i].Key == c[j].Key {
			return c[i].Version < c[j].Version
		}
		return c[i].Key < c[j].Key
	}
	return c[i].Plugin < c[j].Plugin
}
