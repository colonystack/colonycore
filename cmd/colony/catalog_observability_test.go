package main

import (
	"context"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"colonycore/internal/observability"
)

type captureCatalogEvents struct {
	events []observability.Event
}

func (c *captureCatalogEvents) Record(_ context.Context, event observability.Event) {
	c.events = append(c.events, event)
}

func (c *captureCatalogEvents) find(name, status string) (observability.Event, bool) {
	for _, event := range c.events {
		if event.Name == name && event.Status == status {
			return event, true
		}
	}
	return observability.Event{}, false
}

func TestCatalogAddCLIEmitsSuccessEvent(t *testing.T) {
	originalFactory := catalogEventRecorderFactory
	recorder := &captureCatalogEvents{}
	catalogEventRecorderFactory = func(io.Writer) observability.Recorder { return recorder }
	defer func() { catalogEventRecorderFactory = originalFactory }()

	dir := t.TempDir()
	catalogPath := filepath.Join(dir, "catalog", testCatalogFile)
	auditPath := filepath.Join(dir, "catalog", testAuditFile)
	metadataDir := filepath.Join(dir, "catalog", testMetadataDir)
	descriptor := validTemplateDescriptor()
	templatePath := filepath.Join(dir, "template.json")
	writeTemplateFile(t, templatePath, descriptor)

	var stdout, stderr strings.Builder
	code := cli([]string{
		"catalog",
		"add",
		"--catalog", catalogPath,
		"--audit-log", auditPath,
		"--metadata-dir", metadataDir,
		"--actor", "tester",
		"--observability-json",
		templatePath,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected add success, got %d stderr=%q", code, stderr.String())
	}

	event, ok := recorder.find("catalog.add", observability.StatusSuccess)
	if !ok {
		t.Fatalf("expected catalog.add success event, got %+v", recorder.events)
	}
	if event.Labels["template_id"] == "" {
		t.Fatalf("expected template_id label in event: %+v", event)
	}
}

func TestCatalogValidateCLIEmitsErrorEvent(t *testing.T) {
	originalFactory := catalogEventRecorderFactory
	recorder := &captureCatalogEvents{}
	catalogEventRecorderFactory = func(io.Writer) observability.Recorder { return recorder }
	defer func() { catalogEventRecorderFactory = originalFactory }()

	dir := t.TempDir()
	catalogPath := filepath.Join(dir, testCatalogFile)
	auditPath := filepath.Join(dir, "missing", testAuditFile)
	metadataDir := filepath.Join(dir, testMetadataDir)

	var stdout, stderr strings.Builder
	code := cli([]string{
		"catalog",
		"validate",
		"--catalog", catalogPath,
		"--audit-log", auditPath,
		"--metadata-dir", metadataDir,
		"--actor", "tester",
		"--observability-json",
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected validate failure due to missing audit log, got %d stderr=%q", code, stderr.String())
	}
	if _, ok := recorder.find("catalog.validate", observability.StatusError); !ok {
		t.Fatalf("expected catalog.validate error event, got %+v", recorder.events)
	}
}

func TestCatalogCLIObservabilityRequiresOptIn(t *testing.T) {
	dir := t.TempDir()
	catalogPath := filepath.Join(dir, "catalog", testCatalogFile)
	auditPath := filepath.Join(dir, "catalog", testAuditFile)
	metadataDir := filepath.Join(dir, "catalog", testMetadataDir)
	descriptor := validTemplateDescriptor()
	templatePath := filepath.Join(dir, "template.json")
	writeTemplateFile(t, templatePath, descriptor)

	var stdout, stderr strings.Builder
	code := cli([]string{
		"catalog",
		"add",
		"--catalog", catalogPath,
		"--audit-log", auditPath,
		"--metadata-dir", metadataDir,
		templatePath,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected add success without observability flag, got %d stderr=%q", code, stderr.String())
	}
	if strings.Contains(stderr.String(), "\"schema_version\"") {
		t.Fatalf("expected no structured events without opt-in, got %q", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	second := descriptor
	second.Version = "0.2.0"
	second.Slug = second.Plugin + "/" + second.Key + "@0.2.0"
	secondPath := filepath.Join(dir, "template-v2.json")
	writeTemplateFile(t, secondPath, second)
	code = cli([]string{
		"catalog",
		"add",
		"--catalog", catalogPath,
		"--audit-log", auditPath,
		"--metadata-dir", metadataDir,
		"--observability-json",
		secondPath,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected add success with observability flag, got %d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "\"schema_version\":\"colonycore.observability.v1\"") {
		t.Fatalf("expected structured events with opt-in, got %q", stderr.String())
	}
}
