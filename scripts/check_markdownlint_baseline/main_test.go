package markdownlintbaseline

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const sampleLintJSON = `{"docs/a.md":[{"lineNumber":1,"ruleNames":["MD001"]}]}`

func TestParseMarkdownlintOutputEmpty(t *testing.T) {
	issues, err := parseMarkdownlintOutput([]byte("  \n"))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(issues) != 0 {
		t.Fatalf("expected no issues, got %v", issues)
	}
}

func TestParseMarkdownlintOutputValid(t *testing.T) {
	payload := []byte(`{
  "docs/a.md": [{"lineNumber": 3, "ruleNames": ["MD013", "line-length"]}],
  "README.md": [{"lineNumber": 1, "ruleNames": ["MD041"]}]
}`)
	issues, err := parseMarkdownlintOutput(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(issues))
	}
	if issues[0].File == "" || issues[0].Rule == "" {
		t.Fatalf("expected issue fields to be populated, got %+v", issues[0])
	}
}

func TestParseMarkdownlintOutputFlat(t *testing.T) {
	payload := []byte(`[
  {"fileName":"docs/a.md","lineNumber":3,"ruleNames":["MD013","line-length"]},
  {"fileName":"README.md","lineNumber":1,"ruleNames":["MD041"]}
]`)
	issues, err := parseMarkdownlintOutput(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(issues))
	}
	if issues[0].File == "" || issues[0].Rule == "" {
		t.Fatalf("expected issue fields to be populated, got %+v", issues[0])
	}
}

func TestParseMarkdownlintOutputMissingRule(t *testing.T) {
	payload := []byte(`{"docs/a.md": [{"lineNumber": 1, "ruleNames": []}]}`)
	if _, err := parseMarkdownlintOutput(payload); err == nil {
		t.Fatalf("expected error for missing rule name")
	}
}

func TestParseMarkdownlintOutputInvalidJSON(t *testing.T) {
	if _, err := parseMarkdownlintOutput([]byte("{")); err == nil {
		t.Fatalf("expected error for invalid JSON")
	}
}

func TestParseMarkdownlintOutputEmptyMap(t *testing.T) {
	issues, err := parseMarkdownlintOutput([]byte(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(issues) != 0 {
		t.Fatalf("expected no issues, got %+v", issues)
	}
}

func TestFirstRuleName(t *testing.T) {
	if got := firstRuleName([]string{"", "MD001"}); got != "MD001" {
		t.Fatalf("expected MD001, got %q", got)
	}
	if got := firstRuleName([]string{" ", ""}); got != "" {
		t.Fatalf("expected empty rule name, got %q", got)
	}
}

func TestNormalizeIssues(t *testing.T) {
	issues := []lintIssue{
		{File: "b.md", Line: 2, Rule: "MD002"},
		{File: "a.md", Line: 1, Rule: "MD001"},
		{File: "b.md", Line: 2, Rule: "MD002"},
		{File: "", Line: 0, Rule: ""},
	}
	normalized := normalizeIssues(issues)
	if len(normalized) != 2 {
		t.Fatalf("expected 2 normalized issues, got %d", len(normalized))
	}
	if normalized[0].File != "a.md" {
		t.Fatalf("expected sorted issues, got %+v", normalized)
	}
}

func TestLoadBaselineMissing(t *testing.T) {
	if _, err := loadBaseline(filepath.Join(t.TempDir(), "missing.json")); err == nil {
		t.Fatalf("expected error for missing baseline")
	}
}

func TestLoadBaselineInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "baseline.json")
	if err := os.WriteFile(path, []byte("{"), 0o600); err != nil {
		t.Fatalf("write baseline: %v", err)
	}
	if _, err := loadBaseline(path); err == nil {
		t.Fatalf("expected error for invalid JSON")
	}
}

func TestLoadBaselineEmptyObject(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "baseline.json")
	if err := os.WriteFile(path, []byte(`{}`), 0o600); err != nil {
		t.Fatalf("write baseline: %v", err)
	}
	loaded, err := loadBaseline(path)
	if err != nil {
		t.Fatalf("load baseline: %v", err)
	}
	if len(loaded.Issues) != 0 {
		t.Fatalf("expected no issues, got %+v", loaded.Issues)
	}
}

func TestLoadBaselineIssuesOnly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "baseline.json")
	payload := []lintIssue{{File: "a.md", Line: 1, Rule: "MD001"}}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write baseline: %v", err)
	}
	loaded, err := loadBaseline(path)
	if err != nil {
		t.Fatalf("load baseline: %v", err)
	}
	if len(loaded.Issues) != 1 || loaded.Issues[0].Rule != "MD001" {
		t.Fatalf("unexpected baseline issues: %+v", loaded.Issues)
	}
}

func TestWriteBaselineAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "baseline.json")
	meta := baselineMeta{Tool: "markdownlint-cli", Notes: "note"}
	issues := []lintIssue{{File: "a.md", Line: 2, Rule: "MD002"}}
	if err := writeBaseline(path, meta, issues); err != nil {
		t.Fatalf("write baseline: %v", err)
	}
	loaded, err := loadBaseline(path)
	if err != nil {
		t.Fatalf("load baseline: %v", err)
	}
	if loaded.Meta.Tool != meta.Tool {
		t.Fatalf("expected meta to round-trip, got %+v", loaded.Meta)
	}
}

func TestIsZeroMeta(t *testing.T) {
	if !isZeroMeta(baselineMeta{}) {
		t.Fatalf("expected zero meta to be true")
	}
	if isZeroMeta(baselineMeta{Tool: "markdownlint-cli"}) {
		t.Fatalf("expected non-zero meta to be false")
	}
}

func TestMergeMetaPrefersExisting(t *testing.T) {
	defaults := defaultBaselineMeta()
	existing := baselineMeta{
		Tool:             "custom-tool",
		Config:           "config.yaml",
		Scope:            "docs/*.md",
		References:       []string{"#999"},
		Notes:            "note",
		Owner:            "Owner",
		ReductionTargets: []baselineReductionTarget{{Milestone: "M1", TargetPercent: 10, DueDate: "2026-02-15"}},
		NextReviewDate:   "2026-02-15",
		Workflow:         "docs/annex/0005-documentation-standards.md",
		CIPolicy:         "manual",
	}
	merged := mergeMeta(existing, defaults)
	if merged.Tool != existing.Tool {
		t.Fatalf("expected tool to be preserved, got %q", merged.Tool)
	}
	if merged.Config != existing.Config {
		t.Fatalf("expected config to be preserved, got %q", merged.Config)
	}
	if merged.Scope != existing.Scope {
		t.Fatalf("expected scope to be preserved, got %q", merged.Scope)
	}
	if merged.Owner != existing.Owner {
		t.Fatalf("expected owner to be preserved, got %q", merged.Owner)
	}
	if len(merged.References) != 1 || merged.References[0] != "#999" {
		t.Fatalf("expected references to be preserved, got %+v", merged.References)
	}
	if merged.Notes != existing.Notes {
		t.Fatalf("expected notes to be preserved, got %q", merged.Notes)
	}
	if len(merged.ReductionTargets) != 1 || merged.ReductionTargets[0].Milestone != "M1" {
		t.Fatalf("expected reduction targets to be preserved, got %+v", merged.ReductionTargets)
	}
	if merged.NextReviewDate != existing.NextReviewDate {
		t.Fatalf("expected next review date to be preserved, got %q", merged.NextReviewDate)
	}
	if merged.Workflow != existing.Workflow {
		t.Fatalf("expected workflow to be preserved, got %q", merged.Workflow)
	}
	if merged.CIPolicy != existing.CIPolicy {
		t.Fatalf("expected CI policy to be preserved, got %q", merged.CIPolicy)
	}
}

func TestDiffIssues(t *testing.T) {
	current := []lintIssue{
		{File: "a.md", Line: 1, Rule: "MD001"},
		{File: "b.md", Line: 2, Rule: "MD002"},
	}
	baseline := []lintIssue{{File: "a.md", Line: 1, Rule: "MD001"}}
	delta := diffIssues(current, baseline)
	if len(delta) != 1 || delta[0].File != "b.md" {
		t.Fatalf("expected one new issue, got %+v", delta)
	}
}

func TestDiffIssuesEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		current  []lintIssue
		baseline []lintIssue
		wantLen  int
		want     []lintIssue
	}{
		{
			name:     "current empty baseline non-empty",
			current:  nil,
			baseline: []lintIssue{{File: "a.md", Line: 1, Rule: "MD001"}},
			wantLen:  0,
		},
		{
			name:     "baseline empty current populated",
			current:  []lintIssue{{File: "a.md", Line: 1, Rule: "MD001"}, {File: "b.md", Line: 2, Rule: "MD002"}},
			baseline: nil,
			wantLen:  2,
			want:     []lintIssue{{File: "a.md", Line: 1, Rule: "MD001"}, {File: "b.md", Line: 2, Rule: "MD002"}},
		},
		{
			name:     "identical current and baseline",
			current:  []lintIssue{{File: "a.md", Line: 1, Rule: "MD001"}},
			baseline: []lintIssue{{File: "a.md", Line: 1, Rule: "MD001"}},
			wantLen:  0,
			want:     []lintIssue{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			delta := diffIssues(tc.current, tc.baseline)
			if len(delta) != tc.wantLen {
				t.Fatalf("expected %d issue(s), got %d (%+v)", tc.wantLen, len(delta), delta)
			}
			if tc.want != nil && !reflect.DeepEqual(delta, tc.want) {
				t.Fatalf("expected delta %+v, got %+v", tc.want, delta)
			}
		})
	}
}

func TestRunMissingBaselineFlag(t *testing.T) {
	var stderr bytes.Buffer
	exitCode := Run([]string{"cmd"}, &stderr, strings.NewReader("{}"))
	if exitCode != 2 {
		t.Fatalf("expected exit code 2, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "Usage:") {
		t.Fatalf("expected usage output, got %q", stderr.String())
	}
}

func TestRunFlagParseError(t *testing.T) {
	var stderr bytes.Buffer
	exitCode := Run([]string{"cmd", "--unknown-flag"}, &stderr, strings.NewReader("{}"))
	if exitCode != 2 {
		t.Fatalf("expected exit code 2 for flag error, got %d", exitCode)
	}
}

func TestRunMissingBaselineWriteFailure(t *testing.T) {
	exitCode := Run([]string{"cmd"}, failingWriter{}, strings.NewReader("{}"))
	if exitCode != 1 {
		t.Fatalf("expected exit code 1 for write failure, got %d", exitCode)
	}
}

func TestRunUpdateWritesBaseline(t *testing.T) {
	dir := t.TempDir()
	baselinePath := filepath.Join(dir, "baseline.json")
	input := sampleLintJSON
	var stderr bytes.Buffer
	exitCode := Run([]string{"cmd", "--baseline", baselinePath, "--update"}, &stderr, strings.NewReader(input))
	if exitCode != 0 {
		t.Fatalf("expected update to succeed, got %d (%s)", exitCode, stderr.String())
	}
	if _, err := os.Stat(baselinePath); err != nil {
		t.Fatalf("expected baseline file to exist: %v", err)
	}
}

func TestRunParseError(t *testing.T) {
	dir := t.TempDir()
	baselinePath := filepath.Join(dir, "baseline.json")
	var stderr bytes.Buffer
	exitCode := Run([]string{"cmd", "--baseline", baselinePath}, &stderr, strings.NewReader("{"))
	if exitCode != 1 {
		t.Fatalf("expected parse error exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "parse lint output") {
		t.Fatalf("expected parse error output, got %q", stderr.String())
	}
}

func TestRunUpdateWriteBaselineError(t *testing.T) {
	dir := t.TempDir()
	input := sampleLintJSON
	var stderr bytes.Buffer
	exitCode := Run([]string{"cmd", "--baseline", dir, "--update"}, &stderr, strings.NewReader(input))
	if exitCode != 1 {
		t.Fatalf("expected write error exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "write baseline") {
		t.Fatalf("expected write baseline error output, got %q", stderr.String())
	}
}

func TestRunUpdateKeepsExistingMeta(t *testing.T) {
	dir := t.TempDir()
	baselinePath := filepath.Join(dir, "baseline.json")
	originalMeta := baselineMeta{Tool: "markdownlint-cli", Notes: "keep"}
	if err := writeBaseline(baselinePath, originalMeta, []lintIssue{}); err != nil {
		t.Fatalf("write baseline: %v", err)
	}
	input := sampleLintJSON
	var stderr bytes.Buffer
	exitCode := Run([]string{"cmd", "--baseline", baselinePath, "--update"}, &stderr, strings.NewReader(input))
	if exitCode != 0 {
		t.Fatalf("expected update to succeed, got %d (%s)", exitCode, stderr.String())
	}
	loaded, err := loadBaseline(baselinePath)
	if err != nil {
		t.Fatalf("load baseline: %v", err)
	}
	if loaded.Meta.Notes != originalMeta.Notes {
		t.Fatalf("expected meta to be preserved, got %+v", loaded.Meta)
	}
}

func TestRunDetectsNewIssues(t *testing.T) {
	dir := t.TempDir()
	baselinePath := filepath.Join(dir, "baseline.json")
	if err := writeBaseline(baselinePath, defaultBaselineMeta(), []lintIssue{{File: "docs/a.md", Line: 1, Rule: "MD001"}}); err != nil {
		t.Fatalf("write baseline: %v", err)
	}
	input := `{"docs/a.md":[{"lineNumber":1,"ruleNames":["MD001"]}],"docs/b.md":[{"lineNumber":2,"ruleNames":["MD002"]}]}`
	var stderr bytes.Buffer
	exitCode := Run([]string{"cmd", "--baseline", baselinePath}, &stderr, strings.NewReader(input))
	if exitCode != 1 {
		t.Fatalf("expected failure for new issues, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "new markdownlint issue") {
		t.Fatalf("expected new issue output, got %q", stderr.String())
	}
}

func TestRunNoNewIssues(t *testing.T) {
	dir := t.TempDir()
	baselinePath := filepath.Join(dir, "baseline.json")
	if err := writeBaseline(baselinePath, defaultBaselineMeta(), []lintIssue{{File: "docs/a.md", Line: 1, Rule: "MD001"}}); err != nil {
		t.Fatalf("write baseline: %v", err)
	}
	input := `{"docs/a.md":[{"lineNumber":1,"ruleNames":["MD001"]}]}`
	var stderr bytes.Buffer
	exitCode := Run([]string{"cmd", "--baseline", baselinePath}, &stderr, strings.NewReader(input))
	if exitCode != 0 {
		t.Fatalf("expected success, got %d (%s)", exitCode, stderr.String())
	}
}

func TestRunUsesInputFile(t *testing.T) {
	dir := t.TempDir()
	baselinePath := filepath.Join(dir, "baseline.json")
	inputPath := filepath.Join(dir, "lint.json")
	if err := writeBaseline(baselinePath, defaultBaselineMeta(), []lintIssue{}); err != nil {
		t.Fatalf("write baseline: %v", err)
	}
	if err := os.WriteFile(inputPath, []byte(sampleLintJSON), 0o600); err != nil {
		t.Fatalf("write input: %v", err)
	}
	var stderr bytes.Buffer
	exitCode := Run([]string{"cmd", "--baseline", baselinePath, "--input", inputPath}, &stderr, strings.NewReader(""))
	if exitCode != 1 {
		t.Fatalf("expected failure for new issues, got %d", exitCode)
	}
}

func TestReadLintInputMissingFile(t *testing.T) {
	if _, err := readLintInput(filepath.Join(t.TempDir(), "missing.json"), strings.NewReader("")); err == nil {
		t.Fatalf("expected error for missing input file")
	}
}

func TestReportErrorWriterFailure(t *testing.T) {
	exit := reportError(failingWriter{}, "message")
	if exit != 1 {
		t.Fatalf("expected reportError to return 1, got %d", exit)
	}
}

type failingWriter struct{}

func (failingWriter) Write(_ []byte) (int, error) {
	return 0, errWriteFailure
}

var errWriteFailure = errors.New("write failed")
