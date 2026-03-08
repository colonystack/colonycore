package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sampleLintJSON = `{"docs/a.md":[{"lineNumber":1,"ruleNames":["MD001"]}]}`

type baselineIssue struct {
	File string `json:"file"`
	Line int    `json:"line"`
	Rule string `json:"rule"`
}

type baselineFile struct {
	Issues []baselineIssue `json:"issues"`
}

func TestRunUpdateWritesBaselineContents(t *testing.T) {
	dir := t.TempDir()
	baselinePath := filepath.Join(dir, "baseline.json")
	var stderr bytes.Buffer
	exitCode := run([]string{"cmd", "--baseline", baselinePath, "--update"}, &stderr, strings.NewReader(sampleLintJSON))
	if exitCode != 0 {
		t.Fatalf("expected update to succeed, got %d (%s)", exitCode, stderr.String())
	}
	// #nosec G304 -- baseline path is test-controlled via TempDir.
	data, err := os.ReadFile(baselinePath)
	if err != nil {
		t.Fatalf("read baseline: %v", err)
	}
	var baseline baselineFile
	if err := json.Unmarshal(data, &baseline); err != nil {
		t.Fatalf("unmarshal baseline: %v", err)
	}
	if len(baseline.Issues) != 1 {
		t.Fatalf("expected one issue, got %+v", baseline.Issues)
	}
	if baseline.Issues[0].Rule != "MD001" {
		t.Fatalf("expected rule MD001, got %+v", baseline.Issues[0])
	}
}

func TestRunCheckModeReportsNewIssues(t *testing.T) {
	dir := t.TempDir()
	baselinePath := filepath.Join(dir, "baseline.json")
	var stderr bytes.Buffer
	exitCode := run([]string{"cmd", "--baseline", baselinePath, "--update"}, &stderr, strings.NewReader(sampleLintJSON))
	if exitCode != 0 {
		t.Fatalf("expected update to succeed, got %d (%s)", exitCode, stderr.String())
	}
	stderr.Reset()
	exitCode = run([]string{"cmd", "--baseline", baselinePath}, &stderr, strings.NewReader(`{"docs/b.md":[{"lineNumber":2,"ruleNames":["MD002"]}]}`))
	if exitCode != 1 {
		t.Fatalf("expected check mode to fail, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "Found") {
		t.Fatalf("expected diff output, got %q", stderr.String())
	}
}

func TestRunInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	baselinePath := filepath.Join(dir, "baseline.json")
	var stderr bytes.Buffer
	exitCode := run([]string{"cmd", "--baseline", baselinePath}, &stderr, strings.NewReader("{"))
	if exitCode != 1 {
		t.Fatalf("expected parse error, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "parse lint output") {
		t.Fatalf("expected parse error output, got %q", stderr.String())
	}
}

func TestRunUpdateWriteFailure(t *testing.T) {
	dir := t.TempDir()
	var stderr bytes.Buffer
	exitCode := run([]string{"cmd", "--baseline", dir, "--update"}, &stderr, strings.NewReader(sampleLintJSON))
	if exitCode != 1 {
		t.Fatalf("expected write failure, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "write baseline") {
		t.Fatalf("expected write baseline error output, got %q", stderr.String())
	}
}

func TestRunUpdateEmptyInput(t *testing.T) {
	dir := t.TempDir()
	baselinePath := filepath.Join(dir, "baseline.json")
	var stderr bytes.Buffer
	exitCode := run([]string{"cmd", "--baseline", baselinePath, "--update"}, &stderr, strings.NewReader(""))
	if exitCode != 0 {
		t.Fatalf("expected empty input update to succeed, got %d (%s)", exitCode, stderr.String())
	}
}
