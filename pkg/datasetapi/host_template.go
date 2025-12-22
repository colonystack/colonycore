package datasetapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// HostTemplate encapsulates a plugin-provided Template together with
// host-specific runtime state (bound runner, plugin name, validation helpers).
type HostTemplate struct {
	plugin  string
	tpl     Template
	runtime Runner
}

// NewHostTemplate constructs a HostTemplate for the given plugin/template pair
// after performing structural validation. The returned template has no bound
// runner; callers must invoke Bind with the runtime environment before running.
func NewHostTemplate(plugin string, tpl Template) (HostTemplate, error) {
	if err := validateTemplate(tpl); err != nil {
		return HostTemplate{}, err
	}
	return HostTemplate{plugin: strings.TrimSpace(plugin), tpl: cloneTemplate(tpl)}, nil
}

// Plugin returns the plugin identifier associated with the template.
func (h HostTemplate) Plugin() string { return h.plugin }

// Template returns a defensive copy of the underlying template metadata.
func (h HostTemplate) Template() Template { return cloneTemplate(h.tpl) }

// Descriptor produces a TemplateDescriptor snapshot including plugin metadata
// and computed slug.
func (h HostTemplate) Descriptor() TemplateDescriptor {
	return TemplateDescriptor{
		Plugin:        h.plugin,
		Key:           h.tpl.Key,
		Version:       h.tpl.Version,
		Title:         h.tpl.Title,
		Description:   h.tpl.Description,
		Dialect:       h.tpl.Dialect,
		Query:         h.tpl.Query,
		Parameters:    cloneParameters(h.tpl.Parameters),
		Columns:       cloneColumns(h.tpl.Columns),
		Metadata:      cloneMetadata(h.tpl.Metadata),
		OutputFormats: cloneFormats(h.tpl.OutputFormats),
		Slug:          slugFor(h.plugin, h.tpl.Key, h.tpl.Version),
	}
}

// Slug returns the canonical identifier for the template (plugin/key@version).
func (h HostTemplate) Slug() string {
	return slugFor(h.plugin, h.tpl.Key, h.tpl.Version)
}

// SupportsFormat reports whether the template declares the requested format.
func (h HostTemplate) SupportsFormat(format Format) bool {
	for _, candidate := range h.tpl.OutputFormats {
		if candidate == format {
			return true
		}
	}
	return false
}

// ValidateParameters validates supplied parameters against the template
// definition, returning normalized values plus any validation errors.
func (h HostTemplate) ValidateParameters(params map[string]any) (map[string]any, []ParameterError) {
	return validateParameters(h.tpl.Parameters, params)
}

// Bind attaches a runtime runner to the host template using the provided
// environment. Binder implementations originate from plugin authors.
func (h *HostTemplate) Bind(env Environment) error {
	if h == nil {
		return errors.New("datasetapi: host template nil")
	}
	if h.tpl.Binder == nil {
		return errors.New("datasetapi: template binder missing")
	}
	runner, err := h.tpl.Binder(env)
	if err != nil {
		return err
	}
	if runner == nil {
		return errors.New("datasetapi: template binder returned nil runner")
	}
	h.runtime = runner
	return nil
}

// Run executes the bound template after validating parameters. The template
// must be bound via Bind before calling Run.
func (h HostTemplate) Run(ctx context.Context, params map[string]any, scope Scope, format Format) (RunResult, []ParameterError, error) {
	if h.runtime == nil {
		return RunResult{}, nil, errors.New("datasetapi: template not bound")
	}
	cleaned, errs := validateParameters(h.tpl.Parameters, params)
	if len(errs) > 0 {
		return RunResult{}, errs, nil
	}
	result, err := h.runtime(ctx, RunRequest{
		Template:   h.Descriptor(),
		Parameters: cleaned,
		Scope:      cloneScope(scope),
	})
	if err != nil {
		return RunResult{}, nil, err
	}
	if len(result.Schema) == 0 {
		result.Schema = cloneColumns(h.tpl.Columns)
	}
	result.GeneratedAt = result.GeneratedAt.UTC()
	result.Format = format
	return result, nil, nil
}

// Ensure HostTemplate satisfies TemplateRuntime.
var _ TemplateRuntime = (*HostTemplate)(nil)

// SortTemplateDescriptors sorts the slice in-place using plugin/key/version for
// deterministic ordering.
func SortTemplateDescriptors(descriptors []TemplateDescriptor) {
	if len(descriptors) < 2 {
		return
	}
	sort.Slice(descriptors, func(i, j int) bool {
		a := descriptors[i]
		b := descriptors[j]
		if a.Plugin == b.Plugin {
			if a.Key == b.Key {
				return a.Version < b.Version
			}
			return a.Key < b.Key
		}
		return a.Plugin < b.Plugin
	})
}

func validateTemplate(tpl Template) error {
	if strings.TrimSpace(tpl.Key) == "" {
		return errors.New("datasetapi: dataset template key required")
	}
	if strings.TrimSpace(tpl.Version) == "" {
		return errors.New("datasetapi: dataset template version required")
	}
	if strings.TrimSpace(tpl.Title) == "" {
		return errors.New("datasetapi: dataset template title required")
	}
	if strings.TrimSpace(tpl.Query) == "" {
		return errors.New("datasetapi: dataset template query required")
	}
	if len(tpl.Columns) == 0 {
		return errors.New("datasetapi: dataset template requires at least one column")
	}
	if len(tpl.OutputFormats) == 0 {
		return errors.New("datasetapi: dataset template must declare output formats")
	}
	if tpl.Binder == nil {
		return errors.New("datasetapi: dataset template binder required")
	}
	dialectProvider := GetDialectProvider()
	if tpl.Dialect != dialectProvider.SQL() && tpl.Dialect != dialectProvider.DSL() {
		return fmt.Errorf("datasetapi: unsupported dataset dialect %q", tpl.Dialect)
	}
	return nil
}

func validateParameters(definitions []Parameter, supplied map[string]any) (map[string]any, []ParameterError) {
	cleaned := make(map[string]any)
	var errs []ParameterError
	provided := make(map[string]struct{}, len(supplied))
	for k := range supplied {
		provided[strings.ToLower(k)] = struct{}{}
	}
	for _, param := range definitions {
		key := strings.ToLower(param.Name)
		val, ok := findParamValue(param.Name, supplied)
		if !ok {
			if param.Required {
				errs = append(errs, ParameterError{Name: param.Name, Message: "required parameter missing"})
				continue
			}
			if len(param.Default) > 0 {
				coerced, err := coerceDefaultParameter(param)
				if err != nil {
					errs = append(errs, ParameterError{Name: param.Name, Message: err.Error()})
					continue
				}
				cleaned[param.Name] = coerced
			}
			continue
		}
		coerced, err := coerceParameter(param, val)
		if err != nil {
			errs = append(errs, ParameterError{Name: param.Name, Message: err.Error()})
			continue
		}
		cleaned[param.Name] = coerced
		delete(provided, key)
	}
	for leftovers := range provided {
		errs = append(errs, ParameterError{Name: leftovers, Message: "parameter not declared"})
	}
	if len(errs) > 0 {
		sort.Slice(errs, func(i, j int) bool { return errs[i].Name < errs[j].Name })
	}
	return cleaned, errs
}

func coerceDefaultParameter(param Parameter) (any, error) {
	var raw any
	if err := json.Unmarshal(param.Default, &raw); err != nil {
		return nil, fmt.Errorf("parameter %s default is invalid JSON: %w", param.Name, err)
	}
	return coerceParameter(param, raw)
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

func coerceParameter(param Parameter, raw any) (any, error) {
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

func slugFor(plugin, key, version string) string {
	keyPart := strings.TrimSpace(key)
	versionPart := strings.TrimSpace(version)
	if plugin = strings.TrimSpace(plugin); plugin == "" {
		return fmt.Sprintf("%s@%s", keyPart, versionPart)
	}
	return fmt.Sprintf("%s/%s@%s", plugin, keyPart, versionPart)
}

func cloneTemplate(t Template) Template {
	cloned := t
	cloned.Parameters = cloneParameters(t.Parameters)
	cloned.Columns = cloneColumns(t.Columns)
	cloned.Metadata = cloneMetadata(t.Metadata)
	cloned.OutputFormats = cloneFormats(t.OutputFormats)
	return cloned
}

func cloneParameters(params []Parameter) []Parameter {
	if len(params) == 0 {
		return nil
	}
	cloned := make([]Parameter, len(params))
	copy(cloned, params)
	for i := range cloned {
		if len(cloned[i].Example) > 0 {
			cloned[i].Example = append([]byte(nil), cloned[i].Example...)
		}
		if len(cloned[i].Default) > 0 {
			cloned[i].Default = append([]byte(nil), cloned[i].Default...)
		}
		if len(cloned[i].Enum) > 0 {
			cloned[i].Enum = append([]string(nil), cloned[i].Enum...)
		}
	}
	return cloned
}

func cloneColumns(columns []Column) []Column {
	if len(columns) == 0 {
		return nil
	}
	cloned := make([]Column, len(columns))
	copy(cloned, columns)
	return cloned
}

func cloneFormats(formats []Format) []Format {
	if len(formats) == 0 {
		return nil
	}
	cloned := make([]Format, len(formats))
	copy(cloned, formats)
	return cloned
}

func cloneMetadata(metadata Metadata) Metadata {
	cloned := metadata
	if len(metadata.Tags) > 0 {
		cloned.Tags = append([]string(nil), metadata.Tags...)
	}
	if metadata.EntityModelMajor != nil {
		major := *metadata.EntityModelMajor
		cloned.EntityModelMajor = &major
	}
	if len(metadata.Annotations) > 0 {
		cloned.Annotations = make(map[string]string, len(metadata.Annotations))
		for k, v := range metadata.Annotations {
			cloned.Annotations[k] = v
		}
	}
	return cloned
}

func cloneScope(scope Scope) Scope {
	cloned := Scope{Requestor: scope.Requestor}
	if len(scope.Roles) > 0 {
		cloned.Roles = append([]string(nil), scope.Roles...)
	}
	if len(scope.ProjectIDs) > 0 {
		cloned.ProjectIDs = append([]string(nil), scope.ProjectIDs...)
	}
	if len(scope.ProtocolIDs) > 0 {
		cloned.ProtocolIDs = append([]string(nil), scope.ProtocolIDs...)
	}
	return cloned
}

// helper functions to satisfy linting for unused conversions removed.
