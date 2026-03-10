package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"colonycore/pkg/datasetapi"
)

const (
	testCatalogFile = "catalog.json"
	testAuditFile   = "audit.log.jsonl"
	testMetadataDir = "metadata"
)

func TestMainUsesExitFunc(t *testing.T) {
	originalExit := exitFunc
	originalArgs := os.Args
	defer func() {
		exitFunc = originalExit
		os.Args = originalArgs
	}()

	var captured int
	exitFunc = func(code int) {
		captured = code
	}
	os.Args = []string{"colony", "lint", "dataset"}

	main()
	if captured != 2 {
		t.Fatalf("expected main to propagate exit code 2, got %d", captured)
	}
}

func TestLintCLIUnknownAndUsage(t *testing.T) {
	var stdout, stderr strings.Builder
	code := cli(nil, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("expected root cli with no args to return 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "Usage:") {
		t.Fatalf("expected root usage output, got %q", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = lintCLI([]string{"unknown"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("expected unknown lint command to return 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "unknown lint command") {
		t.Fatalf("expected unknown lint output, got %q", stderr.String())
	}

	stderr.Reset()
	printLintUsage(&stderr)
	if !strings.Contains(stderr.String(), "colony lint dataset") {
		t.Fatalf("expected lint usage output, got %q", stderr.String())
	}
}

func TestStringListFlagSetValidation(t *testing.T) {
	var nilValues *stringListFlag
	if got := nilValues.String(); got != "" {
		t.Fatalf("expected nil string list rendering to be empty, got %q", got)
	}

	var values stringListFlag
	if err := values.Set("   "); err == nil {
		t.Fatalf("expected empty list flag value to fail")
	}
	if err := values.Set("one"); err != nil {
		t.Fatalf("set first list flag value: %v", err)
	}
	if err := values.Set("two"); err != nil {
		t.Fatalf("set second list flag value: %v", err)
	}
	if got := values.String(); got != "one,two" {
		t.Fatalf("unexpected string list rendering: %s", got)
	}
}

func TestCatalogCLIUsageAndUnknown(t *testing.T) {
	var stdout, stderr strings.Builder
	if code := catalogCLI(nil, &stdout, &stderr); code != 2 {
		t.Fatalf("expected missing catalog subcommand to return 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "colony catalog add") {
		t.Fatalf("expected catalog usage text, got %q", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := catalogCLI([]string{"unknown"}, &stdout, &stderr); code != 2 {
		t.Fatalf("expected unknown catalog subcommand to return 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "unknown catalog command") {
		t.Fatalf("expected unknown catalog message, got %q", stderr.String())
	}
}

func TestCatalogCommandInputValidation(t *testing.T) {
	dir := t.TempDir()
	catalogPath := filepath.Join(dir, testCatalogFile)
	auditPath := filepath.Join(dir, testAuditFile)
	metadataDir := filepath.Join(dir, testMetadataDir)

	cases := []struct {
		name string
		args []string
	}{
		{name: "add-missing-path", args: catalogArgs("add", catalogPath, auditPath, metadataDir)},
		{name: "deprecate-missing-reason", args: catalogArgs("deprecate", catalogPath, auditPath, metadataDir, "frog/template@1")},
		{name: "deprecate-invalid-window", args: catalogArgs("deprecate", catalogPath, auditPath, metadataDir, "--reason", "r", "--window-days", "0", "frog/template@1")},
		{name: "migrate-missing-args", args: catalogArgs("migrate", catalogPath, auditPath, metadataDir, "frog/template@1")},
		{name: "validate-extra-arg", args: catalogArgs("validate", catalogPath, auditPath, metadataDir, "extra")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr strings.Builder
			if code := cli(tc.args, &stdout, &stderr); code != 2 {
				t.Fatalf("expected usage error code 2, got %d (stderr=%q)", code, stderr.String())
			}
		})
	}
}

func TestCatalogCommandFailurePaths(t *testing.T) {
	dir := t.TempDir()
	catalogPath := filepath.Join(dir, "catalog", testCatalogFile)
	auditPath := filepath.Join(dir, "catalog", testAuditFile)
	metadataDir := filepath.Join(dir, "catalog", testMetadataDir)

	t.Run("add-missing-file", func(t *testing.T) {
		var stdout, stderr strings.Builder
		code := cli(catalogArgs("add", catalogPath, auditPath, metadataDir, filepath.Join(dir, "missing.json")), &stdout, &stderr)
		if code != 1 {
			t.Fatalf("expected add failure code 1, got %d", code)
		}
	})

	t.Run("deprecate-not-found", func(t *testing.T) {
		var stdout, stderr strings.Builder
		code := cli(catalogArgs("deprecate", catalogPath, auditPath, metadataDir, "--reason", "r", "frog/missing@1"), &stdout, &stderr)
		if code != 1 {
			t.Fatalf("expected deprecate failure code 1, got %d", code)
		}
	})

	t.Run("migrate-same-template", func(t *testing.T) {
		descriptor := validTemplateDescriptor()
		descriptor.Version = "3.0.0"
		descriptor.Slug = fmt.Sprintf("%s/%s@%s", descriptor.Plugin, descriptor.Key, descriptor.Version)
		templatePath := filepath.Join(dir, "same.json")
		writeTemplateFile(t, templatePath, descriptor)

		var stdout, stderr strings.Builder
		if code := cli(catalogArgs("add", catalogPath, auditPath, metadataDir, templatePath), &stdout, &stderr); code != 0 {
			t.Fatalf("expected setup add to succeed, got %d stderr=%q", code, stderr.String())
		}
		stdout.Reset()
		stderr.Reset()
		code := cli(catalogArgs("migrate", catalogPath, auditPath, metadataDir, descriptor.Slug, descriptor.Slug), &stdout, &stderr)
		if code != 1 {
			t.Fatalf("expected migrate same-template failure, got %d", code)
		}
	})

	t.Run("validate-duplicate-slugs", func(t *testing.T) {
		dup := validTemplateDescriptor()
		dup.Slug = "frog/frog_population_snapshot@9.0.0"
		dup.Version = "9.0.0"
		catalog := catalogRegistry{
			Version:   catalogSchemaVersion,
			UpdatedAt: time.Now().UTC(),
			Templates: []catalogTemplateRecord{{Descriptor: dup}, {Descriptor: dup}},
		}
		if err := saveCatalogRegistry(catalogPath, catalog); err != nil {
			t.Fatalf("save duplicate catalog: %v", err)
		}
		var stdout, stderr strings.Builder
		code := cli(catalogArgs("validate", catalogPath, auditPath, metadataDir), &stdout, &stderr)
		if code != 1 {
			t.Fatalf("expected duplicate slug validation failure, got %d", code)
		}
		if !strings.Contains(stderr.String(), "duplicate template slug") {
			t.Fatalf("expected duplicate slug output, got %q", stderr.String())
		}
	})
}

func TestCatalogAddErrorBranches(t *testing.T) {
	dir := t.TempDir()
	catalogPath := filepath.Join(dir, "catalog", testCatalogFile)
	auditPath := filepath.Join(dir, "catalog", testAuditFile)
	metadataDir := filepath.Join(dir, "catalog", testMetadataDir)

	descriptor := validTemplateDescriptor()
	templatePath := filepath.Join(dir, "template.json")
	writeTemplateFile(t, templatePath, descriptor)

	if err := os.MkdirAll(filepath.Dir(catalogPath), 0o750); err != nil {
		t.Fatalf("mkdir catalog dir: %v", err)
	}
	if err := os.WriteFile(catalogPath, []byte("{"), 0o600); err != nil {
		t.Fatalf("write invalid catalog file: %v", err)
	}
	var stdout, stderr strings.Builder
	code := cli(catalogArgs("add", catalogPath, auditPath, metadataDir, templatePath), &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected add to fail for invalid catalog, got %d", code)
	}

	descriptor.Plugin = ""
	badPath := filepath.Join(dir, "template-bad.json")
	writeTemplateFile(t, badPath, descriptor)
	stdout.Reset()
	stderr.Reset()
	code = cli(catalogArgs("add", filepath.Join(dir, "fresh", testCatalogFile), auditPath, metadataDir, badPath), &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected add to fail for invalid descriptor, got %d", code)
	}
	if !strings.Contains(stderr.String(), "template plugin is required") {
		t.Fatalf("expected plugin validation error, got %q", stderr.String())
	}
}

func TestCatalogDeprecateMetadataFailure(t *testing.T) {
	dir := t.TempDir()
	catalogPath := filepath.Join(dir, testCatalogFile)
	auditPath := filepath.Join(dir, testAuditFile)
	metadataDir := filepath.Join(dir, testMetadataDir)

	descriptor := validTemplateDescriptor()
	descriptor.Version = "4.0.0"
	descriptor.Slug = fmt.Sprintf("%s/%s@%s", descriptor.Plugin, descriptor.Key, descriptor.Version)
	templatePath := filepath.Join(dir, "template.json")
	writeTemplateFile(t, templatePath, descriptor)

	var stdout, stderr strings.Builder
	if code := cli(catalogArgs("add", catalogPath, auditPath, metadataDir, templatePath), &stdout, &stderr); code != 0 {
		t.Fatalf("setup add failed code=%d stderr=%q", code, stderr.String())
	}

	blocking := filepath.Join(dir, "blocking")
	if err := os.WriteFile(blocking, []byte("x"), 0o600); err != nil {
		t.Fatalf("write blocking file: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	code := cli(catalogArgs("deprecate", catalogPath, auditPath, filepath.Join(blocking, "meta"), "--reason", "replace", descriptor.Slug), &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected deprecate metadata write failure, got %d", code)
	}
}

func TestCatalogMigrateAdditionalBranches(t *testing.T) {
	dir := t.TempDir()
	catalogPath := filepath.Join(dir, testCatalogFile)
	auditPath := filepath.Join(dir, testAuditFile)
	metadataDir := filepath.Join(dir, testMetadataDir)

	base := validTemplateDescriptor()
	base.Version = "5.0.0"
	base.Slug = fmt.Sprintf("%s/%s@%s", base.Plugin, base.Key, base.Version)
	basePath := filepath.Join(dir, "base.json")
	writeTemplateFile(t, basePath, base)

	next := base
	next.Version = "5.1.0"
	next.Slug = fmt.Sprintf("%s/%s@%s", next.Plugin, next.Key, next.Version)
	nextPath := filepath.Join(dir, "next.json")
	writeTemplateFile(t, nextPath, next)

	var stdout, stderr strings.Builder
	if code := cli(catalogArgs("add", catalogPath, auditPath, metadataDir, basePath), &stdout, &stderr); code != 0 {
		t.Fatalf("setup add base failed code=%d stderr=%q", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := cli(catalogArgs("add", catalogPath, auditPath, metadataDir, nextPath), &stdout, &stderr); code != 0 {
		t.Fatalf("setup add next failed code=%d stderr=%q", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code := cli(catalogArgs("migrate", catalogPath, auditPath, metadataDir, base.Slug, "frog/missing@1"), &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected migrate missing new template failure, got %d", code)
	}

	outputPath := filepath.Join(dir, "plans", "migration.json")
	stdout.Reset()
	stderr.Reset()
	code = cli(catalogArgs("migrate", catalogPath, auditPath, metadataDir, "--output", outputPath, base.Slug, next.Slug), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected migrate with explicit output to succeed, got %d stderr=%q", code, stderr.String())
	}
	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("expected migration output path to exist, got %v", err)
	}
}

func TestCatalogValidateDeprecationWindowFailures(t *testing.T) {
	dir := t.TempDir()
	catalogPath := filepath.Join(dir, testCatalogFile)
	auditPath := filepath.Join(dir, testAuditFile)
	metadataDir := filepath.Join(dir, testMetadataDir)

	desc := validTemplateDescriptor()
	desc.Version = "7.0.0"
	desc.Slug = fmt.Sprintf("%s/%s@%s", desc.Plugin, desc.Key, desc.Version)
	now := time.Now().UTC()
	registry := catalogRegistry{
		Version:   catalogSchemaVersion,
		UpdatedAt: now,
		Templates: []catalogTemplateRecord{{
			Descriptor: desc,
			Deprecated: &catalogDeprecationWindow{
				Reason:       "",
				DeprecatedAt: now,
				SunsetAt:     now,
				MetadataPath: filepath.Join(metadataDir, "dep.json"),
			},
		}},
	}
	if err := saveCatalogRegistry(catalogPath, registry); err != nil {
		t.Fatalf("save catalog: %v", err)
	}

	var stdout, stderr strings.Builder
	code := cli(catalogArgs("validate", catalogPath, auditPath, metadataDir), &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected deprecation validation to fail, got %d", code)
	}
	if !strings.Contains(stderr.String(), "deprecation reason required") {
		t.Fatalf("expected deprecation reason failure, got %q", stderr.String())
	}

	registry.Templates[0].Deprecated.Reason = "valid"
	registry.Templates[0].Deprecated.SunsetAt = now.Add(-time.Minute)
	if err := saveCatalogRegistry(catalogPath, registry); err != nil {
		t.Fatalf("save catalog: %v", err)
	}
	stdout.Reset()
	stderr.Reset()
	code = cli(catalogArgs("validate", catalogPath, auditPath, metadataDir), &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected deprecation sunset validation to fail, got %d", code)
	}
	if !strings.Contains(stderr.String(), "sunset must be after") {
		t.Fatalf("expected deprecation sunset failure, got %q", stderr.String())
	}
}

func TestCatalogFileHelpers(t *testing.T) {
	dir := t.TempDir()

	if err := writeJSONAtomically("", map[string]string{"x": "y"}); err == nil {
		t.Fatalf("expected empty path write to fail")
	}

	metaPath, err := writeCatalogMetadata(filepath.Join(dir, "meta"), "sub", "frog/frog_population_snapshot@1.0.0", map[string]string{"k": "v"})
	if err != nil {
		t.Fatalf("write metadata: %v", err)
	}
	if !strings.HasSuffix(metaPath, ".json") {
		t.Fatalf("expected metadata path to end with .json, got %s", metaPath)
	}
	if _, err := os.Stat(metaPath); err != nil {
		t.Fatalf("expected metadata file to exist, got %v", err)
	}

	invalidPath := filepath.Join(dir, "invalid.json")
	if err := os.WriteFile(invalidPath, []byte("{"), 0o600); err != nil {
		t.Fatalf("write invalid catalog: %v", err)
	}
	if _, err := loadCatalogRegistry(invalidPath); err == nil {
		t.Fatalf("expected invalid JSON catalog parse to fail")
	}

	trailingPath := filepath.Join(dir, "trailing.json")
	if err := os.WriteFile(trailingPath, []byte("{}{}"), 0o600); err != nil {
		t.Fatalf("write trailing catalog: %v", err)
	}
	if _, err := loadCatalogRegistry(trailingPath); err == nil {
		t.Fatalf("expected trailing catalog parse to fail")
	}

	descriptorPath := filepath.Join(dir, "descriptor.json")
	if err := os.WriteFile(descriptorPath, []byte("{}{}"), 0o600); err != nil {
		t.Fatalf("write trailing descriptor: %v", err)
	}
	if _, err := readTemplateDescriptor(descriptorPath); err == nil {
		t.Fatalf("expected trailing descriptor parse to fail")
	}

	blocking := filepath.Join(dir, "blocking")
	if err := os.WriteFile(blocking, []byte("x"), 0o600); err != nil {
		t.Fatalf("write blocking file: %v", err)
	}
	if err := writeJSONAtomically(filepath.Join(blocking, "out.json"), map[string]string{"x": "y"}); err == nil {
		t.Fatalf("expected writeJSONAtomically to fail when parent is a file")
	}
}

func TestCatalogSortingAndUtilityHelpers(t *testing.T) {
	ts := time.Unix(100, 0).UTC()
	migrations := []catalogMigrationRecord{
		{From: "b", To: "a", CreatedAt: ts},
		{From: "a", To: "b", CreatedAt: ts},
		{From: "c", To: "a", CreatedAt: ts.Add(time.Second)},
	}
	sortMigrationRecords(migrations)
	if migrations[0].From != "a" || migrations[0].To != "b" {
		t.Fatalf("unexpected migration sort order: %+v", migrations)
	}
	if migrations[1].From != "b" || migrations[2].From != "c" {
		t.Fatalf("unexpected migration sort order: %+v", migrations)
	}

	if got := slugFilename(" frog/alpha@1.0.0 "); got != "frog_alpha_1.0.0" {
		t.Fatalf("unexpected slug filename: %s", got)
	}
	if got := slugFilename("***"); got != "catalog" {
		t.Fatalf("unexpected slug filename fallback: %s", got)
	}
	if got := randomCatalogID(); got == "" {
		t.Fatalf("expected random catalog id")
	}
	if got := formatSet([]datasetapi.Format{datasetapi.GetFormatProvider().JSON()}); len(got) != 1 {
		t.Fatalf("expected one format in set, got %d", len(got))
	}
	if got := cloneAnyMap(nil); got != nil {
		t.Fatalf("expected nil map clone for nil input")
	}

	var usage strings.Builder
	printCatalogUsage(&usage)
	if !strings.Contains(usage.String(), "colony catalog migrate") {
		t.Fatalf("expected catalog usage text, got %q", usage.String())
	}
}

func TestRandomCatalogIDFallback(t *testing.T) {
	originalReader := catalogRandomRead
	defer func() {
		catalogRandomRead = originalReader
	}()
	catalogRandomRead = func([]byte) (int, error) {
		return 0, fmt.Errorf("forced random failure")
	}

	if got := randomCatalogID(); !strings.HasPrefix(got, "catalog-") {
		t.Fatalf("expected fallback random id prefix, got %s", got)
	}
}

func TestCatalogAuditWarningAndVerificationErrors(t *testing.T) {
	dir := t.TempDir()

	blockingFile := filepath.Join(dir, "blocking")
	if err := os.WriteFile(blockingFile, []byte("x"), 0o600); err != nil {
		t.Fatalf("write blocking file: %v", err)
	}
	invalidLogger := catalogAuditLogger{path: filepath.Join(blockingFile, "audit.log"), actor: "tester", catalog: filepath.Join(dir, testCatalogFile)}
	var stderr strings.Builder
	writeCatalogAudit(&stderr, invalidLogger, "catalog_add", catalogAuditStatusSuccess, nil, nil)
	if !strings.Contains(stderr.String(), "warning: unable to write catalog audit log") {
		t.Fatalf("expected audit warning output, got %q", stderr.String())
	}

	auditPath := filepath.Join(dir, "audit.log")
	if err := os.WriteFile(auditPath, []byte("not-json\n"), 0o600); err != nil {
		t.Fatalf("write invalid audit log: %v", err)
	}
	if err := verifyCatalogAuditLogChain(auditPath); err == nil {
		t.Fatalf("expected invalid audit log parsing to fail")
	}

	if err := appendCatalogAuditEntry(filepath.Join(blockingFile, "log.jsonl"), catalogAuditEntry{}); err == nil {
		t.Fatalf("expected append catalog audit to fail when parent is a file")
	}

	hashPath := filepath.Join(dir, "hash.log")
	if got, err := readLastCatalogAuditHash(hashPath); err != nil || got != "" {
		t.Fatalf("expected missing audit hash to be empty, got hash=%q err=%v", got, err)
	}
	if err := os.WriteFile(hashPath, []byte("\n"), 0o600); err != nil {
		t.Fatalf("write empty audit log: %v", err)
	}
	if got, err := readLastCatalogAuditHash(hashPath); err != nil || got != "" {
		t.Fatalf("expected empty audit hash to be empty, got hash=%q err=%v", got, err)
	}
	if err := os.WriteFile(hashPath, []byte("{\"hash\":\"\"}\n"), 0o600); err != nil {
		t.Fatalf("write hash-missing audit log: %v", err)
	}
	if _, err := readLastCatalogAuditHash(hashPath); err == nil {
		t.Fatalf("expected missing hash to fail")
	}
	if err := os.WriteFile(hashPath, []byte("not-json\n"), 0o600); err != nil {
		t.Fatalf("write malformed audit log: %v", err)
	}
	if _, err := readLastCatalogAuditHash(hashPath); err == nil {
		t.Fatalf("expected malformed hash tail to fail")
	}
}
