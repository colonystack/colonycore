package main

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"colonycore/pkg/datasetapi"
)

const (
	defaultCatalogPath               = "catalog/catalog.json"
	defaultCatalogAuditLogPath       = "catalog/audit.log.jsonl"
	defaultCatalogMetadataPath       = "catalog/metadata"
	catalogSchemaVersion             = "v1"
	defaultDeprecationWindowInDays   = 90
	catalogDeprecationMetadataPrefix = "deprecations"
	catalogMigrationMetadataPrefix   = "migrations"
)

var (
	catalogNowFunc = func() time.Time { return time.Now().UTC() }
	catalogActor   = func() string {
		if actor := strings.TrimSpace(os.Getenv("COLONY_ACTOR")); actor != "" {
			return actor
		}
		if actor := strings.TrimSpace(os.Getenv("USER")); actor != "" {
			return actor
		}
		if actor := strings.TrimSpace(os.Getenv("USERNAME")); actor != "" {
			return actor
		}
		return "unknown"
	}
	catalogRandomRead        = rand.Read
	catalogFilenameSanitizer = regexp.MustCompile(`[^A-Za-z0-9_.-]+`)
)

type catalogRegistry struct {
	Version    string                   `json:"version"`
	UpdatedAt  time.Time                `json:"updated_at"`
	Templates  []catalogTemplateRecord  `json:"templates"`
	Migrations []catalogMigrationRecord `json:"migrations,omitempty"`
}

type catalogTemplateRecord struct {
	Descriptor datasetapi.TemplateDescriptor `json:"descriptor"`
	AddedAt    time.Time                     `json:"added_at"`
	AddedBy    string                        `json:"added_by,omitempty"`
	Deprecated *catalogDeprecationWindow     `json:"deprecated,omitempty"`
}

type catalogDeprecationWindow struct {
	Reason       string    `json:"reason"`
	DeprecatedAt time.Time `json:"deprecated_at"`
	SunsetAt     time.Time `json:"sunset_at"`
	MetadataPath string    `json:"metadata_path"`
}

type catalogMigrationRecord struct {
	ID            string                      `json:"id"`
	From          string                      `json:"from"`
	To            string                      `json:"to"`
	CreatedAt     time.Time                   `json:"created_at"`
	MetadataPath  string                      `json:"metadata_path"`
	Compatibility catalogCompatibilitySummary `json:"compatibility"`
}

type catalogCompatibilitySummary struct {
	Breaking              bool     `json:"breaking"`
	DialectChanged        bool     `json:"dialect_changed"`
	AddedParameters       []string `json:"added_parameters,omitempty"`
	RemovedParameters     []string `json:"removed_parameters,omitempty"`
	ChangedParameterTypes []string `json:"changed_parameter_types,omitempty"`
	AddedColumns          []string `json:"added_columns,omitempty"`
	RemovedColumns        []string `json:"removed_columns,omitempty"`
	ChangedColumnTypes    []string `json:"changed_column_types,omitempty"`
	AddedOutputFormats    []string `json:"added_output_formats,omitempty"`
	RemovedOutputFormats  []string `json:"removed_output_formats,omitempty"`
	Notes                 []string `json:"notes,omitempty"`
}

type catalogMigrationPlan struct {
	Kind          string                      `json:"kind"`
	Version       string                      `json:"version"`
	From          string                      `json:"from"`
	To            string                      `json:"to"`
	GeneratedAt   time.Time                   `json:"generated_at"`
	Compatibility catalogCompatibilitySummary `json:"compatibility"`
	Steps         []catalogMigrationPlanStep  `json:"steps"`
}

type catalogMigrationPlanStep struct {
	ID          string `json:"id"`
	Required    bool   `json:"required"`
	Description string `json:"description"`
}

type catalogDeprecationMetadata struct {
	Kind            string    `json:"kind"`
	Version         string    `json:"version"`
	TemplateSlug    string    `json:"template_slug"`
	Plugin          string    `json:"plugin"`
	Key             string    `json:"key"`
	TemplateVersion string    `json:"template_version"`
	Reason          string    `json:"reason"`
	DeprecatedAt    time.Time `json:"deprecated_at"`
	SunsetAt        time.Time `json:"sunset_at"`
}

type catalogAuditStatus string

const (
	catalogAuditStatusSuccess catalogAuditStatus = "success"
	catalogAuditStatusError   catalogAuditStatus = "error"
)

type catalogAuditEntry struct {
	ID        string             `json:"id"`
	Timestamp time.Time          `json:"timestamp"`
	Operation string             `json:"operation"`
	Status    catalogAuditStatus `json:"status"`
	Actor     string             `json:"actor"`
	Catalog   string             `json:"catalog"`
	Details   map[string]any     `json:"details,omitempty"`
	Error     string             `json:"error,omitempty"`
	PrevHash  string             `json:"prev_hash,omitempty"`
	Hash      string             `json:"hash"`
}

type catalogAuditLogger struct {
	path       string
	catalog    string
	actor      string
	timestampf func() time.Time
}

type catalogFlags struct {
	catalogPath string
	auditPath   string
	metadataDir string
	actor       string
}

func catalogCLI(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printCatalogUsage(stderr)
		return 2
	}

	switch args[0] {
	case "add":
		return catalogAddCLI(args[1:], stdout, stderr)
	case "deprecate":
		return catalogDeprecateCLI(args[1:], stdout, stderr)
	case "migrate":
		return catalogMigrateCLI(args[1:], stdout, stderr)
	case "validate":
		return catalogValidateCLI(args[1:], stdout, stderr)
	default:
		_, _ = fmt.Fprintf(stderr, "unknown catalog command %q\n", args[0])
		printCatalogUsage(stderr)
		return 2
	}
}

func catalogAddCLI(args []string, stdout, stderr io.Writer) int {
	flagSet := flag.NewFlagSet("colony catalog add", flag.ContinueOnError)
	flagSet.SetOutput(stderr)
	flags := defaultCatalogFlags()
	registerCatalogFlags(flagSet, &flags)
	if err := flagSet.Parse(args); err != nil {
		return 2
	}
	if flagSet.NArg() != 1 {
		_, _ = fmt.Fprintln(stderr, "colony catalog add: expected exactly one template descriptor file")
		return 2
	}
	audit := newCatalogAuditLogger(flags)
	templatePath := flagSet.Arg(0)

	descriptor, err := readTemplateDescriptor(templatePath)
	if err != nil {
		writeCatalogAudit(stderr, audit, "catalog_add", catalogAuditStatusError, map[string]any{"template_path": templatePath}, err)
		_, _ = fmt.Fprintf(stderr, "colony catalog add: %v\n", err)
		return 1
	}
	descriptor, err = normalizeCatalogDescriptor(descriptor)
	if err != nil {
		writeCatalogAudit(stderr, audit, "catalog_add", catalogAuditStatusError, map[string]any{"template_path": templatePath}, err)
		_, _ = fmt.Fprintf(stderr, "colony catalog add: %v\n", err)
		return 1
	}

	catalog, err := loadCatalogRegistry(flags.catalogPath)
	if err != nil {
		writeCatalogAudit(stderr, audit, "catalog_add", catalogAuditStatusError, map[string]any{"template": descriptor.Slug}, err)
		_, _ = fmt.Fprintf(stderr, "colony catalog add: %v\n", err)
		return 1
	}
	if _, idx, err := resolveTemplateByReference(catalog, descriptor.Slug); err == nil {
		err = fmt.Errorf("template %s already exists in catalog", catalog.Templates[idx].Descriptor.Slug)
		writeCatalogAudit(stderr, audit, "catalog_add", catalogAuditStatusError, map[string]any{"template": descriptor.Slug}, err)
		_, _ = fmt.Fprintf(stderr, "colony catalog add: %v\n", err)
		return 1
	}

	now := catalogNowFunc()
	catalog.Templates = append(catalog.Templates, catalogTemplateRecord{
		Descriptor: descriptor,
		AddedAt:    now,
		AddedBy:    strings.TrimSpace(flags.actor),
	})
	sortTemplateRecords(catalog.Templates)
	catalog.Version = catalogSchemaVersion
	catalog.UpdatedAt = now
	if err := saveCatalogRegistry(flags.catalogPath, catalog); err != nil {
		writeCatalogAudit(stderr, audit, "catalog_add", catalogAuditStatusError, map[string]any{"template": descriptor.Slug}, err)
		_, _ = fmt.Fprintf(stderr, "colony catalog add: %v\n", err)
		return 1
	}

	writeCatalogAudit(stderr, audit, "catalog_add", catalogAuditStatusSuccess, map[string]any{"template": descriptor.Slug, "template_path": templatePath}, nil)
	_, _ = fmt.Fprintf(stdout, "added template %s\n", descriptor.Slug)
	return 0
}

func catalogDeprecateCLI(args []string, stdout, stderr io.Writer) int {
	flagSet := flag.NewFlagSet("colony catalog deprecate", flag.ContinueOnError)
	flagSet.SetOutput(stderr)
	flags := defaultCatalogFlags()
	registerCatalogFlags(flagSet, &flags)
	reason := flagSet.String("reason", "", "deprecation reason")
	windowDays := flagSet.Int("window-days", defaultDeprecationWindowInDays, "deprecation window in days")
	if err := flagSet.Parse(args); err != nil {
		return 2
	}
	if flagSet.NArg() != 1 {
		_, _ = fmt.Fprintln(stderr, "colony catalog deprecate: expected a template key or slug")
		return 2
	}
	if strings.TrimSpace(*reason) == "" {
		_, _ = fmt.Fprintln(stderr, "colony catalog deprecate: --reason is required")
		return 2
	}
	if *windowDays <= 0 {
		_, _ = fmt.Fprintln(stderr, "colony catalog deprecate: --window-days must be greater than zero")
		return 2
	}

	audit := newCatalogAuditLogger(flags)
	reference := flagSet.Arg(0)

	catalog, err := loadCatalogRegistry(flags.catalogPath)
	if err != nil {
		writeCatalogAudit(stderr, audit, "catalog_deprecate", catalogAuditStatusError, map[string]any{"template": reference}, err)
		_, _ = fmt.Fprintf(stderr, "colony catalog deprecate: %v\n", err)
		return 1
	}
	record, idx, err := resolveTemplateByReference(catalog, reference)
	if err != nil {
		writeCatalogAudit(stderr, audit, "catalog_deprecate", catalogAuditStatusError, map[string]any{"template": reference}, err)
		_, _ = fmt.Fprintf(stderr, "colony catalog deprecate: %v\n", err)
		return 1
	}

	now := catalogNowFunc()
	sunset := now.AddDate(0, 0, *windowDays)
	metadata := catalogDeprecationMetadata{
		Kind:            "catalog_deprecation",
		Version:         catalogSchemaVersion,
		TemplateSlug:    record.Descriptor.Slug,
		Plugin:          record.Descriptor.Plugin,
		Key:             record.Descriptor.Key,
		TemplateVersion: record.Descriptor.Version,
		Reason:          strings.TrimSpace(*reason),
		DeprecatedAt:    now,
		SunsetAt:        sunset,
	}
	metadataPath, err := writeCatalogMetadata(flags.metadataDir, catalogDeprecationMetadataPrefix, record.Descriptor.Slug+".json", metadata)
	if err != nil {
		writeCatalogAudit(stderr, audit, "catalog_deprecate", catalogAuditStatusError, map[string]any{"template": record.Descriptor.Slug}, err)
		_, _ = fmt.Fprintf(stderr, "colony catalog deprecate: %v\n", err)
		return 1
	}

	record.Deprecated = &catalogDeprecationWindow{
		Reason:       strings.TrimSpace(*reason),
		DeprecatedAt: now,
		SunsetAt:     sunset,
		MetadataPath: metadataPath,
	}
	catalog.Templates[idx] = record
	catalog.UpdatedAt = now
	if err := saveCatalogRegistry(flags.catalogPath, catalog); err != nil {
		writeCatalogAudit(stderr, audit, "catalog_deprecate", catalogAuditStatusError, map[string]any{"template": record.Descriptor.Slug}, err)
		_, _ = fmt.Fprintf(stderr, "colony catalog deprecate: %v\n", err)
		return 1
	}

	writeCatalogAudit(stderr, audit, "catalog_deprecate", catalogAuditStatusSuccess, map[string]any{"template": record.Descriptor.Slug, "reason": strings.TrimSpace(*reason), "sunset_at": sunset.Format(time.RFC3339), "metadata": metadataPath}, nil)
	_, _ = fmt.Fprintf(stdout, "deprecated template %s until %s\n", record.Descriptor.Slug, sunset.Format(time.RFC3339))
	return 0
}

func catalogMigrateCLI(args []string, stdout, stderr io.Writer) int {
	flagSet := flag.NewFlagSet("colony catalog migrate", flag.ContinueOnError)
	flagSet.SetOutput(stderr)
	flags := defaultCatalogFlags()
	registerCatalogFlags(flagSet, &flags)
	output := flagSet.String("output", "", "optional migration plan output path")
	if err := flagSet.Parse(args); err != nil {
		return 2
	}
	if flagSet.NArg() != 2 {
		_, _ = fmt.Fprintln(stderr, "colony catalog migrate: expected old and new template identifiers")
		return 2
	}
	oldRef := flagSet.Arg(0)
	newRef := flagSet.Arg(1)
	audit := newCatalogAuditLogger(flags)

	catalog, err := loadCatalogRegistry(flags.catalogPath)
	if err != nil {
		writeCatalogAudit(stderr, audit, "catalog_migrate", catalogAuditStatusError, map[string]any{"old": oldRef, "new": newRef}, err)
		_, _ = fmt.Fprintf(stderr, "colony catalog migrate: %v\n", err)
		return 1
	}
	oldRecord, _, oldErr := resolveTemplateByReference(catalog, oldRef)
	if oldErr != nil {
		writeCatalogAudit(stderr, audit, "catalog_migrate", catalogAuditStatusError, map[string]any{"old": oldRef, "new": newRef}, oldErr)
		_, _ = fmt.Fprintf(stderr, "colony catalog migrate: %v\n", oldErr)
		return 1
	}
	newRecord, _, newErr := resolveTemplateByReference(catalog, newRef)
	if newErr != nil {
		err := fmt.Errorf("new template: %w", newErr)
		writeCatalogAudit(stderr, audit, "catalog_migrate", catalogAuditStatusError, map[string]any{"old": oldRef, "new": newRef}, err)
		_, _ = fmt.Fprintf(stderr, "colony catalog migrate: %v\n", err)
		return 1
	}
	if oldRecord.Descriptor.Slug == newRecord.Descriptor.Slug {
		err := errors.New("old and new template identifiers must differ")
		writeCatalogAudit(stderr, audit, "catalog_migrate", catalogAuditStatusError, map[string]any{"old": oldRef, "new": newRef}, err)
		_, _ = fmt.Fprintf(stderr, "colony catalog migrate: %v\n", err)
		return 1
	}

	now := catalogNowFunc()
	compatibility := buildCatalogCompatibility(oldRecord.Descriptor, newRecord.Descriptor)
	plan := catalogMigrationPlan{
		Kind:          "catalog_migration_plan",
		Version:       catalogSchemaVersion,
		From:          oldRecord.Descriptor.Slug,
		To:            newRecord.Descriptor.Slug,
		GeneratedAt:   now,
		Compatibility: compatibility,
		Steps:         buildMigrationPlanSteps(compatibility),
	}

	metadataPath := strings.TrimSpace(*output)
	if metadataPath == "" {
		defaultName := fmt.Sprintf("%s_to_%s.json", slugFilename(oldRecord.Descriptor.Slug), slugFilename(newRecord.Descriptor.Slug))
		metadataPath, err = writeCatalogMetadata(flags.metadataDir, catalogMigrationMetadataPrefix, defaultName, plan)
	} else {
		err = writeJSONAtomically(metadataPath, plan)
	}
	if err != nil {
		writeCatalogAudit(stderr, audit, "catalog_migrate", catalogAuditStatusError, map[string]any{"old": oldRecord.Descriptor.Slug, "new": newRecord.Descriptor.Slug}, err)
		_, _ = fmt.Fprintf(stderr, "colony catalog migrate: %v\n", err)
		return 1
	}

	catalog.Migrations = append(catalog.Migrations, catalogMigrationRecord{
		ID:            randomCatalogID(),
		From:          oldRecord.Descriptor.Slug,
		To:            newRecord.Descriptor.Slug,
		CreatedAt:     now,
		MetadataPath:  metadataPath,
		Compatibility: compatibility,
	})
	sortMigrationRecords(catalog.Migrations)
	catalog.UpdatedAt = now
	if err := saveCatalogRegistry(flags.catalogPath, catalog); err != nil {
		writeCatalogAudit(stderr, audit, "catalog_migrate", catalogAuditStatusError, map[string]any{"old": oldRecord.Descriptor.Slug, "new": newRecord.Descriptor.Slug}, err)
		_, _ = fmt.Fprintf(stderr, "colony catalog migrate: %v\n", err)
		return 1
	}

	writeCatalogAudit(stderr, audit, "catalog_migrate", catalogAuditStatusSuccess, map[string]any{"old": oldRecord.Descriptor.Slug, "new": newRecord.Descriptor.Slug, "metadata": metadataPath, "breaking": compatibility.Breaking}, nil)
	_, _ = fmt.Fprintf(stdout, "generated migration plan %s -> %s (%s)\n", oldRecord.Descriptor.Slug, newRecord.Descriptor.Slug, metadataPath)
	return 0
}

func catalogValidateCLI(args []string, stdout, stderr io.Writer) int {
	flagSet := flag.NewFlagSet("colony catalog validate", flag.ContinueOnError)
	flagSet.SetOutput(stderr)
	flags := defaultCatalogFlags()
	registerCatalogFlags(flagSet, &flags)
	if err := flagSet.Parse(args); err != nil {
		return 2
	}
	if flagSet.NArg() > 0 {
		_, _ = fmt.Fprintln(stderr, "colony catalog validate: unexpected positional arguments")
		return 2
	}
	audit := newCatalogAuditLogger(flags)

	catalog, err := loadCatalogRegistry(flags.catalogPath)
	if err != nil {
		writeCatalogAudit(stderr, audit, "catalog_validate", catalogAuditStatusError, nil, err)
		_, _ = fmt.Fprintf(stderr, "colony catalog validate: %v\n", err)
		return 1
	}

	failures := 0
	seen := make(map[string]struct{}, len(catalog.Templates))
	for _, record := range catalog.Templates {
		desc := record.Descriptor
		desc, normalizeErr := normalizeCatalogDescriptor(desc)
		if normalizeErr != nil {
			failures++
			_, _ = fmt.Fprintf(stderr, "%s: FAIL\n  %v\n", desc.Slug, normalizeErr)
			continue
		}
		if _, dup := seen[desc.Slug]; dup {
			failures++
			_, _ = fmt.Fprintf(stderr, "%s: FAIL\n  duplicate template slug\n", desc.Slug)
			continue
		}
		seen[desc.Slug] = struct{}{}
		if err := datasetapi.ValidateTemplateDescriptor(desc); err != nil {
			failures++
			reportLintFailure(stderr, desc.Slug, err)
			continue
		}
		if record.Deprecated != nil {
			if strings.TrimSpace(record.Deprecated.Reason) == "" {
				failures++
				_, _ = fmt.Fprintf(stderr, "%s: FAIL\n  deprecation reason required\n", desc.Slug)
				continue
			}
			if !record.Deprecated.SunsetAt.After(record.Deprecated.DeprecatedAt) {
				failures++
				_, _ = fmt.Fprintf(stderr, "%s: FAIL\n  deprecation sunset must be after deprecation timestamp\n", desc.Slug)
				continue
			}
		}
		_, _ = fmt.Fprintf(stdout, "%s: OK\n", desc.Slug)
	}

	if err := verifyCatalogAuditLogChain(flags.auditPath); err != nil {
		failures++
		_, _ = fmt.Fprintf(stderr, "audit log: FAIL\n  %v\n", err)
	} else {
		_, _ = fmt.Fprintln(stdout, "audit log: OK")
	}

	if failures > 0 {
		writeCatalogAudit(stderr, audit, "catalog_validate", catalogAuditStatusError, map[string]any{"templates": len(catalog.Templates), "failures": failures}, fmt.Errorf("catalog validation failed with %d issue(s)", failures))
		_, _ = fmt.Fprintf(stderr, "catalog validation failed: %d issue(s)\n", failures)
		return 1
	}

	writeCatalogAudit(stderr, audit, "catalog_validate", catalogAuditStatusSuccess, map[string]any{"templates": len(catalog.Templates), "failures": 0}, nil)
	_, _ = fmt.Fprintf(stdout, "catalog validation passed: %d template(s)\n", len(catalog.Templates))
	return 0
}

func defaultCatalogFlags() catalogFlags {
	return catalogFlags{
		catalogPath: defaultCatalogPath,
		auditPath:   defaultCatalogAuditLogPath,
		metadataDir: defaultCatalogMetadataPath,
		actor:       catalogActor(),
	}
}

func registerCatalogFlags(flagSet *flag.FlagSet, flags *catalogFlags) {
	flagSet.StringVar(&flags.catalogPath, "catalog", defaultCatalogPath, "catalog registry path")
	flagSet.StringVar(&flags.auditPath, "audit-log", defaultCatalogAuditLogPath, "catalog audit log path")
	flagSet.StringVar(&flags.metadataDir, "metadata-dir", defaultCatalogMetadataPath, "catalog metadata output directory")
	flagSet.StringVar(&flags.actor, "actor", catalogActor(), "catalog actor identity")
}

func newCatalogAuditLogger(flags catalogFlags) catalogAuditLogger {
	return catalogAuditLogger{
		path:       strings.TrimSpace(flags.auditPath),
		catalog:    strings.TrimSpace(flags.catalogPath),
		actor:      strings.TrimSpace(flags.actor),
		timestampf: catalogNowFunc,
	}
}

func writeCatalogAudit(stderr io.Writer, audit catalogAuditLogger, operation string, status catalogAuditStatus, details map[string]any, opErr error) {
	if err := audit.Record(operation, status, details, opErr); err != nil {
		_, _ = fmt.Fprintf(stderr, "warning: unable to write catalog audit log: %v\n", err)
	}
}

func (a catalogAuditLogger) Record(operation string, status catalogAuditStatus, details map[string]any, operationErr error) error {
	if strings.TrimSpace(a.path) == "" {
		return nil
	}
	entry := catalogAuditEntry{
		ID:        randomCatalogID(),
		Timestamp: time.Now().UTC(),
		Operation: operation,
		Status:    status,
		Actor:     strings.TrimSpace(a.actor),
		Catalog:   strings.TrimSpace(a.catalog),
		Details:   cloneAnyMap(details),
	}
	if a.timestampf != nil {
		entry.Timestamp = a.timestampf().UTC()
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}
	if operationErr != nil {
		entry.Error = operationErr.Error()
	}
	if entry.Actor == "" {
		entry.Actor = "unknown"
	}
	previousHash, err := readLastCatalogAuditHash(a.path)
	if err != nil {
		return err
	}
	entry.PrevHash = previousHash
	entry.Hash = catalogAuditHash(entry)
	return appendCatalogAuditEntry(a.path, entry)
}

func catalogAuditHash(entry catalogAuditEntry) string {
	payload := struct {
		ID        string             `json:"id"`
		Timestamp string             `json:"timestamp"`
		Operation string             `json:"operation"`
		Status    catalogAuditStatus `json:"status"`
		Actor     string             `json:"actor"`
		Catalog   string             `json:"catalog"`
		Details   map[string]any     `json:"details,omitempty"`
		Error     string             `json:"error,omitempty"`
		PrevHash  string             `json:"prev_hash,omitempty"`
	}{
		ID:        entry.ID,
		Timestamp: entry.Timestamp.UTC().Format(time.RFC3339Nano),
		Operation: entry.Operation,
		Status:    entry.Status,
		Actor:     entry.Actor,
		Catalog:   entry.Catalog,
		Details:   cloneAnyMap(entry.Details),
		Error:     entry.Error,
		PrevHash:  entry.PrevHash,
	}
	raw, _ := json.Marshal(payload)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func appendCatalogAuditEntry(path string, entry catalogAuditEntry) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("create audit directory: %w", err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600) // #nosec G304: local operator-configured path
	if err != nil {
		return fmt.Errorf("open audit log: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()
	encoder := json.NewEncoder(file)
	if err := encoder.Encode(entry); err != nil {
		return fmt.Errorf("write audit log: %w", err)
	}
	return nil
}

func readLastCatalogAuditHash(path string) (string, error) {
	file, err := os.Open(path) // #nosec G304: local operator-configured path
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", fmt.Errorf("read audit log: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	scanner := bufio.NewScanner(file)
	lastLine := ""
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		lastLine = line
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("scan audit log: %w", err)
	}
	if lastLine == "" {
		return "", nil
	}
	var entry catalogAuditEntry
	if err := json.Unmarshal([]byte(lastLine), &entry); err != nil {
		return "", fmt.Errorf("parse audit log tail: %w", err)
	}
	if strings.TrimSpace(entry.Hash) == "" {
		return "", errors.New("audit log tail missing hash")
	}
	return entry.Hash, nil
}

func verifyCatalogAuditLogChain(path string) error {
	file, err := os.Open(path) // #nosec G304: local operator-configured path
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("open audit log: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	scanner := bufio.NewScanner(file)
	lineNo := 0
	previous := ""
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var entry catalogAuditEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return fmt.Errorf("parse audit log line %d: %w", lineNo, err)
		}
		expected := catalogAuditHash(entry)
		if entry.Hash != expected {
			return fmt.Errorf("audit log line %d hash mismatch", lineNo)
		}
		if entry.PrevHash != previous {
			return fmt.Errorf("audit log line %d prev_hash mismatch", lineNo)
		}
		previous = entry.Hash
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan audit log: %w", err)
	}
	return nil
}

func loadCatalogRegistry(path string) (catalogRegistry, error) {
	payload, err := os.ReadFile(path) // #nosec G304: local operator-configured path
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return catalogRegistry{Version: catalogSchemaVersion}, nil
		}
		return catalogRegistry{}, fmt.Errorf("read catalog registry: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	var catalog catalogRegistry
	if err := decoder.Decode(&catalog); err != nil {
		return catalogRegistry{}, fmt.Errorf("parse catalog registry: %w", err)
	}
	if err := decoder.Decode(new(struct{})); err != io.EOF {
		return catalogRegistry{}, fmt.Errorf("parse catalog registry: unexpected trailing data")
	}
	if strings.TrimSpace(catalog.Version) == "" {
		catalog.Version = catalogSchemaVersion
	}
	return catalog, nil
}

func saveCatalogRegistry(path string, catalog catalogRegistry) error {
	catalog.Version = catalogSchemaVersion
	sortTemplateRecords(catalog.Templates)
	sortMigrationRecords(catalog.Migrations)
	return writeJSONAtomically(path, catalog)
}

func writeJSONAtomically(path string, value any) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("output path must not be empty")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	payload, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal JSON: %w", err)
	}
	payload = append(payload, '\n')

	tmpFile, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-")
	if err != nil {
		return fmt.Errorf("create temporary file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()
	if _, err := tmpFile.Write(payload); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("write temporary file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temporary file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace output file: %w", err)
	}
	return nil
}

func writeCatalogMetadata(root, subDir, filename string, payload any) (string, error) {
	path := filepath.Join(root, subDir, slugFilename(filename))
	if !strings.HasSuffix(path, ".json") {
		path += ".json"
	}
	if err := writeJSONAtomically(path, payload); err != nil {
		return "", err
	}
	return path, nil
}

func buildCatalogCompatibility(oldDesc, newDesc datasetapi.TemplateDescriptor) catalogCompatibilitySummary {
	summary := catalogCompatibilitySummary{}
	if oldDesc.Dialect != newDesc.Dialect {
		summary.DialectChanged = true
		summary.Breaking = true
		summary.Notes = append(summary.Notes, fmt.Sprintf("dialect changed from %s to %s", oldDesc.Dialect, newDesc.Dialect))
	}

	oldParams := make(map[string]datasetapi.Parameter, len(oldDesc.Parameters))
	newParams := make(map[string]datasetapi.Parameter, len(newDesc.Parameters))
	for _, param := range oldDesc.Parameters {
		oldParams[strings.ToLower(param.Name)] = param
	}
	for _, param := range newDesc.Parameters {
		newParams[strings.ToLower(param.Name)] = param
	}
	for key, oldParam := range oldParams {
		if newParam, ok := newParams[key]; !ok {
			summary.RemovedParameters = append(summary.RemovedParameters, oldParam.Name)
			summary.Breaking = true
		} else if oldParam.Type != newParam.Type {
			summary.ChangedParameterTypes = append(summary.ChangedParameterTypes, fmt.Sprintf("%s: %s -> %s", oldParam.Name, oldParam.Type, newParam.Type))
			summary.Breaking = true
		}
	}
	for key, newParam := range newParams {
		if _, ok := oldParams[key]; !ok {
			summary.AddedParameters = append(summary.AddedParameters, newParam.Name)
		}
	}

	oldColumns := make(map[string]datasetapi.Column, len(oldDesc.Columns))
	newColumns := make(map[string]datasetapi.Column, len(newDesc.Columns))
	for _, column := range oldDesc.Columns {
		oldColumns[strings.ToLower(column.Name)] = column
	}
	for _, column := range newDesc.Columns {
		newColumns[strings.ToLower(column.Name)] = column
	}
	for key, oldColumn := range oldColumns {
		if newColumn, ok := newColumns[key]; !ok {
			summary.RemovedColumns = append(summary.RemovedColumns, oldColumn.Name)
			summary.Breaking = true
		} else if oldColumn.Type != newColumn.Type {
			summary.ChangedColumnTypes = append(summary.ChangedColumnTypes, fmt.Sprintf("%s: %s -> %s", oldColumn.Name, oldColumn.Type, newColumn.Type))
			summary.Breaking = true
		}
	}
	for key, newColumn := range newColumns {
		if _, ok := oldColumns[key]; !ok {
			summary.AddedColumns = append(summary.AddedColumns, newColumn.Name)
		}
	}

	oldFormats := formatSet(oldDesc.OutputFormats)
	newFormats := formatSet(newDesc.OutputFormats)
	for format := range oldFormats {
		if _, ok := newFormats[format]; !ok {
			summary.RemovedOutputFormats = append(summary.RemovedOutputFormats, string(format))
			summary.Breaking = true
		}
	}
	for format := range newFormats {
		if _, ok := oldFormats[format]; !ok {
			summary.AddedOutputFormats = append(summary.AddedOutputFormats, string(format))
		}
	}

	sort.Strings(summary.AddedParameters)
	sort.Strings(summary.RemovedParameters)
	sort.Strings(summary.ChangedParameterTypes)
	sort.Strings(summary.AddedColumns)
	sort.Strings(summary.RemovedColumns)
	sort.Strings(summary.ChangedColumnTypes)
	sort.Strings(summary.AddedOutputFormats)
	sort.Strings(summary.RemovedOutputFormats)
	sort.Strings(summary.Notes)
	return summary
}

func buildMigrationPlanSteps(summary catalogCompatibilitySummary) []catalogMigrationPlanStep {
	steps := []catalogMigrationPlanStep{
		{ID: "validate-target", Required: true, Description: "Validate target template descriptor before rollout."},
		{ID: "dual-run", Required: true, Description: "Run old and new templates in parallel for one release cycle."},
		{ID: "switch-consumers", Required: true, Description: "Update downstream consumers to use the new template slug."},
	}
	if summary.Breaking {
		steps = append(steps, catalogMigrationPlanStep{ID: "breaking-review", Required: true, Description: "Address breaking changes listed in compatibility report before cutover."})
	}
	if len(summary.RemovedParameters) > 0 || len(summary.ChangedParameterTypes) > 0 {
		steps = append(steps, catalogMigrationPlanStep{ID: "parameter-mapping", Required: true, Description: "Map removed or type-changed parameters to supported replacements."})
	}
	if len(summary.RemovedColumns) > 0 || len(summary.ChangedColumnTypes) > 0 {
		steps = append(steps, catalogMigrationPlanStep{ID: "consumer-schema-update", Required: true, Description: "Update consumers for removed or changed output columns."})
	}
	steps = append(steps, catalogMigrationPlanStep{ID: "deprecate-source", Required: false, Description: "Deprecate source template with a documented sunset window after migration completes."})
	return steps
}

func resolveTemplateByReference(catalog catalogRegistry, reference string) (catalogTemplateRecord, int, error) {
	key := strings.TrimSpace(reference)
	if key == "" {
		return catalogTemplateRecord{}, -1, errors.New("template reference must not be empty")
	}
	for idx, record := range catalog.Templates {
		if record.Descriptor.Slug == key {
			return record, idx, nil
		}
	}
	matches := make([]int, 0, 1)
	for idx, record := range catalog.Templates {
		if record.Descriptor.Key == key {
			matches = append(matches, idx)
		}
	}
	if len(matches) == 1 {
		idx := matches[0]
		return catalog.Templates[idx], idx, nil
	}
	if len(matches) > 1 {
		ambiguous := make([]string, 0, len(matches))
		for _, idx := range matches {
			ambiguous = append(ambiguous, catalog.Templates[idx].Descriptor.Slug)
		}
		sort.Strings(ambiguous)
		return catalogTemplateRecord{}, -1, fmt.Errorf("template key %s is ambiguous; use one of: %s", key, strings.Join(ambiguous, ", "))
	}
	return catalogTemplateRecord{}, -1, fmt.Errorf("template %s not found", key)
}

func normalizeCatalogDescriptor(descriptor datasetapi.TemplateDescriptor) (datasetapi.TemplateDescriptor, error) {
	descriptor.Plugin = strings.TrimSpace(descriptor.Plugin)
	descriptor.Key = strings.TrimSpace(descriptor.Key)
	descriptor.Version = strings.TrimSpace(descriptor.Version)
	if descriptor.Plugin == "" {
		return descriptor, errors.New("template plugin is required")
	}
	if descriptor.Key == "" {
		return descriptor, errors.New("template key is required")
	}
	if descriptor.Version == "" {
		return descriptor, errors.New("template version is required")
	}
	expectedSlug := fmt.Sprintf("%s/%s@%s", descriptor.Plugin, descriptor.Key, descriptor.Version)
	if strings.TrimSpace(descriptor.Slug) == "" {
		descriptor.Slug = expectedSlug
	}
	if descriptor.Slug != expectedSlug {
		return descriptor, fmt.Errorf("template slug %q must equal %q", descriptor.Slug, expectedSlug)
	}
	if err := datasetapi.ValidateTemplateDescriptor(descriptor); err != nil {
		return descriptor, err
	}
	return descriptor, nil
}

func readTemplateDescriptor(path string) (datasetapi.TemplateDescriptor, error) {
	payload, err := os.ReadFile(path) // #nosec G304: local operator-supplied path
	if err != nil {
		return datasetapi.TemplateDescriptor{}, fmt.Errorf("read template descriptor: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	var descriptor datasetapi.TemplateDescriptor
	if err := decoder.Decode(&descriptor); err != nil {
		return datasetapi.TemplateDescriptor{}, fmt.Errorf("parse template descriptor: %w", err)
	}
	if err := decoder.Decode(new(struct{})); err != io.EOF {
		return datasetapi.TemplateDescriptor{}, fmt.Errorf("parse template descriptor: unexpected trailing data")
	}
	return descriptor, nil
}

func formatSet(formats []datasetapi.Format) map[datasetapi.Format]struct{} {
	set := make(map[datasetapi.Format]struct{}, len(formats))
	for _, format := range formats {
		set[format] = struct{}{}
	}
	return set
}

func sortTemplateRecords(records []catalogTemplateRecord) {
	sort.Slice(records, func(i, j int) bool {
		left := records[i].Descriptor
		right := records[j].Descriptor
		if left.Plugin == right.Plugin {
			if left.Key == right.Key {
				return left.Version < right.Version
			}
			return left.Key < right.Key
		}
		return left.Plugin < right.Plugin
	})
}

func sortMigrationRecords(records []catalogMigrationRecord) {
	sort.Slice(records, func(i, j int) bool {
		if records[i].CreatedAt.Equal(records[j].CreatedAt) {
			if records[i].From == records[j].From {
				return records[i].To < records[j].To
			}
			return records[i].From < records[j].From
		}
		return records[i].CreatedAt.Before(records[j].CreatedAt)
	})
}

func cloneAnyMap(values map[string]any) map[string]any {
	if len(values) == 0 {
		return nil
	}
	cloned := make(map[string]any, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func slugFilename(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return commandCatalog
	}
	filename := catalogFilenameSanitizer.ReplaceAllString(value, "_")
	filename = strings.Trim(filename, "_")
	if filename == "" {
		return commandCatalog
	}
	return filename
}

func randomCatalogID() string {
	buf := make([]byte, 16)
	if _, err := catalogRandomRead(buf); err != nil {
		return fmt.Sprintf("catalog-%d", time.Now().UTC().UnixNano())
	}
	return hex.EncodeToString(buf)
}

func printCatalogUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, "Usage:")
	_, _ = fmt.Fprintln(w, "  colony catalog add [--catalog <path>] [--audit-log <path>] [--metadata-dir <path>] [--actor <id>] <template.json>")
	_, _ = fmt.Fprintln(w, "  colony catalog deprecate [--catalog <path>] [--audit-log <path>] [--metadata-dir <path>] [--actor <id>] --reason <text> [--window-days <n>] <key|slug>")
	_, _ = fmt.Fprintln(w, "  colony catalog migrate [--catalog <path>] [--audit-log <path>] [--metadata-dir <path>] [--actor <id>] [--output <path>] <old> <new>")
	_, _ = fmt.Fprintln(w, "  colony catalog validate [--catalog <path>] [--audit-log <path>] [--metadata-dir <path>] [--actor <id>]")
}
