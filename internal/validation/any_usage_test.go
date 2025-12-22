package validation

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAnyAllowlistErrors(t *testing.T) {
	if _, err := LoadAnyAllowlist(filepath.Join(t.TempDir(), "missing.json")); err == nil {
		t.Fatalf("expected error for missing allowlist")
	}
	path := filepath.Join(t.TempDir(), "allow.json")
	if err := os.WriteFile(path, []byte("invalid"), 0o600); err != nil {
		t.Fatalf("write invalid allowlist: %v", err)
	}
	if _, err := LoadAnyAllowlist(path); err == nil {
		t.Fatalf("expected error for invalid allowlist json")
	}
}

func TestValidateAnyUsageFromFile(t *testing.T) {
	base := t.TempDir()
	writeFile(t, filepath.Join(base, "pkg", "pluginapi", "payload.go"), `package pluginapi
type Payload map[string]any
`)
	allowlist := AnyAllowlist{
		Version:      1,
		ExcludeGlobs: []string{"**/*_test.go"},
		Entries: []AnyAllowlistEntry{
			{
				Path:      "pkg/pluginapi/payload.go",
				Category:  "json-boundary",
				Public:    true,
				Rationale: "test allowlist file load",
				Owner:     "core maintainers",
			},
		},
	}
	data, err := json.Marshal(allowlist)
	if err != nil {
		t.Fatalf("marshal allowlist: %v", err)
	}
	allowPath := filepath.Join(base, "allowlist.json")
	if err := os.WriteFile(allowPath, data, 0o600); err != nil {
		t.Fatalf("write allowlist: %v", err)
	}
	violations, err := ValidateAnyUsageFromFile(allowPath, base, []string{"pkg/pluginapi"})
	if err != nil {
		t.Fatalf("validate any usage from file: %v", err)
	}
	if len(violations) != 0 {
		t.Fatalf("expected no violations, got %v", violations)
	}
}

func TestValidateAnyUsageAllowsListedSymbol(t *testing.T) {
	base := t.TempDir()
	writeFile(t, filepath.Join(base, "pkg", "pluginapi", "payload.go"), `package pluginapi
type Payload map[string]any
`)

	allowlist := AnyAllowlist{
		Version:      1,
		ExcludeGlobs: []string{"**/*_test.go"},
		Entries: []AnyAllowlistEntry{
			{
				Path:      "pkg/pluginapi/payload.go",
				Symbols:   []string{"Payload"},
				Category:  "json-boundary",
				Public:    true,
				Rationale: "test json boundary",
				Owner:     "core maintainers",
			},
		},
	}

	violations, err := ValidateAnyUsage(allowlist, base, []string{"pkg/pluginapi"})
	if err != nil {
		t.Fatalf("validate any usage: %v", err)
	}
	if len(violations) != 0 {
		t.Fatalf("expected no violations, got %v", violations)
	}
}

func TestValidateAnyUsageFlagsUnlistedAny(t *testing.T) {
	base := t.TempDir()
	writeFile(t, filepath.Join(base, "pkg", "pluginapi", "payload.go"), `package pluginapi
type Payload map[string]any
`)

	allowlist := AnyAllowlist{
		Version:      1,
		ExcludeGlobs: []string{"**/*_test.go"},
	}

	violations, err := ValidateAnyUsage(allowlist, base, []string{"pkg/pluginapi"})
	if err != nil {
		t.Fatalf("validate any usage: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
	if violations[0].File != "pkg/pluginapi/payload.go" {
		t.Fatalf("unexpected file: %s", violations[0].File)
	}
	if violations[0].Line != 2 {
		t.Fatalf("unexpected line: %d", violations[0].Line)
	}
	if violations[0].Code != "type Payload map[string]any" {
		t.Fatalf("unexpected code: %q", violations[0].Code)
	}
}

func TestValidateAnyUsageSkipsExcludedGlobs(t *testing.T) {
	base := t.TempDir()
	writeFile(t, filepath.Join(base, "pkg", "pluginapi", "payload_test.go"), `package pluginapi
type Payload map[string]any
`)

	allowlist := AnyAllowlist{
		Version:      1,
		ExcludeGlobs: []string{"**/*_test.go"},
	}

	violations, err := ValidateAnyUsage(allowlist, base, []string{"pkg/pluginapi"})
	if err != nil {
		t.Fatalf("validate any usage: %v", err)
	}
	if len(violations) != 0 {
		t.Fatalf("expected no violations, got %v", violations)
	}
}

func TestValidateAnyUsageAllowsFileLevelEntry(t *testing.T) {
	base := t.TempDir()
	writeFile(t, filepath.Join(base, "pkg", "pluginapi", "payload.go"), `package pluginapi
type Payload map[string]any
`)

	allowlist := AnyAllowlist{
		Version:      1,
		ExcludeGlobs: []string{"**/*_test.go"},
		Entries: []AnyAllowlistEntry{
			{
				Path:      "pkg/pluginapi/payload.go",
				Category:  "json-boundary",
				Public:    true,
				Rationale: "test file-level allowlist",
				Owner:     "core maintainers",
			},
		},
	}

	violations, err := ValidateAnyUsage(allowlist, base, []string{"pkg/pluginapi"})
	if err != nil {
		t.Fatalf("validate any usage: %v", err)
	}
	if len(violations) != 0 {
		t.Fatalf("expected no violations, got %v", violations)
	}
}

func TestValidateAnyUsageAllowsTypeParamConstraint(t *testing.T) {
	base := t.TempDir()
	writeFile(t, filepath.Join(base, "pkg", "pluginapi", "generic.go"), `package pluginapi
func Use[T any](v T) {}
`)

	allowlist := AnyAllowlist{
		Version:      1,
		ExcludeGlobs: []string{"**/*_test.go"},
	}

	violations, err := ValidateAnyUsage(allowlist, base, []string{"pkg/pluginapi"})
	if err != nil {
		t.Fatalf("validate any usage: %v", err)
	}
	if len(violations) != 0 {
		t.Fatalf("expected no violations, got %v", violations)
	}
}

func TestValidateAnyUsageAllowsReceiverSymbol(t *testing.T) {
	base := t.TempDir()
	writeFile(t, filepath.Join(base, "pkg", "datasetapi", "host_template.go"), `package datasetapi
type HostTemplate struct{}
func (h HostTemplate) Run(params map[string]any) {}
`)

	allowlist := AnyAllowlist{
		Version:      1,
		ExcludeGlobs: []string{"**/*_test.go"},
		Entries: []AnyAllowlistEntry{
			{
				Path:      "pkg/datasetapi/host_template.go",
				Symbols:   []string{"HostTemplate"},
				Category:  "json-boundary",
				Public:    true,
				Rationale: "test receiver mapping",
				Owner:     "core maintainers",
			},
		},
	}

	violations, err := ValidateAnyUsage(allowlist, base, []string{"pkg/datasetapi"})
	if err != nil {
		t.Fatalf("validate any usage: %v", err)
	}
	if len(violations) != 0 {
		t.Fatalf("expected no violations, got %v", violations)
	}
}

func TestValidateAnyUsageCoversTypeExpressions(t *testing.T) {
	base := t.TempDir()
	writeFile(t, filepath.Join(base, "pkg", "pluginapi", "types.go"), `package pluginapi
type Box[T any] struct{}
type Pair[A, B any] struct{}
func Use(value any) {
	_ = value.(any)
	_ = any(value)
	var _ Box[any]
	var _ Pair[int, any]
}
`)

	allowlist := AnyAllowlist{
		Version:      1,
		ExcludeGlobs: []string{"**/*_test.go"},
		Entries: []AnyAllowlistEntry{
			{
				Path:      "pkg/pluginapi/types.go",
				Symbols:   []string{"Use"},
				Category:  "internal-helper",
				Public:    false,
				Rationale: "test type expression coverage",
				Owner:     "core maintainers",
			},
		},
	}

	violations, err := ValidateAnyUsage(allowlist, base, []string{"pkg/pluginapi"})
	if err != nil {
		t.Fatalf("validate any usage: %v", err)
	}
	if len(violations) != 0 {
		t.Fatalf("expected no violations, got %v", violations)
	}
}

func TestValidateAnyUsageMissingRoot(t *testing.T) {
	allowlist := AnyAllowlist{Version: 1}
	if _, err := ValidateAnyUsage(allowlist, t.TempDir(), []string{"missing"}); err == nil {
		t.Fatalf("expected error for missing root")
	}
}

func TestValidateAnyUsageRejectsInvalidGoFile(t *testing.T) {
	base := t.TempDir()
	writeFile(t, filepath.Join(base, "pkg", "pluginapi", "broken.go"), "package pluginapi\nfunc\n")
	allowlist := AnyAllowlist{
		Version:      1,
		ExcludeGlobs: []string{"**/*_test.go"},
	}
	if _, err := ValidateAnyUsage(allowlist, base, []string{"pkg/pluginapi"}); err == nil {
		t.Fatalf("expected error for invalid go file")
	}
}

func TestValidateAnyUsageRequiresRoots(t *testing.T) {
	allowlist := AnyAllowlist{Version: 1}
	if _, err := ValidateAnyUsage(allowlist, t.TempDir(), nil); err == nil {
		t.Fatalf("expected error for missing roots")
	}
}

func TestValidateAnyUsageRejectsNonDirectoryRoot(t *testing.T) {
	base := t.TempDir()
	filePath := filepath.Join(base, "file.go")
	if err := os.WriteFile(filePath, []byte("package main\n"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	allowlist := AnyAllowlist{Version: 1}
	if _, err := ValidateAnyUsage(allowlist, base, []string{filePath}); err == nil {
		t.Fatalf("expected error for non-directory root")
	}
}

func TestValidateAnyUsageSkipsEmptyRoot(t *testing.T) {
	allowlist := AnyAllowlist{Version: 1}
	violations, err := ValidateAnyUsage(allowlist, t.TempDir(), []string{""})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(violations) != 0 {
		t.Fatalf("expected no violations, got %v", violations)
	}
}

func TestValidateAnyUsageRejectsInvalidAllowlistVersion(t *testing.T) {
	allowlist := AnyAllowlist{Version: 0}
	if _, err := ValidateAnyUsage(allowlist, t.TempDir(), []string{"pkg/pluginapi"}); err == nil {
		t.Fatalf("expected error for invalid allowlist version")
	}
}

func TestAllowlistValidationCatchesUnknownCategory(t *testing.T) {
	base := t.TempDir()
	writeFile(t, filepath.Join(base, "pkg", "pluginapi", "payload.go"), `package pluginapi
type Payload map[string]any
`)
	allowlist := AnyAllowlist{
		Version: 1,
		Entries: []AnyAllowlistEntry{
			{
				Path:      "pkg/pluginapi/payload.go",
				Symbols:   []string{"Payload"},
				Category:  "unknown",
				Public:    true,
				Rationale: "test",
				Owner:     "core maintainers",
			},
		},
	}
	data, err := json.Marshal(allowlist)
	if err != nil {
		t.Fatalf("marshal allowlist: %v", err)
	}
	path := filepath.Join(base, "allowlist.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write allowlist: %v", err)
	}
	if _, err := LoadAnyAllowlist(path); err == nil {
		t.Fatalf("expected error for unknown category")
	}
}

func TestAllowlistValidationRejectsPublicNonBoundary(t *testing.T) {
	base := t.TempDir()
	allowlist := AnyAllowlist{
		Version: 1,
		Entries: []AnyAllowlistEntry{
			{
				Path:      "pkg/pluginapi/payload.go",
				Symbols:   []string{"Payload"},
				Category:  "internal-helper",
				Public:    true,
				Rationale: "test",
				Owner:     "core maintainers",
			},
		},
	}
	data, err := json.Marshal(allowlist)
	if err != nil {
		t.Fatalf("marshal allowlist: %v", err)
	}
	path := filepath.Join(base, "allowlist.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write allowlist: %v", err)
	}
	if _, err := LoadAnyAllowlist(path); err == nil {
		t.Fatalf("expected error for public non-boundary category")
	}
}

func TestValidateAllowlistErrors(t *testing.T) {
	cases := []AnyAllowlist{
		{Version: 0},
		{
			Version: 1,
			Entries: []AnyAllowlistEntry{{Category: "json-boundary", Public: true, Rationale: "r", Owner: "o"}},
		},
		{
			Version: 1,
			Entries: []AnyAllowlistEntry{{Path: "pkg/pluginapi/payload.go", Public: true, Rationale: "r", Owner: "o"}},
		},
		{
			Version: 1,
			Entries: []AnyAllowlistEntry{{Path: "pkg/pluginapi/payload.go", Category: "json-boundary", Public: true, Owner: "o"}},
		},
		{
			Version: 1,
			Entries: []AnyAllowlistEntry{{Path: "pkg/pluginapi/payload.go", Category: "json-boundary", Public: true, Rationale: "r"}},
		},
	}
	for i, tc := range cases {
		if err := validateAllowlist(&tc); err == nil {
			t.Fatalf("expected error for case %d", i)
		}
	}
}

func TestValidateAllowlistTrimsFields(t *testing.T) {
	allowlist := AnyAllowlist{
		Version:      1,
		ExcludeGlobs: []string{" **/*_test.go "},
		Entries: []AnyAllowlistEntry{
			{
				Path:      " pkg/pluginapi/payload.go ",
				Symbols:   []string{" Payload ", ""},
				Category:  "json-boundary",
				Public:    true,
				Rationale: " ok ",
				Owner:     " core ",
			},
		},
	}
	if err := validateAllowlist(&allowlist); err != nil {
		t.Fatalf("validate allowlist: %v", err)
	}
	entry := allowlist.Entries[0]
	if entry.Path != "pkg/pluginapi/payload.go" {
		t.Fatalf("unexpected path: %q", entry.Path)
	}
	if entry.Owner != "core" {
		t.Fatalf("unexpected owner: %q", entry.Owner)
	}
	if entry.Rationale != "ok" {
		t.Fatalf("unexpected rationale: %q", entry.Rationale)
	}
	if len(entry.Symbols) != 1 || entry.Symbols[0] != "Payload" {
		t.Fatalf("unexpected symbols: %v", entry.Symbols)
	}
	if allowlist.ExcludeGlobs[0] != "**/*_test.go" {
		t.Fatalf("unexpected exclude glob: %q", allowlist.ExcludeGlobs[0])
	}
}

func TestNormalizeSymbols(t *testing.T) {
	if got := normalizeSymbols(nil); got != nil {
		t.Fatalf("expected nil symbols, got %v", got)
	}
	if got := normalizeSymbols([]string{"", " "}); got != nil {
		t.Fatalf("expected nil symbols for empty entries, got %v", got)
	}
	got := normalizeSymbols([]string{" Foo ", "Bar"})
	if len(got) != 2 || got[0] != "Foo" || got[1] != "Bar" {
		t.Fatalf("unexpected symbols: %v", got)
	}
}

func TestAllowlistIndexIsAllowed(t *testing.T) {
	index := anyAllowlistIndex{
		allowAll: map[string]bool{"allowed.go": true},
		symbols: map[string]map[string]struct{}{
			"symbols.go": {"Allowed": {}},
		},
	}
	if !index.isAllowed("allowed.go", "") {
		t.Fatalf("expected allowAll to pass")
	}
	if !index.isAllowed("symbols.go", "Allowed") {
		t.Fatalf("expected symbol allowlist to pass")
	}
	if index.isAllowed("symbols.go", "Missing") {
		t.Fatalf("did not expect missing symbol to pass")
	}
	if index.isAllowed("missing.go", "Allowed") {
		t.Fatalf("did not expect unknown file to pass")
	}
}

func TestTypeParamRangesNil(t *testing.T) {
	if got := typeParamRanges(nil); got != nil {
		t.Fatalf("expected nil ranges, got %v", got)
	}
}

func TestIsTypeIdentCases(t *testing.T) {
	ident := &ast.Ident{Name: "any"}
	cases := []struct {
		name   string
		parent ast.Node
		child  ast.Node
		want   bool
	}{
		{"array", &ast.ArrayType{Elt: ident}, ident, true},
		{"map-key", &ast.MapType{Key: ident, Value: &ast.Ident{Name: "string"}}, ident, true},
		{"map-value", &ast.MapType{Key: &ast.Ident{Name: "string"}, Value: ident}, ident, true},
		{"chan", &ast.ChanType{Value: ident}, ident, true},
		{"star", &ast.StarExpr{X: ident}, ident, true},
		{"ellipsis", &ast.Ellipsis{Elt: ident}, ident, true},
		{"field", &ast.Field{Type: ident}, ident, true},
		{"value-spec", &ast.ValueSpec{Type: ident}, ident, true},
		{"type-spec", &ast.TypeSpec{Type: ident}, ident, true},
		{"type-assert", &ast.TypeAssertExpr{Type: ident}, ident, true},
		{"index-expr", &ast.IndexExpr{Index: ident}, ident, true},
		{"index-list", &ast.IndexListExpr{Indices: []ast.Expr{ident}}, ident, true},
		{"call-expr", &ast.CallExpr{Fun: ident}, ident, true},
		{"short-stack", ident, ident, false},
		{"unknown-parent", &ast.BasicLit{}, ident, false},
	}
	for _, tc := range cases {
		stack := []ast.Node{tc.parent, tc.child}
		if tc.name == "short-stack" {
			stack = []ast.Node{tc.parent}
		}
		if got := isTypeIdent(stack); got != tc.want {
			t.Fatalf("case %s: expected %v, got %v", tc.name, tc.want, got)
		}
	}
}

func TestReceiverTypeName(t *testing.T) {
	const receiverName = "Host"
	if got := receiverTypeName(&ast.Ident{Name: receiverName}); got != receiverName {
		t.Fatalf("expected Host, got %q", got)
	}
	if got := receiverTypeName(&ast.StarExpr{X: &ast.Ident{Name: receiverName}}); got != receiverName {
		t.Fatalf("expected Host for pointer, got %q", got)
	}
	if got := receiverTypeName(&ast.IndexExpr{X: &ast.Ident{Name: receiverName}}); got != receiverName {
		t.Fatalf("expected Host for index expr, got %q", got)
	}
	if got := receiverTypeName(&ast.IndexListExpr{X: &ast.Ident{Name: receiverName}}); got != receiverName {
		t.Fatalf("expected Host for index list expr, got %q", got)
	}
	if got := receiverTypeName(&ast.ArrayType{}); got != "" {
		t.Fatalf("expected empty name, got %q", got)
	}
}

func TestBuildSymbolRanges(t *testing.T) {
	src := `package sample
type Widget struct{}
type alias int
var Value int
const Answer = 42
func (w *Widget) Run() {}
func Free() {}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "sample.go", src, 0)
	if err != nil {
		t.Fatalf("parse file: %v", err)
	}
	ranges := buildSymbolRanges(file)
	names := make(map[string]struct{}, len(ranges))
	for _, r := range ranges {
		names[r.name] = struct{}{}
	}
	for _, name := range []string{"Widget", "alias", "Value", "Answer", "Free"} {
		if _, ok := names[name]; !ok {
			t.Fatalf("expected symbol %q", name)
		}
	}
}

func TestSymbolForPos(t *testing.T) {
	ranges := []symbolRange{
		{name: "Alpha", start: token.Pos(10), end: token.Pos(20)},
	}
	if got := symbolForPos(ranges, token.Pos(15)); got != "Alpha" {
		t.Fatalf("expected Alpha, got %q", got)
	}
	if got := symbolForPos(ranges, token.Pos(25)); got != "" {
		t.Fatalf("expected empty symbol, got %q", got)
	}
}

func TestShouldExcludeAndMatchGlob(t *testing.T) {
	if !shouldExclude("pkg/pluginapi/foo_test.go", []string{"**/*_test.go"}) {
		t.Fatalf("expected glob to exclude test file")
	}
	if shouldExclude("pkg/pluginapi/foo.go", []string{"**/*_test.go"}) {
		t.Fatalf("did not expect glob to exclude non-test file")
	}
	ok, err := matchGlob("pkg/**/foo*.go", "pkg/pluginapi/foo_test.go")
	if err != nil || !ok {
		t.Fatalf("expected match for recursive glob, got %v (err=%v)", ok, err)
	}
	ok, err = matchGlob("pkg/?oo.go", "pkg/foo.go")
	if err != nil || !ok {
		t.Fatalf("expected match for single-char glob, got %v (err=%v)", ok, err)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
}
