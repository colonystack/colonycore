package datasetapi

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

type stringerValue struct{ value string }

func (s stringerValue) String() string { return s.value }

func TestNewHostTemplateAndRuntime(t *testing.T) {
	now := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	dialectProvider := GetDialectProvider()
	formatProvider := GetFormatProvider()
	tpl := Template{
		Key:         "demo",
		Version:     "1.0.0",
		Title:       "Demo",
		Description: "demo",
		Dialect:     dialectProvider.SQL(),
		Query:       "SELECT 1",
		Parameters: []Parameter{{
			Name:        "limit",
			Type:        "integer",
			Required:    true,
			Description: "limit results",
		}},
		Columns: []Column{{
			Name:        "value",
			Type:        "integer",
			Description: "value column",
		}},
		Metadata: Metadata{
			Source:          "tests",
			Documentation:   "docs",
			RefreshInterval: "PT1H",
			Tags:            []string{"tag"},
			Annotations:     map[string]string{"k": "v"},
		},
		OutputFormats: []Format{formatProvider.JSON(), formatProvider.CSV()},
	}

	tpl.Binder = func(env Environment) (Runner, error) {
		if env.Now == nil {
			t.Fatalf("expected now function")
		}
		return func(_ context.Context, req RunRequest) (RunResult, error) {
			if req.Template.Key != "demo" {
				t.Fatalf("unexpected template key: %s", req.Template.Key)
			}
			if req.Scope.Requestor != "analyst" {
				t.Fatalf("unexpected requestor: %s", req.Scope.Requestor)
			}
			return RunResult{
				Schema: []Column{{Name: "value", Type: "integer"}},
				Rows:   []Row{{"value": 7}},
				Metadata: map[string]any{
					"note": "ok",
				},
				GeneratedAt: env.Now(),
				Format:      formatProvider.CSV(),
			}, nil
		}, nil
	}

	host, err := NewHostTemplate("frog", tpl)
	if err != nil {
		t.Fatalf("NewHostTemplate: %v", err)
	}
	if host.Slug() != "frog/demo@1.0.0" {
		t.Fatalf("unexpected slug: %s", host.Slug())
	}
	if !host.SupportsFormat(formatProvider.JSON()) || host.SupportsFormat(formatProvider.PNG()) {
		t.Fatalf("unexpected format support")
	}

	env := Environment{Now: func() time.Time { return now }}
	if err := host.Bind(env); err != nil {
		t.Fatalf("Bind: %v", err)
	}

	params, errs := host.ValidateParameters(map[string]any{"limit": 5})
	if len(errs) != 0 {
		t.Fatalf("unexpected validation errors: %+v", errs)
	}
	if params["limit"].(int) != 5 {
		t.Fatalf("expected cleaned parameters to retain value")
	}

	scope := Scope{Requestor: "analyst", ProjectIDs: []string{"project"}}
	result, paramErrs, err := host.Run(context.Background(), map[string]any{"limit": 5}, scope, formatProvider.JSON())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(paramErrs) != 0 {
		t.Fatalf("unexpected parameter errors: %+v", paramErrs)
	}
	if result.Format != formatProvider.JSON() {
		t.Fatalf("expected JSON format, got %s", result.Format)
	}
	if len(result.Rows) != 1 || result.Rows[0]["value"].(int) != 7 {
		t.Fatalf("unexpected rows: %+v", result.Rows)
	}
	if result.GeneratedAt != now {
		t.Fatalf("expected generated timestamp %v, got %v", now, result.GeneratedAt)
	}

	helper := host
	// Ensure TemplateRuntime interface is satisfied by rerunning via helper copy.
	if _, _, err := helper.Run(context.Background(), map[string]any{"limit": 5}, scope, formatProvider.JSON()); err != nil {
		t.Fatalf("helper run: %v", err)
	}
}

func TestNewHostTemplateValidationErrors(t *testing.T) {
	dialectProvider := GetDialectProvider()
	formatProvider := GetFormatProvider()

	cases := []struct {
		name string
		mut  func(*Template)
	}{
		{"missing key", func(t *Template) { t.Key = "" }},
		{"missing version", func(t *Template) { t.Version = "" }},
		{"missing title", func(t *Template) { t.Title = "" }},
		{"missing query", func(t *Template) { t.Query = "" }},
		{"missing columns", func(t *Template) { t.Columns = nil }},
		{"missing formats", func(t *Template) { t.OutputFormats = nil }},
		{"missing binder", func(t *Template) { t.Binder = nil }},
		{"unsupported dialect", func(t *Template) { t.Dialect = Dialect("graphql") }},
	}

	base := Template{
		Key:           "k",
		Version:       "1",
		Title:         "t",
		Dialect:       dialectProvider.SQL(),
		Query:         "select 1",
		Columns:       []Column{{Name: "c", Type: "string"}},
		OutputFormats: []Format{formatProvider.JSON()},
		Binder:        func(Environment) (Runner, error) { return nil, nil },
	}

	for _, tc := range cases {
		tpl := base
		tc.mut(&tpl)
		if _, err := NewHostTemplate("frog", tpl); err == nil {
			t.Fatalf("expected validation failure for %s", tc.name)
		}
	}
}

func TestHostTemplateBindErrors(t *testing.T) {
	dialectProvider := GetDialectProvider()
	formatProvider := GetFormatProvider()

	tpl := Template{
		Key:           "k",
		Version:       "1",
		Title:         "t",
		Dialect:       dialectProvider.SQL(),
		Query:         "select 1",
		Columns:       []Column{{Name: "c", Type: "string"}},
		OutputFormats: []Format{formatProvider.JSON()},
		Binder:        func(Environment) (Runner, error) { return nil, nil },
	}
	host, err := NewHostTemplate("plugin", tpl)
	if err != nil {
		t.Fatalf("NewHostTemplate: %v", err)
	}
	if err := host.Bind(Environment{}); err == nil {
		t.Fatalf("expected bind error for nil runner")
	}

	tpl.Binder = func(Environment) (Runner, error) { return nil, errors.New("fail") }
	host, err = NewHostTemplate("plugin", tpl)
	if err != nil {
		t.Fatalf("NewHostTemplate: %v", err)
	}
	if err := host.Bind(Environment{}); err == nil || !strings.Contains(err.Error(), "fail") {
		t.Fatalf("expected binder failure, got %v", err)
	}
}

func TestValidateParametersErrors(t *testing.T) {
	dialectProvider := GetDialectProvider()
	formatProvider := GetFormatProvider()

	tpl := Template{
		Key:           "k",
		Version:       "1",
		Title:         "t",
		Dialect:       dialectProvider.SQL(),
		Query:         "select 1",
		Columns:       []Column{{Name: "c", Type: "string"}},
		OutputFormats: []Format{formatProvider.JSON()},
		Parameters: []Parameter{
			{Name: "stage", Type: "string", Enum: []string{"adult", "larva"}},
			{Name: "limit", Type: "integer", Required: true},
			{Name: "flag", Type: "boolean"},
			{Name: "when", Type: "timestamp"},
			{Name: "mode", Type: "string", Default: "auto"},
			{Name: "alias", Type: "string"},
		},
		Binder: func(Environment) (Runner, error) {
			return func(context.Context, RunRequest) (RunResult, error) { return RunResult{}, nil }, nil
		},
	}
	host, err := NewHostTemplate("plugin", tpl)
	if err != nil {
		t.Fatalf("NewHostTemplate: %v", err)
	}
	if host.SupportsFormat(formatProvider.PNG()) {
		t.Fatalf("expected PNG to be unsupported")
	}

	scope := Scope{Requestor: "user"}
	params := map[string]any{
		"stage": "adult",
		"limit": "42",
		"flag":  "true",
		"when":  time.Now().UTC().Format(time.RFC3339),
		"alias": stringerValue{value: "stringer"},
	}
	cleaned, errs := host.ValidateParameters(params)
	if len(errs) != 0 {
		t.Fatalf("expected successful validation, got %+v", errs)
	}
	if _, ok := cleaned["limit"].(int); !ok {
		t.Fatalf("expected integer coercion, got %#v", cleaned["limit"])
	}
	if _, ok := cleaned["flag"].(bool); !ok {
		t.Fatalf("expected boolean coercion")
	}
	if ts, ok := cleaned["when"].(time.Time); !ok || ts.IsZero() {
		t.Fatalf("expected timestamp coercion")
	}
	if cleaned["mode"].(string) != "auto" {
		t.Fatalf("expected default parameter value")
	}
	if cleaned["alias"].(string) != "stringer" {
		t.Fatalf("expected stringer coercion to string")
	}

	_, paramErrs := host.ValidateParameters(map[string]any{"stage": "unknown", "limit": 1})
	if len(paramErrs) == 0 || !strings.Contains(paramErrs[0].Message, "value must be one of") {
		t.Fatalf("expected enum validation error, got %+v", paramErrs)
	}
	_, leftovers := host.ValidateParameters(map[string]any{"stage": "adult", "limit": 1, "mystery": 1})
	if len(leftovers) == 0 || leftovers[len(leftovers)-1].Name != "mystery" {
		t.Fatalf("expected leftover parameter error, got %+v", leftovers)
	}

	// Exercise run to ensure cleaned parameters accepted.
	host.runtime = func(context.Context, RunRequest) (RunResult, error) {
		return RunResult{Format: formatProvider.JSON()}, nil
	}
	if _, _, err := host.Run(context.Background(), cleaned, scope, formatProvider.JSON()); err != nil {
		t.Fatalf("run with cleaned params failed: %v", err)
	}
}

func TestSortTemplateDescriptors(t *testing.T) {
	descriptors := []TemplateDescriptor{
		{Plugin: "b", Key: "alpha", Version: "2"},
		{Plugin: "a", Key: "beta", Version: "1"},
		{Plugin: "a", Key: "alpha", Version: "2"},
		{Plugin: "a", Key: "alpha", Version: "1"},
	}
	SortTemplateDescriptors(descriptors)
	expected := []TemplateDescriptor{
		{Plugin: "a", Key: "alpha", Version: "1"},
		{Plugin: "a", Key: "alpha", Version: "2"},
		{Plugin: "a", Key: "beta", Version: "1"},
		{Plugin: "b", Key: "alpha", Version: "2"},
	}
	for i, want := range expected {
		got := descriptors[i]
		if got.Plugin != want.Plugin || got.Key != want.Key || got.Version != want.Version {
			t.Fatalf("unexpected ordering at %d: %+v (want %+v)", i, got, want)
		}
	}
}

func TestSlugAndCloneHelpers(t *testing.T) {
	if slug := slugFor("", "key", "v1"); slug != "key@v1" {
		t.Fatalf("unexpected slug %s", slug)
	}
	if slug := slugFor("plugin", "key", "v1"); slug != "plugin/key@v1" {
		t.Fatalf("unexpected slug with plugin: %s", slug)
	}

	dialectProvider := GetDialectProvider()
	formatProvider := GetFormatProvider()

	tpl := Template{
		Key:           "clone",
		Version:       "1",
		Title:         "Clone",
		Dialect:       dialectProvider.SQL(),
		Query:         "select 1",
		Parameters:    []Parameter{{Name: "enum", Type: "string", Enum: []string{"a"}}},
		Columns:       []Column{{Name: "c", Type: "string"}},
		Metadata:      Metadata{Tags: []string{"t"}, Annotations: map[string]string{"k": "v"}},
		OutputFormats: []Format{formatProvider.JSON()},
	}
	clone := cloneTemplate(tpl)
	clone.Parameters[0].Enum[0] = mutatedLiteral
	clone.Metadata.Tags[0] = mutatedLiteral
	if tpl.Parameters[0].Enum[0] != "a" || tpl.Metadata.Tags[0] != "t" {
		t.Fatalf("expected clone to be defensive")
	}

	scope := Scope{Requestor: "user", Roles: []string{"role"}, ProjectIDs: []string{"project"}}
	clonedScope := cloneScope(scope)
	clonedScope.Roles[0] = "changed"
	if scope.Roles[0] != "role" {
		t.Fatalf("expected scope clone independence")
	}

	if v, err := coerceParameter(Parameter{Name: "num", Type: "number"}, "3.14"); err != nil || v.(float64) != 3.14 {
		t.Fatalf("expected numeric coercion, got %v (%v)", v, err)
	}
	if v, err := coerceParameter(Parameter{Name: "flag", Type: "boolean"}, json.RawMessage([]byte(`"true"`))); err == nil {
		t.Fatalf("expected boolean coercion error for JSON raw message, got %v", v)
	}
	if !containsString([]string{"a", "b"}, "b") {
		t.Fatalf("containsString should find element")
	}
	if err := enumError([]string{"x"}); err == nil {
		t.Fatalf("expected enum error")
	}
}

func TestHostTemplateAccessors(t *testing.T) {
	dialectProvider := GetDialectProvider()
	formatProvider := GetFormatProvider()

	tpl := Template{
		Key:           "key",
		Version:       "1",
		Title:         "title",
		Dialect:       dialectProvider.SQL(),
		Query:         "select 1",
		Columns:       []Column{{Name: "c", Type: "string"}},
		OutputFormats: []Format{formatProvider.JSON()},
		Binder: func(Environment) (Runner, error) {
			return func(context.Context, RunRequest) (RunResult, error) { return RunResult{}, nil }, nil
		},
	}
	if host, err := NewHostTemplate("frog", tpl); err != nil {
		t.Fatalf("NewHostTemplate: %v", err)
	} else {
		if host.Plugin() != "frog" {
			t.Fatalf("unexpected plugin accessor")
		}
		copied := host.Template()
		copied.Key = "mutated"
		if host.Template().Key != "key" {
			t.Fatalf("expected Template accessor to return copy")
		}
	}
}

func TestHostTemplateRunRequiresBind(t *testing.T) {
	dialectProvider := GetDialectProvider()
	formatProvider := GetFormatProvider()

	tpl := Template{
		Key:           "k",
		Version:       "1",
		Title:         "t",
		Dialect:       dialectProvider.SQL(),
		Query:         "select 1",
		Columns:       []Column{{Name: "c", Type: "string"}},
		OutputFormats: []Format{formatProvider.JSON()},
		Parameters:    []Parameter{{Name: "p", Type: "string"}},
		Binder: func(Environment) (Runner, error) {
			return func(context.Context, RunRequest) (RunResult, error) { return RunResult{}, nil }, nil
		},
	}
	host, err := NewHostTemplate("plugin", tpl)
	if err != nil {
		t.Fatalf("NewHostTemplate: %v", err)
	}
	if _, _, err := host.Run(context.Background(), nil, Scope{}, formatProvider.JSON()); err == nil {
		t.Fatalf("expected run to fail when not bound")
	}
}

func TestValidateTemplateDetailedErrors(t *testing.T) {
	bad := Template{}
	if err := validateTemplate(bad); err == nil {
		t.Fatalf("expected validation failure for empty template")
	}
	formatProvider := GetFormatProvider()
	bad = Template{Key: "k", Version: "1", Title: "t", Query: "select 1", Columns: []Column{{Name: "c", Type: "string"}}, OutputFormats: []Format{formatProvider.JSON()}, Binder: func(Environment) (Runner, error) { return nil, nil }}
	bad.Dialect = Dialect("graphql")
	if err := validateTemplate(bad); err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("expected unsupported dialect error, got %v", err)
	}
}

func TestCoerceParameterErrorBranches(t *testing.T) {
	if _, err := coerceParameter(Parameter{Name: "stage", Type: "string", Enum: []string{"ok"}}, 10); err == nil {
		t.Fatalf("expected string type error")
	}
	if _, err := coerceParameter(Parameter{Name: "limit", Type: "integer"}, "abc"); err == nil {
		t.Fatalf("expected integer parse error")
	}
	if _, err := coerceParameter(Parameter{Name: "num", Type: "number"}, true); err == nil {
		t.Fatalf("expected number parse error")
	}
	if _, err := coerceParameter(Parameter{Name: "flag", Type: "boolean"}, 123); err == nil {
		t.Fatalf("expected boolean parse error")
	}
	if _, err := coerceParameter(Parameter{Name: "when", Type: "timestamp"}, "not-a-time"); err == nil {
		t.Fatalf("expected timestamp parse error")
	}
	if _, err := coerceParameter(Parameter{Name: "unknown", Type: "mystery"}, "val"); err == nil {
		t.Fatalf("expected unsupported type error")
	}
}

func TestCoerceParameterSuccessBranches(t *testing.T) {
	if v, err := coerceParameter(Parameter{Name: "stage", Type: "string", Enum: []string{"ok"}}, stringerValue{value: "ok"}); err != nil || v.(string) != "ok" {
		t.Fatalf("expected string success, got %v (%v)", v, err)
	}
	if v, err := coerceParameter(Parameter{Name: "limit", Type: "integer"}, float64(5)); err != nil || v.(int) != 5 {
		t.Fatalf("expected integer success, got %v (%v)", v, err)
	}
	if v, err := coerceParameter(Parameter{Name: "num", Type: "number"}, 7); err != nil || v.(float64) != 7 {
		t.Fatalf("expected number success, got %v (%v)", v, err)
	}
	if v, err := coerceParameter(Parameter{Name: "flag", Type: "boolean"}, "false"); err != nil || v.(bool) != false {
		t.Fatalf("expected boolean success, got %v (%v)", v, err)
	}
	if v, err := coerceParameter(Parameter{Name: "when", Type: "timestamp"}, time.Now().UTC()); err != nil {
		t.Fatalf("expected timestamp success, got %v", err)
	} else if _, ok := v.(time.Time); !ok {
		t.Fatalf("expected timestamp coercion result")
	}
}
