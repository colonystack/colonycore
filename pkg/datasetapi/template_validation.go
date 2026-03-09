package datasetapi

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

const (
	parameterTypeString    = "string"
	parameterTypeInteger   = "integer"
	parameterTypeNumber    = "number"
	parameterTypeBoolean   = "boolean"
	parameterTypeTimestamp = "timestamp"
)

var (
	templateNamePattern       = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)
	sqlMutationPattern        = regexp.MustCompile(`(?im)(^|[^[:alnum:]_])(insert|update|delete|drop|alter|create|truncate|replace|merge)\b`)
	dslReportPattern          = regexp.MustCompile(`(?i)\breport\b`)
	dslSelectPattern          = regexp.MustCompile(`(?i)\bselect\b`)
	allowedParameterTypes     = map[string]struct{}{parameterTypeString: {}, parameterTypeInteger: {}, parameterTypeNumber: {}, parameterTypeBoolean: {}, parameterTypeTimestamp: {}}
	allowedTemplateFormatsSet = func() map[Format]struct{} {
		formatProvider := GetFormatProvider()
		return map[Format]struct{}{
			formatProvider.JSON():    {},
			formatProvider.CSV():     {},
			formatProvider.Parquet(): {},
			formatProvider.PNG():     {},
			formatProvider.HTML():    {},
		}
	}()
)

// TemplateValidationIssue captures one validation rule violation.
type TemplateValidationIssue struct {
	Field   string
	Message string
}

// TemplateValidationError aggregates one or more validation issues.
type TemplateValidationError struct {
	Issues []TemplateValidationIssue
}

// Error returns a human-readable summary of all template validation issues.
func (e *TemplateValidationError) Error() string {
	if e == nil || len(e.Issues) == 0 {
		return "datasetapi: dataset template validation failed"
	}
	parts := make([]string, 0, len(e.Issues))
	for _, issue := range e.Issues {
		parts = append(parts, issue.Field+": "+issue.Message)
	}
	return "datasetapi: dataset template validation failed: " + strings.Join(parts, "; ")
}

// ValidateTemplate validates a runtime template definition and returns a detailed
// error when the template violates dataset specification invariants.
func ValidateTemplate(template Template) error {
	issues := validateTemplateSpec(templateValidationInput{
		Key:           template.Key,
		Version:       template.Version,
		Title:         template.Title,
		Dialect:       template.Dialect,
		Query:         template.Query,
		Parameters:    template.Parameters,
		Columns:       template.Columns,
		Metadata:      template.Metadata,
		OutputFormats: template.OutputFormats,
		Binder:        template.Binder,
		requireBinder: true,
	})
	return issuesAsError(issues)
}

// ValidateTemplateDescriptor validates a serialization-focused dataset template
// descriptor. This path is used by static tooling where no runtime binder exists.
func ValidateTemplateDescriptor(descriptor TemplateDescriptor) error {
	issues := validateTemplateSpec(templateValidationInput{
		Key:           descriptor.Key,
		Version:       descriptor.Version,
		Title:         descriptor.Title,
		Dialect:       descriptor.Dialect,
		Query:         descriptor.Query,
		Parameters:    descriptor.Parameters,
		Columns:       descriptor.Columns,
		Metadata:      descriptor.Metadata,
		OutputFormats: descriptor.OutputFormats,
		Plugin:        descriptor.Plugin,
		Slug:          descriptor.Slug,
		requireBinder: false,
	})
	return issuesAsError(issues)
}

type templateValidationInput struct {
	Key           string
	Version       string
	Title         string
	Dialect       Dialect
	Query         string
	Parameters    []Parameter
	Columns       []Column
	Metadata      Metadata
	OutputFormats []Format
	Binder        Binder
	Plugin        string
	Slug          string
	requireBinder bool
}

func issuesAsError(issues []TemplateValidationIssue) error {
	if len(issues) == 0 {
		return nil
	}
	sort.Slice(issues, func(i, j int) bool {
		if issues[i].Field == issues[j].Field {
			return issues[i].Message < issues[j].Message
		}
		return issues[i].Field < issues[j].Field
	})
	return &TemplateValidationError{Issues: issues}
}

func validateTemplateSpec(input templateValidationInput) []TemplateValidationIssue {
	issues := make([]TemplateValidationIssue, 0, 8)
	addIssue := func(field, message string) {
		issues = append(issues, TemplateValidationIssue{Field: field, Message: message})
	}

	key := strings.TrimSpace(input.Key)
	if key == "" {
		addIssue("key", "required")
	}
	if strings.TrimSpace(input.Version) == "" {
		addIssue("version", "required")
	}
	if strings.TrimSpace(input.Title) == "" {
		addIssue("title", "required")
	}
	if strings.TrimSpace(input.Query) == "" {
		addIssue("query", "required")
	}

	dialectProvider := GetDialectProvider()
	switch input.Dialect {
	case dialectProvider.SQL():
		validateSQLSemantics(input.Query, addIssue)
	case dialectProvider.DSL():
		validateDSLSemantics(input.Query, addIssue)
	default:
		if strings.TrimSpace(string(input.Dialect)) == "" {
			addIssue("dialect", "required")
		} else {
			addIssue("dialect", fmt.Sprintf("unsupported dataset dialect %q", input.Dialect))
		}
	}

	validateParameterSpec(input.Parameters, addIssue)
	validateColumnSpec(input.Columns, addIssue)
	validateOutputFormats(input.OutputFormats, addIssue)
	validateMetadata(input.Metadata, addIssue)

	if input.requireBinder && input.Binder == nil {
		addIssue("binder", "required")
	}

	if strings.TrimSpace(input.Slug) != "" {
		expected := slugFor(input.Plugin, input.Key, input.Version)
		if input.Slug != expected {
			addIssue("slug", fmt.Sprintf("must equal %q", expected))
		}
	}

	return issues
}

func validateSQLSemantics(query string, addIssue func(field, message string)) {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return
	}
	upper := strings.ToUpper(trimmed)
	if !strings.HasPrefix(upper, "SELECT") && !strings.HasPrefix(upper, "WITH") {
		addIssue("query", "sql dialect requires a read-only SELECT/WITH statement")
	}
	normalized := stripSQLComments(stripSQLStringLiterals(trimmed))
	if sqlMutationPattern.MatchString(normalized) {
		addIssue("query", "sql dialect forbids mutating statements")
	}
}

func validateDSLSemantics(query string, addIssue func(field, message string)) {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return
	}
	if !dslReportPattern.MatchString(trimmed) {
		addIssue("query", "dsl dialect requires a REPORT declaration")
	}
	if !dslSelectPattern.MatchString(trimmed) {
		addIssue("query", "dsl dialect requires a SELECT clause")
	}
}

func validateParameterSpec(parameters []Parameter, addIssue func(field, message string)) {
	seen := make(map[string]struct{}, len(parameters))
	for i, param := range parameters {
		field := fmt.Sprintf("parameters[%d]", i)
		name := strings.TrimSpace(param.Name)
		if name == "" {
			addIssue(field+".name", "required")
		} else {
			if !templateNamePattern.MatchString(name) {
				addIssue(field+".name", "must match [A-Za-z][A-Za-z0-9_]*")
			}
			key := strings.ToLower(name)
			if _, exists := seen[key]; exists {
				addIssue(field+".name", fmt.Sprintf("duplicate parameter name %q", name))
			}
			seen[key] = struct{}{}
		}

		paramType := strings.TrimSpace(param.Type)
		if paramType == "" {
			addIssue(field+".type", "required")
		} else if _, ok := allowedParameterTypes[paramType]; !ok {
			addIssue(field+".type", fmt.Sprintf("unsupported parameter type %q", paramType))
		}

		if len(param.Enum) > 0 {
			if paramType != parameterTypeString {
				addIssue(field+".enum", "only supported for string parameters")
			}
			enumSeen := make(map[string]struct{}, len(param.Enum))
			for idx, entry := range param.Enum {
				entryField := fmt.Sprintf("%s.enum[%d]", field, idx)
				if strings.TrimSpace(entry) == "" {
					addIssue(entryField, "must not be empty")
					continue
				}
				if _, exists := enumSeen[entry]; exists {
					addIssue(entryField, fmt.Sprintf("duplicate enum value %q", entry))
				}
				enumSeen[entry] = struct{}{}
			}
		}

		if len(param.Example) > 0 {
			if !json.Valid(param.Example) {
				addIssue(field+".example", "must contain valid JSON")
			}
		}

		if len(param.Default) > 0 {
			if !json.Valid(param.Default) {
				addIssue(field+".default", "must contain valid JSON")
			} else if _, ok := allowedParameterTypes[paramType]; ok {
				if _, err := coerceDefaultParameter(param); err != nil {
					addIssue(field+".default", err.Error())
				}
			}
		}
	}
}

func validateColumnSpec(columns []Column, addIssue func(field, message string)) {
	if len(columns) == 0 {
		addIssue("columns", "requires at least one column")
		return
	}

	seen := make(map[string]struct{}, len(columns))
	for i, column := range columns {
		field := fmt.Sprintf("columns[%d]", i)
		name := strings.TrimSpace(column.Name)
		if name == "" {
			addIssue(field+".name", "required")
		} else {
			if !templateNamePattern.MatchString(name) {
				addIssue(field+".name", "must match [A-Za-z][A-Za-z0-9_]*")
			}
			key := strings.ToLower(name)
			if _, exists := seen[key]; exists {
				addIssue(field+".name", fmt.Sprintf("duplicate column name %q", name))
			}
			seen[key] = struct{}{}
		}
		if strings.TrimSpace(column.Type) == "" {
			addIssue(field+".type", "required")
		}
	}
}

func validateOutputFormats(formats []Format, addIssue func(field, message string)) {
	if len(formats) == 0 {
		addIssue("output_formats", "must declare at least one format")
		return
	}

	seen := make(map[Format]struct{}, len(formats))
	for i, format := range formats {
		field := fmt.Sprintf("output_formats[%d]", i)
		if strings.TrimSpace(string(format)) == "" {
			addIssue(field, "required")
			continue
		}
		if _, ok := allowedTemplateFormatsSet[format]; !ok {
			addIssue(field, fmt.Sprintf("unsupported format %q", format))
		}
		if _, exists := seen[format]; exists {
			addIssue(field, fmt.Sprintf("duplicate format %q", format))
		}
		seen[format] = struct{}{}
	}
}

func validateMetadata(metadata Metadata, addIssue func(field, message string)) {
	if metadata.EntityModelMajor != nil && *metadata.EntityModelMajor <= 0 {
		addIssue("metadata.entity_model_major", "must be greater than zero")
	}
}

// stripSQLStringLiterals masks single/double quoted literals to reduce false
// positives when scanning for SQL mutation keywords.
func stripSQLStringLiterals(query string) string {
	var b strings.Builder
	b.Grow(len(query))

	inSingle := false
	inDouble := false

	for i := 0; i < len(query); i++ {
		ch := query[i]
		switch {
		case inSingle:
			if ch == '\'' {
				if i+1 < len(query) && query[i+1] == '\'' {
					i++
				} else {
					inSingle = false
				}
			}
			b.WriteByte(' ')
		case inDouble:
			if ch == '"' {
				if i+1 < len(query) && query[i+1] == '"' {
					i++
				} else {
					inDouble = false
				}
			}
			b.WriteByte(' ')
		default:
			switch ch {
			case '\'':
				inSingle = true
				b.WriteByte(' ')
			case '"':
				inDouble = true
				b.WriteByte(' ')
			default:
				b.WriteByte(ch)
			}
		}
	}
	return b.String()
}

func stripSQLComments(query string) string {
	var b strings.Builder
	b.Grow(len(query))

	inLine := false
	inBlock := false

	for i := 0; i < len(query); i++ {
		ch := query[i]
		switch {
		case inLine:
			if ch == '\n' {
				inLine = false
				b.WriteByte('\n')
			} else {
				b.WriteByte(' ')
			}
		case inBlock:
			if ch == '*' && i+1 < len(query) && query[i+1] == '/' {
				inBlock = false
				b.WriteString("  ")
				i++
				continue
			}
			if ch == '\n' {
				b.WriteByte('\n')
			} else {
				b.WriteByte(' ')
			}
		default:
			if ch == '-' && i+1 < len(query) && query[i+1] == '-' {
				inLine = true
				b.WriteString("  ")
				i++
				continue
			}
			if ch == '/' && i+1 < len(query) && query[i+1] == '*' {
				inBlock = true
				b.WriteString("  ")
				i++
				continue
			}
			b.WriteByte(ch)
		}
	}
	return b.String()
}
