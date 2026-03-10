package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"colonycore/pkg/datasetapi"
)

const (
	testVersion010 = "0.1.0"
	testVersion020 = "0.2.0"
)

func TestCatalogLifecycleCommandsEndToEnd(t *testing.T) {
	dir := t.TempDir()
	catalogPath := filepath.Join(dir, "catalog", "catalog.json")
	auditPath := filepath.Join(dir, "catalog", "audit.log.jsonl")
	metadataDir := filepath.Join(dir, "catalog", "metadata")

	oldDescriptor := validTemplateDescriptor()
	oldDescriptor.Version = testVersion010
	oldDescriptor.Slug = fmt.Sprintf("%s/%s@%s", oldDescriptor.Plugin, oldDescriptor.Key, oldDescriptor.Version)
	oldPath := filepath.Join(dir, "old.json")
	writeTemplateFile(t, oldPath, oldDescriptor)

	newDescriptor := oldDescriptor
	newDescriptor.Version = testVersion020
	newDescriptor.Query = "REPORT frog_population_snapshot\nSELECT organism_id, species FROM organisms"
	newDescriptor.Parameters = append(newDescriptor.Parameters, datasetapi.Parameter{Name: "species", Type: "string"})
	newDescriptor.Columns = append(newDescriptor.Columns, datasetapi.Column{Name: "species", Type: "string"})
	newDescriptor.Slug = fmt.Sprintf("%s/%s@%s", newDescriptor.Plugin, newDescriptor.Key, newDescriptor.Version)
	newPath := filepath.Join(dir, "new.json")
	writeTemplateFile(t, newPath, newDescriptor)

	for _, invocation := range [][]string{
		catalogArgs("add", catalogPath, auditPath, metadataDir, oldPath),
		catalogArgs("add", catalogPath, auditPath, metadataDir, newPath),
		catalogArgs("deprecate", catalogPath, auditPath, metadataDir, "--reason", "upgrade available", "--window-days", "45", oldDescriptor.Slug),
		catalogArgs("migrate", catalogPath, auditPath, metadataDir, oldDescriptor.Slug, newDescriptor.Slug),
		catalogArgs("validate", catalogPath, auditPath, metadataDir),
	} {
		var stdout, stderr strings.Builder
		code := cli(invocation, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("command %q failed with code=%d stderr=%q", strings.Join(invocation, " "), code, stderr.String())
		}
	}

	catalog := readCatalogRegistryForTests(t, catalogPath)
	if len(catalog.Templates) != 2 {
		t.Fatalf("expected 2 templates in catalog, got %d", len(catalog.Templates))
	}
	if len(catalog.Migrations) != 1 {
		t.Fatalf("expected 1 migration record, got %d", len(catalog.Migrations))
	}
	if catalog.Templates[0].Deprecated == nil {
		t.Fatalf("expected one template to be deprecated")
	}
	if catalog.Templates[0].Deprecated.Reason != "upgrade available" {
		t.Fatalf("unexpected deprecation reason: %s", catalog.Templates[0].Deprecated.Reason)
	}
	if _, err := os.Stat(catalog.Templates[0].Deprecated.MetadataPath); err != nil {
		t.Fatalf("expected deprecation metadata file, got %v", err)
	}

	var deprecationMetadata catalogDeprecationMetadata
	readJSONFile(t, catalog.Templates[0].Deprecated.MetadataPath, &deprecationMetadata)
	if deprecationMetadata.TemplateSlug != oldDescriptor.Slug {
		t.Fatalf("deprecation metadata template mismatch: %s", deprecationMetadata.TemplateSlug)
	}
	if deprecationMetadata.Reason != "upgrade available" {
		t.Fatalf("deprecation metadata reason mismatch: %s", deprecationMetadata.Reason)
	}

	migrationRecord := catalog.Migrations[0]
	if _, err := os.Stat(migrationRecord.MetadataPath); err != nil {
		t.Fatalf("expected migration metadata file, got %v", err)
	}
	var migrationPlan catalogMigrationPlan
	readJSONFile(t, migrationRecord.MetadataPath, &migrationPlan)
	if migrationPlan.From != oldDescriptor.Slug || migrationPlan.To != newDescriptor.Slug {
		t.Fatalf("unexpected migration plan endpoints: %s -> %s", migrationPlan.From, migrationPlan.To)
	}
	if len(migrationPlan.Steps) == 0 {
		t.Fatalf("expected migration plan steps")
	}

	if err := verifyCatalogAuditLogChain(auditPath); err != nil {
		t.Fatalf("expected valid audit chain, got %v", err)
	}
	entries := readCatalogAuditEntries(t, auditPath)
	if len(entries) != 5 {
		t.Fatalf("expected 5 audit entries, got %d", len(entries))
	}
	if entries[len(entries)-1].Operation != "catalog_validate" || entries[len(entries)-1].Status != catalogAuditStatusSuccess {
		t.Fatalf("expected final audit entry to be successful validation, got %+v", entries[len(entries)-1])
	}
}

func TestCatalogDeprecateAmbiguousKeyFails(t *testing.T) {
	dir := t.TempDir()
	catalogPath := filepath.Join(dir, "catalog.json")
	auditPath := filepath.Join(dir, "audit.log.jsonl")
	metadataDir := filepath.Join(dir, "metadata")

	first := validTemplateDescriptor()
	first.Version = testVersion010
	first.Slug = fmt.Sprintf("%s/%s@%s", first.Plugin, first.Key, first.Version)
	firstPath := filepath.Join(dir, "first.json")
	writeTemplateFile(t, firstPath, first)
	second := first
	second.Version = testVersion020
	second.Slug = fmt.Sprintf("%s/%s@%s", second.Plugin, second.Key, second.Version)
	secondPath := filepath.Join(dir, "second.json")
	writeTemplateFile(t, secondPath, second)

	for _, invocation := range [][]string{
		catalogArgs("add", catalogPath, auditPath, metadataDir, firstPath),
		catalogArgs("add", catalogPath, auditPath, metadataDir, secondPath),
	} {
		var stdout, stderr strings.Builder
		if code := cli(invocation, &stdout, &stderr); code != 0 {
			t.Fatalf("setup command failed code=%d stderr=%q", code, stderr.String())
		}
	}

	var stdout, stderr strings.Builder
	code := cli(catalogArgs("deprecate", catalogPath, auditPath, metadataDir, "--reason", "replace", first.Key), &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected deprecate command to fail, got code=%d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "ambiguous") {
		t.Fatalf("expected ambiguity error, got %q", stderr.String())
	}

	entries := readCatalogAuditEntries(t, auditPath)
	if got := entries[len(entries)-1].Status; got != catalogAuditStatusError {
		t.Fatalf("expected audit status error, got %s", got)
	}
}

func TestCatalogValidateFailsForInvalidTemplate(t *testing.T) {
	dir := t.TempDir()
	catalogPath := filepath.Join(dir, "catalog.json")
	auditPath := filepath.Join(dir, "audit.log.jsonl")
	metadataDir := filepath.Join(dir, "metadata")

	invalid := catalogRegistry{
		Version:   catalogSchemaVersion,
		UpdatedAt: time.Now().UTC(),
		Templates: []catalogTemplateRecord{{
			Descriptor: datasetapi.TemplateDescriptor{
				Plugin:  "frog",
				Key:     "",
				Version: "1.0.0",
				Title:   "Broken",
				Dialect: datasetapi.GetDialectProvider().SQL(),
				Query:   "SELECT 1",
				Columns: []datasetapi.Column{{Name: "value", Type: "integer"}},
				OutputFormats: []datasetapi.Format{
					datasetapi.GetFormatProvider().JSON(),
				},
				Slug: "frog/@1.0.0",
			},
		}},
	}
	if err := saveCatalogRegistry(catalogPath, invalid); err != nil {
		t.Fatalf("save invalid catalog: %v", err)
	}

	var stdout, stderr strings.Builder
	code := cli(catalogArgs("validate", catalogPath, auditPath, metadataDir), &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected validation failure, got code=%d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "template key is required") {
		t.Fatalf("expected invalid template error, got %q", stderr.String())
	}

	entries := readCatalogAuditEntries(t, auditPath)
	if len(entries) == 0 {
		t.Fatalf("expected validation audit entry")
	}
	if entries[len(entries)-1].Operation != "catalog_validate" || entries[len(entries)-1].Status != catalogAuditStatusError {
		t.Fatalf("expected failed validation audit entry, got %+v", entries[len(entries)-1])
	}
}

func TestCatalogValidateFailsForInvalidDeprecationMetadata(t *testing.T) {
	dir := t.TempDir()
	catalogPath := filepath.Join(dir, "catalog.json")
	auditPath := filepath.Join(dir, "audit.log.jsonl")
	metadataDir := filepath.Join(dir, "metadata")

	descriptor := validTemplateDescriptor()
	descriptor.Version = testVersion010
	descriptor.Slug = fmt.Sprintf("%s/%s@%s", descriptor.Plugin, descriptor.Key, descriptor.Version)
	templatePath := filepath.Join(dir, "template.json")
	writeTemplateFile(t, templatePath, descriptor)

	for _, invocation := range [][]string{
		catalogArgs("add", catalogPath, auditPath, metadataDir, templatePath),
		catalogArgs("deprecate", catalogPath, auditPath, metadataDir, "--reason", "superseded", descriptor.Slug),
	} {
		var stdout, stderr strings.Builder
		if code := cli(invocation, &stdout, &stderr); code != 0 {
			t.Fatalf("setup command failed code=%d stderr=%q", code, stderr.String())
		}
	}

	catalog := readCatalogRegistryForTests(t, catalogPath)
	if len(catalog.Templates) != 1 || catalog.Templates[0].Deprecated == nil {
		t.Fatalf("expected one deprecated template in catalog")
	}
	if err := os.WriteFile(catalog.Templates[0].Deprecated.MetadataPath, []byte("{"), 0o600); err != nil {
		t.Fatalf("corrupt deprecation metadata: %v", err)
	}

	var stdout, stderr strings.Builder
	code := cli(catalogArgs("validate", catalogPath, auditPath, metadataDir), &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected validate to fail, got code=%d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "deprecation metadata") {
		t.Fatalf("expected deprecation metadata failure, got %q", stderr.String())
	}
}

func TestCatalogValidateFailsForInvalidMigrationMetadata(t *testing.T) {
	dir := t.TempDir()
	catalogPath := filepath.Join(dir, "catalog.json")
	auditPath := filepath.Join(dir, "audit.log.jsonl")
	metadataDir := filepath.Join(dir, "metadata")

	oldDescriptor := validTemplateDescriptor()
	oldDescriptor.Version = testVersion010
	oldDescriptor.Slug = fmt.Sprintf("%s/%s@%s", oldDescriptor.Plugin, oldDescriptor.Key, oldDescriptor.Version)
	oldPath := filepath.Join(dir, "old.json")
	writeTemplateFile(t, oldPath, oldDescriptor)

	newDescriptor := oldDescriptor
	newDescriptor.Version = testVersion020
	newDescriptor.Slug = fmt.Sprintf("%s/%s@%s", newDescriptor.Plugin, newDescriptor.Key, newDescriptor.Version)
	newPath := filepath.Join(dir, "new.json")
	writeTemplateFile(t, newPath, newDescriptor)

	for _, invocation := range [][]string{
		catalogArgs("add", catalogPath, auditPath, metadataDir, oldPath),
		catalogArgs("add", catalogPath, auditPath, metadataDir, newPath),
		catalogArgs("migrate", catalogPath, auditPath, metadataDir, oldDescriptor.Slug, newDescriptor.Slug),
	} {
		var stdout, stderr strings.Builder
		if code := cli(invocation, &stdout, &stderr); code != 0 {
			t.Fatalf("setup command failed code=%d stderr=%q", code, stderr.String())
		}
	}

	catalog := readCatalogRegistryForTests(t, catalogPath)
	if len(catalog.Migrations) != 1 {
		t.Fatalf("expected one migration record")
	}
	if err := os.WriteFile(catalog.Migrations[0].MetadataPath, []byte("{"), 0o600); err != nil {
		t.Fatalf("corrupt migration metadata: %v", err)
	}

	var stdout, stderr strings.Builder
	code := cli(catalogArgs("validate", catalogPath, auditPath, metadataDir), &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected validate to fail, got code=%d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "migration metadata") {
		t.Fatalf("expected migration metadata failure, got %q", stderr.String())
	}
}

func TestCatalogValidateFailsOnTamperedAuditLog(t *testing.T) {
	dir := t.TempDir()
	catalogPath := filepath.Join(dir, "catalog.json")
	auditPath := filepath.Join(dir, "audit.log.jsonl")
	metadataDir := filepath.Join(dir, "metadata")

	descriptor := validTemplateDescriptor()
	descriptor.Version = "1.0.0"
	descriptor.Slug = fmt.Sprintf("%s/%s@%s", descriptor.Plugin, descriptor.Key, descriptor.Version)
	path := filepath.Join(dir, "template.json")
	writeTemplateFile(t, path, descriptor)

	var stdout, stderr strings.Builder
	if code := cli(catalogArgs("add", catalogPath, auditPath, metadataDir, path), &stdout, &stderr); code != 0 {
		t.Fatalf("catalog add failed code=%d stderr=%q", code, stderr.String())
	}

	tampered := catalogAuditEntry{
		ID:        "tampered",
		Timestamp: time.Now().UTC(),
		Operation: "manual",
		Status:    catalogAuditStatusSuccess,
		Actor:     "tester",
		Catalog:   catalogPath,
		PrevHash:  "tampered-prev-hash",
	}
	tampered.Hash = catalogAuditHash(tampered)
	if err := appendCatalogAuditEntry(auditPath, tampered); err != nil {
		t.Fatalf("append tampered entry: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	code := cli(catalogArgs("validate", catalogPath, auditPath, metadataDir), &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected validate to fail for tampered audit log, got code=%d", code)
	}
	if !strings.Contains(stderr.String(), "audit log: FAIL") {
		t.Fatalf("expected audit log failure output, got %q", stderr.String())
	}
}

func TestBuildCatalogCompatibilityDetectsBreakingChanges(t *testing.T) {
	oldDesc := validTemplateDescriptor()
	oldDesc.Version = testVersion010
	oldDesc.Slug = fmt.Sprintf("%s/%s@%s", oldDesc.Plugin, oldDesc.Key, oldDesc.Version)
	newDesc := oldDesc
	newDesc.Version = testVersion020
	newDesc.Dialect = datasetapi.GetDialectProvider().SQL()
	newDesc.Parameters = []datasetapi.Parameter{{Name: "limit", Type: "string"}}
	newDesc.Columns = []datasetapi.Column{{Name: "value", Type: "integer"}}
	newDesc.OutputFormats = []datasetapi.Format{datasetapi.GetFormatProvider().JSON()}
	newDesc.Slug = fmt.Sprintf("%s/%s@%s", newDesc.Plugin, newDesc.Key, newDesc.Version)

	summary := buildCatalogCompatibility(oldDesc, newDesc)
	if !summary.Breaking {
		t.Fatalf("expected breaking compatibility summary")
	}
	if !summary.DialectChanged {
		t.Fatalf("expected dialect change to be detected")
	}
	if len(summary.ChangedParameterTypes) == 0 {
		t.Fatalf("expected parameter type changes")
	}
	if len(summary.RemovedColumns) == 0 {
		t.Fatalf("expected removed columns")
	}
	if len(summary.RemovedOutputFormats) == 0 {
		t.Fatalf("expected removed output formats")
	}

	steps := buildMigrationPlanSteps(summary)
	if len(steps) < 4 {
		t.Fatalf("expected additional migration steps for breaking change, got %d", len(steps))
	}
}

func TestResolveTemplateByReference(t *testing.T) {
	catalog := catalogRegistry{
		Templates: []catalogTemplateRecord{
			{Descriptor: datasetapi.TemplateDescriptor{Plugin: "frog", Key: "summary", Version: "1.0.0", Slug: "frog/summary@1.0.0"}},
			{Descriptor: datasetapi.TemplateDescriptor{Plugin: "frog", Key: "summary", Version: "1.1.0", Slug: "frog/summary@1.1.0"}},
		},
	}

	_, _, err := resolveTemplateByReference(catalog, "")
	if err == nil {
		t.Fatalf("expected empty reference to fail")
	}
	_, _, err = resolveTemplateByReference(catalog, "missing")
	if err == nil {
		t.Fatalf("expected missing template error")
	}
	_, _, err = resolveTemplateByReference(catalog, "summary")
	if err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("expected ambiguous key error, got %v", err)
	}
	record, idx, err := resolveTemplateByReference(catalog, "frog/summary@1.1.0")
	if err != nil {
		t.Fatalf("expected slug lookup success, got %v", err)
	}
	if idx != 1 || record.Descriptor.Slug != "frog/summary@1.1.0" {
		t.Fatalf("unexpected lookup result: idx=%d slug=%s", idx, record.Descriptor.Slug)
	}
}

func TestNormalizeCatalogDescriptor(t *testing.T) {
	descriptor := validTemplateDescriptor()
	descriptor.Slug = ""
	normalized, err := normalizeCatalogDescriptor(descriptor)
	if err != nil {
		t.Fatalf("normalize descriptor: %v", err)
	}
	expected := fmt.Sprintf("%s/%s@%s", descriptor.Plugin, descriptor.Key, descriptor.Version)
	if normalized.Slug != expected {
		t.Fatalf("expected slug %s, got %s", expected, normalized.Slug)
	}

	broken := descriptor
	broken.Plugin = ""
	if _, err := normalizeCatalogDescriptor(broken); err == nil {
		t.Fatalf("expected missing plugin validation error")
	}

	badSlug := descriptor
	badSlug.Slug = "wrong/slug@1"
	if _, err := normalizeCatalogDescriptor(badSlug); err == nil {
		t.Fatalf("expected slug mismatch validation error")
	}
}

func TestCatalogAuditChainHelpers(t *testing.T) {
	dir := t.TempDir()
	auditPath := filepath.Join(dir, "audit.log.jsonl")

	logger := catalogAuditLogger{path: auditPath, actor: "tester", catalog: filepath.Join(dir, "catalog.json"), timestampf: func() time.Time { return time.Now().UTC() }}
	if err := logger.Record("catalog_add", catalogAuditStatusSuccess, map[string]any{"template": "frog/summary@1"}, nil); err != nil {
		t.Fatalf("record audit entry: %v", err)
	}
	if err := logger.Record("catalog_validate", catalogAuditStatusSuccess, nil, nil); err != nil {
		t.Fatalf("record audit entry: %v", err)
	}
	if err := verifyCatalogAuditLogChain(auditPath); err != nil {
		t.Fatalf("expected audit log chain to be valid, got %v", err)
	}

	entries := readCatalogAuditEntries(t, auditPath)
	if len(entries) != 2 {
		t.Fatalf("expected 2 audit entries, got %d", len(entries))
	}
	if entries[0].Hash == "" || entries[1].PrevHash == "" {
		t.Fatalf("expected hash chaining fields to be populated")
	}

	entries[1].PrevHash = "tampered"
	entries[1].Hash = catalogAuditHash(entries[1])
	overwriteCatalogAuditEntries(t, auditPath, entries)
	if err := verifyCatalogAuditLogChain(auditPath); err == nil {
		t.Fatalf("expected tampered chain verification to fail")
	}
}

func TestCatalogAuditRecordFailsWhenLockIsHeld(t *testing.T) {
	dir := t.TempDir()
	auditPath := filepath.Join(dir, "audit.log.jsonl")
	lockPath := auditPath + ".lock"
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o750); err != nil {
		t.Fatalf("create lock directory: %v", err)
	}
	if err := os.WriteFile(lockPath, []byte("locked"), 0o600); err != nil {
		t.Fatalf("create lock file: %v", err)
	}

	prevTimeout := catalogLockTimeout
	prevInterval := catalogLockRetryInterval
	catalogLockTimeout = 100 * time.Millisecond
	catalogLockRetryInterval = 10 * time.Millisecond
	t.Cleanup(func() {
		catalogLockTimeout = prevTimeout
		catalogLockRetryInterval = prevInterval
	})

	logger := catalogAuditLogger{path: auditPath, actor: "tester", catalog: filepath.Join(dir, "catalog.json"), timestampf: func() time.Time { return time.Now().UTC() }}
	err := logger.Record("catalog_add", catalogAuditStatusSuccess, nil, nil)
	if err == nil {
		t.Fatalf("expected lock timeout error")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("expected timeout error, got %v", err)
	}
}

func catalogArgs(subcommand, catalogPath, auditPath, metadataDir string, tail ...string) []string {
	args := []string{"catalog", subcommand, "--catalog", catalogPath, "--audit-log", auditPath, "--metadata-dir", metadataDir, "--actor", "tester"}
	return append(args, tail...)
}

func readCatalogRegistryForTests(t *testing.T, path string) catalogRegistry {
	t.Helper()
	var catalog catalogRegistry
	readJSONFile(t, path, &catalog)
	return catalog
}

func readJSONFile(t *testing.T, path string, target any) {
	t.Helper()
	payload, err := os.ReadFile(path) // #nosec G304: test fixture path
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if err := json.Unmarshal(payload, target); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
}

func readCatalogAuditEntries(t *testing.T, path string) []catalogAuditEntry {
	t.Helper()
	file, err := os.Open(path) // #nosec G304: test fixture path
	if err != nil {
		t.Fatalf("open audit log: %v", err)
	}
	defer func() {
		_ = file.Close()
	}()

	entries := make([]catalogAuditEntry, 0, 4)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var entry catalogAuditEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("parse audit log entry %q: %v", line, err)
		}
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan audit log: %v", err)
	}
	return entries
}

func overwriteCatalogAuditEntries(t *testing.T, path string, entries []catalogAuditEntry) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("create audit dir: %v", err)
	}
	file, err := os.Create(path) // #nosec G304: test fixture path
	if err != nil {
		t.Fatalf("create audit log: %v", err)
	}
	defer func() {
		_ = file.Close()
	}()
	encoder := json.NewEncoder(file)
	for _, entry := range entries {
		if err := encoder.Encode(entry); err != nil {
			t.Fatalf("encode audit entry: %v", err)
		}
	}
}
