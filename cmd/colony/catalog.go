package main

import (
	"bufio"
	"bytes"
	"context"
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
	"strconv"
	"strings"
	"time"

	"colonycore/internal/observability"
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
	catalogRandomRead           = rand.Read
	catalogFilenameSanitizer    = regexp.MustCompile(`[^A-Za-z0-9_.-]+`)
	catalogLockTimeout          = 5 * time.Second
	catalogLockRetryInterval    = 25 * time.Millisecond
	catalogLockStaleAfter       = 30 * time.Minute
	catalogEventRecorderFactory = func(writer io.Writer) observability.Recorder {
		return observability.NewJSONRecorder(writer, "cmd.colony.catalog")
	}
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

type catalogLockMetadata struct {
	PID       int       `json:"pid"`
	Timestamp time.Time `json:"timestamp"`
}

type catalogFlags struct {
	catalogPath string
	auditPath   string
	metadataDir string
	actor       string
	emitEvents  bool
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

func catalogAddCLI(args []string, stdout, stderr io.Writer) (exitCode int) {
	var recorder observability.Recorder = observability.NoopRecorder{}
	started := time.Now()
	labels := map[string]string{
		"operation": "catalog_add",
	}
	measures := map[string]float64{}
	var opErr error
	defer func() {
		recordCatalogOperationEvent(recorder, "catalog.add", started, opErr, labels, measures)
	}()

	flagSet := flag.NewFlagSet("colony catalog add", flag.ContinueOnError)
	flagSet.SetOutput(stderr)
	flags := defaultCatalogFlags()
	registerCatalogFlags(flagSet, &flags)
	if err := flagSet.Parse(args); err != nil {
		opErr = err
		return 2
	}
	recorder = catalogRecorder(flags.emitEvents, stderr)
	labels["catalog_path"] = strings.TrimSpace(flags.catalogPath)
	labels["audit_log_path"] = strings.TrimSpace(flags.auditPath)
	if flagSet.NArg() != 1 {
		opErr = errors.New("expected exactly one template descriptor file")
		_, _ = fmt.Fprintln(stderr, "colony catalog add: expected exactly one template descriptor file")
		return 2
	}
	audit := newCatalogAuditLogger(flags)
	templatePath := flagSet.Arg(0)
	labels["template_path"] = templatePath

	descriptor, err := readTemplateDescriptor(templatePath)
	if err != nil {
		opErr = err
		_, _ = fmt.Fprintf(stderr, "colony catalog add: %v\n", err)
		if auditErr := writeCatalogAudit(audit, "catalog_add", catalogAuditStatusError, map[string]any{"template_path": templatePath}, err); auditErr != nil {
			reportCatalogAuditFailure(stderr, "colony catalog add", auditErr)
		}
		return 1
	}
	descriptor, err = normalizeCatalogDescriptor(descriptor)
	if err != nil {
		opErr = err
		_, _ = fmt.Fprintf(stderr, "colony catalog add: %v\n", err)
		if auditErr := writeCatalogAudit(audit, "catalog_add", catalogAuditStatusError, map[string]any{"template_path": templatePath}, err); auditErr != nil {
			reportCatalogAuditFailure(stderr, "colony catalog add", auditErr)
		}
		return 1
	}
	labels["template_id"] = descriptor.Slug

	if err := withCatalogPathLock(flags.catalogPath, func() error {
		catalog, err := loadCatalogRegistry(flags.catalogPath)
		if err != nil {
			return err
		}
		if _, idx, err := resolveTemplateByReference(catalog, descriptor.Slug); err == nil {
			return fmt.Errorf("template %s already exists in catalog", catalog.Templates[idx].Descriptor.Slug)
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
		measures["templates_total"] = float64(len(catalog.Templates))
		return saveCatalogRegistry(flags.catalogPath, catalog)
	}); err != nil {
		opErr = err
		_, _ = fmt.Fprintf(stderr, "colony catalog add: %v\n", err)
		if auditErr := writeCatalogAudit(audit, "catalog_add", catalogAuditStatusError, map[string]any{"template": descriptor.Slug}, err); auditErr != nil {
			reportCatalogAuditFailure(stderr, "colony catalog add", auditErr)
		}
		return 1
	}

	if err := writeCatalogAudit(audit, "catalog_add", catalogAuditStatusSuccess, map[string]any{"template": descriptor.Slug, "template_path": templatePath}, nil); err != nil {
		opErr = err
		reportCatalogAuditFailure(stderr, "colony catalog add", err)
		return 1
	}
	_, _ = fmt.Fprintf(stdout, "added template %s\n", descriptor.Slug)
	return 0
}

func catalogDeprecateCLI(args []string, stdout, stderr io.Writer) (exitCode int) {
	var recorder observability.Recorder = observability.NoopRecorder{}
	started := time.Now()
	labels := map[string]string{
		"operation": "catalog_deprecate",
	}
	measures := map[string]float64{}
	var opErr error
	defer func() {
		recordCatalogOperationEvent(recorder, "catalog.deprecate", started, opErr, labels, measures)
	}()

	flagSet := flag.NewFlagSet("colony catalog deprecate", flag.ContinueOnError)
	flagSet.SetOutput(stderr)
	flags := defaultCatalogFlags()
	registerCatalogFlags(flagSet, &flags)
	reason := flagSet.String("reason", "", "deprecation reason")
	windowDays := flagSet.Int("window-days", defaultDeprecationWindowInDays, "deprecation window in days")
	if err := flagSet.Parse(args); err != nil {
		opErr = err
		return 2
	}
	recorder = catalogRecorder(flags.emitEvents, stderr)
	labels["catalog_path"] = strings.TrimSpace(flags.catalogPath)
	labels["audit_log_path"] = strings.TrimSpace(flags.auditPath)
	if flagSet.NArg() != 1 {
		opErr = errors.New("expected a template key or slug")
		_, _ = fmt.Fprintln(stderr, "colony catalog deprecate: expected a template key or slug")
		return 2
	}
	if strings.TrimSpace(*reason) == "" {
		opErr = errors.New("deprecation reason is required")
		_, _ = fmt.Fprintln(stderr, "colony catalog deprecate: --reason is required")
		return 2
	}
	if *windowDays <= 0 {
		opErr = errors.New("deprecation window must be greater than zero")
		_, _ = fmt.Fprintln(stderr, "colony catalog deprecate: --window-days must be greater than zero")
		return 2
	}
	measures["window_days"] = float64(*windowDays)

	audit := newCatalogAuditLogger(flags)
	reference := flagSet.Arg(0)
	labels["template_ref"] = reference

	var deprecatedSlug string
	var sunset time.Time
	var metadataPath string
	if err := withCatalogPathLock(flags.catalogPath, func() error {
		catalog, err := loadCatalogRegistry(flags.catalogPath)
		if err != nil {
			return err
		}
		record, idx, err := resolveTemplateByReference(catalog, reference)
		if err != nil {
			return err
		}

		now := catalogNowFunc()
		sunset = now.AddDate(0, 0, *windowDays)
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
		metadataPath, err = writeCatalogMetadata(flags.metadataDir, catalogDeprecationMetadataPrefix, record.Descriptor.Slug+".json", metadata)
		if err != nil {
			return err
		}

		record.Deprecated = &catalogDeprecationWindow{
			Reason:       strings.TrimSpace(*reason),
			DeprecatedAt: now,
			SunsetAt:     sunset,
			MetadataPath: metadataPath,
		}
		catalog.Templates[idx] = record
		catalog.UpdatedAt = now
		deprecatedSlug = record.Descriptor.Slug
		measures["templates_total"] = float64(len(catalog.Templates))
		return saveCatalogRegistry(flags.catalogPath, catalog)
	}); err != nil {
		opErr = err
		_, _ = fmt.Fprintf(stderr, "colony catalog deprecate: %v\n", err)
		if auditErr := writeCatalogAudit(audit, "catalog_deprecate", catalogAuditStatusError, map[string]any{"template": reference}, err); auditErr != nil {
			reportCatalogAuditFailure(stderr, "colony catalog deprecate", auditErr)
		}
		return 1
	}
	labels["template_id"] = deprecatedSlug
	labels["metadata_path"] = metadataPath
	labels["sunset_at"] = sunset.Format(time.RFC3339)

	if err := writeCatalogAudit(audit, "catalog_deprecate", catalogAuditStatusSuccess, map[string]any{"template": deprecatedSlug, "reason": strings.TrimSpace(*reason), "sunset_at": sunset.Format(time.RFC3339), "metadata": metadataPath}, nil); err != nil {
		opErr = err
		reportCatalogAuditFailure(stderr, "colony catalog deprecate", err)
		return 1
	}
	_, _ = fmt.Fprintf(stdout, "deprecated template %s until %s\n", deprecatedSlug, sunset.Format(time.RFC3339))
	return 0
}

func catalogMigrateCLI(args []string, stdout, stderr io.Writer) (exitCode int) {
	var recorder observability.Recorder = observability.NoopRecorder{}
	started := time.Now()
	labels := map[string]string{
		"operation": "catalog_migrate",
	}
	measures := map[string]float64{}
	var opErr error
	defer func() {
		recordCatalogOperationEvent(recorder, "catalog.migrate", started, opErr, labels, measures)
	}()

	flagSet := flag.NewFlagSet("colony catalog migrate", flag.ContinueOnError)
	flagSet.SetOutput(stderr)
	flags := defaultCatalogFlags()
	registerCatalogFlags(flagSet, &flags)
	output := flagSet.String("output", "", "optional migration plan output path")
	if err := flagSet.Parse(args); err != nil {
		opErr = err
		return 2
	}
	recorder = catalogRecorder(flags.emitEvents, stderr)
	labels["catalog_path"] = strings.TrimSpace(flags.catalogPath)
	labels["audit_log_path"] = strings.TrimSpace(flags.auditPath)
	if flagSet.NArg() != 2 {
		opErr = errors.New("expected old and new template identifiers")
		_, _ = fmt.Fprintln(stderr, "colony catalog migrate: expected old and new template identifiers")
		return 2
	}
	oldRef := flagSet.Arg(0)
	newRef := flagSet.Arg(1)
	labels["old_ref"] = oldRef
	labels["new_ref"] = newRef
	audit := newCatalogAuditLogger(flags)

	var oldSlug string
	var newSlug string
	var metadataPath string
	var compatibility catalogCompatibilitySummary
	if err := withCatalogPathLock(flags.catalogPath, func() error {
		catalog, err := loadCatalogRegistry(flags.catalogPath)
		if err != nil {
			return err
		}
		oldRecord, _, oldErr := resolveTemplateByReference(catalog, oldRef)
		if oldErr != nil {
			return oldErr
		}
		newRecord, _, newErr := resolveTemplateByReference(catalog, newRef)
		if newErr != nil {
			return fmt.Errorf("new template: %w", newErr)
		}
		if oldRecord.Descriptor.Slug == newRecord.Descriptor.Slug {
			return errors.New("old and new template identifiers must differ")
		}

		now := catalogNowFunc()
		compatibility = buildCatalogCompatibility(oldRecord.Descriptor, newRecord.Descriptor)
		plan := catalogMigrationPlan{
			Kind:          "catalog_migration_plan",
			Version:       catalogSchemaVersion,
			From:          oldRecord.Descriptor.Slug,
			To:            newRecord.Descriptor.Slug,
			GeneratedAt:   now,
			Compatibility: compatibility,
			Steps:         buildMigrationPlanSteps(compatibility),
		}

		metadataPath = strings.TrimSpace(*output)
		if metadataPath == "" {
			defaultName := fmt.Sprintf("%s_to_%s_%s.json", slugFilename(oldRecord.Descriptor.Slug), slugFilename(newRecord.Descriptor.Slug), randomCatalogID())
			metadataPath, err = writeCatalogMetadata(flags.metadataDir, catalogMigrationMetadataPrefix, defaultName, plan)
		} else {
			err = writeJSONAtomically(metadataPath, plan)
		}
		if err != nil {
			return err
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
		oldSlug = oldRecord.Descriptor.Slug
		newSlug = newRecord.Descriptor.Slug
		measures["migrations_total"] = float64(len(catalog.Migrations))
		return saveCatalogRegistry(flags.catalogPath, catalog)
	}); err != nil {
		opErr = err
		_, _ = fmt.Fprintf(stderr, "colony catalog migrate: %v\n", err)
		if auditErr := writeCatalogAudit(audit, "catalog_migrate", catalogAuditStatusError, map[string]any{"old": oldRef, "new": newRef}, err); auditErr != nil {
			reportCatalogAuditFailure(stderr, "colony catalog migrate", auditErr)
		}
		return 1
	}
	labels["old_template_id"] = oldSlug
	labels["new_template_id"] = newSlug
	labels["metadata_path"] = metadataPath
	if compatibility.Breaking {
		labels["breaking_change"] = "true"
	} else {
		labels["breaking_change"] = "false"
	}

	if err := writeCatalogAudit(audit, "catalog_migrate", catalogAuditStatusSuccess, map[string]any{"old": oldSlug, "new": newSlug, "metadata": metadataPath, "breaking": compatibility.Breaking}, nil); err != nil {
		opErr = err
		reportCatalogAuditFailure(stderr, "colony catalog migrate", err)
		return 1
	}
	_, _ = fmt.Fprintf(stdout, "generated migration plan %s -> %s (%s)\n", oldSlug, newSlug, metadataPath)
	return 0
}

func catalogValidateCLI(args []string, stdout, stderr io.Writer) (exitCode int) {
	var recorder observability.Recorder = observability.NoopRecorder{}
	started := time.Now()
	labels := map[string]string{
		"operation": "catalog_validate",
	}
	measures := map[string]float64{}
	var opErr error
	defer func() {
		recordCatalogOperationEvent(recorder, "catalog.validate", started, opErr, labels, measures)
	}()

	flagSet := flag.NewFlagSet("colony catalog validate", flag.ContinueOnError)
	flagSet.SetOutput(stderr)
	flags := defaultCatalogFlags()
	registerCatalogFlags(flagSet, &flags)
	if err := flagSet.Parse(args); err != nil {
		opErr = err
		return 2
	}
	recorder = catalogRecorder(flags.emitEvents, stderr)
	labels["catalog_path"] = strings.TrimSpace(flags.catalogPath)
	labels["audit_log_path"] = strings.TrimSpace(flags.auditPath)
	if flagSet.NArg() > 0 {
		opErr = errors.New("unexpected positional arguments")
		_, _ = fmt.Fprintln(stderr, "colony catalog validate: unexpected positional arguments")
		return 2
	}
	audit := newCatalogAuditLogger(flags)

	catalog, err := loadCatalogRegistry(flags.catalogPath)
	if err != nil {
		opErr = err
		_, _ = fmt.Fprintf(stderr, "colony catalog validate: %v\n", err)
		if auditErr := writeCatalogAudit(audit, "catalog_validate", catalogAuditStatusError, nil, err); auditErr != nil {
			reportCatalogAuditFailure(stderr, "colony catalog validate", auditErr)
		}
		return 1
	}
	measures["templates_total"] = float64(len(catalog.Templates))
	measures["migrations_total"] = float64(len(catalog.Migrations))

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
			if strings.TrimSpace(record.Deprecated.MetadataPath) == "" {
				failures++
				_, _ = fmt.Fprintf(stderr, "%s: FAIL\n  deprecation metadata_path required\n", desc.Slug)
				continue
			}

			var metadata catalogDeprecationMetadata
			if err := decodeStrictJSONFile(record.Deprecated.MetadataPath, &metadata); err != nil {
				failures++
				_, _ = fmt.Fprintf(stderr, "%s: FAIL\n  deprecation metadata: %v\n", desc.Slug, err)
				continue
			}
			if metadata.Kind != "catalog_deprecation" {
				failures++
				_, _ = fmt.Fprintf(stderr, "%s: FAIL\n  deprecation metadata kind must equal catalog_deprecation\n", desc.Slug)
				continue
			}
			if metadata.TemplateSlug != desc.Slug {
				failures++
				_, _ = fmt.Fprintf(stderr, "%s: FAIL\n  deprecation metadata template slug mismatch\n", desc.Slug)
				continue
			}
			if metadata.DeprecatedAt.IsZero() || metadata.SunsetAt.IsZero() {
				failures++
				_, _ = fmt.Fprintf(stderr, "%s: FAIL\n  deprecation metadata timestamps are required\n", desc.Slug)
				continue
			}
		}
		_, _ = fmt.Fprintf(stdout, "%s: OK\n", desc.Slug)
	}

	for _, migration := range catalog.Migrations {
		ref := migration.ID
		if strings.TrimSpace(ref) == "" {
			ref = fmt.Sprintf("%s->%s", migration.From, migration.To)
		}
		if strings.TrimSpace(migration.MetadataPath) == "" {
			failures++
			_, _ = fmt.Fprintf(stderr, "%s: FAIL\n  migration metadata_path required\n", ref)
			continue
		}

		var plan catalogMigrationPlan
		if err := decodeStrictJSONFile(migration.MetadataPath, &plan); err != nil {
			failures++
			_, _ = fmt.Fprintf(stderr, "%s: FAIL\n  migration metadata: %v\n", ref, err)
			continue
		}
		if plan.Kind != "catalog_migration_plan" {
			failures++
			_, _ = fmt.Fprintf(stderr, "%s: FAIL\n  migration metadata kind must equal catalog_migration_plan\n", ref)
			continue
		}
		if plan.From != migration.From || plan.To != migration.To {
			failures++
			_, _ = fmt.Fprintf(stderr, "%s: FAIL\n  migration metadata endpoint mismatch\n", ref)
			continue
		}
		if plan.GeneratedAt.IsZero() {
			failures++
			_, _ = fmt.Fprintf(stderr, "%s: FAIL\n  migration metadata generated_at is required\n", ref)
			continue
		}
	}

	if err := verifyCatalogAuditLogChain(flags.auditPath); err != nil {
		failures++
		_, _ = fmt.Fprintf(stderr, "audit log: FAIL\n  %v\n", err)
	} else {
		_, _ = fmt.Fprintln(stdout, "audit log: OK")
	}

	if failures > 0 {
		opErr = fmt.Errorf("catalog validation failed with %d issue(s)", failures)
		_, _ = fmt.Fprintf(stderr, "catalog validation failed: %d issue(s)\n", failures)
		measures["failures_total"] = float64(failures)
		auditErr := writeCatalogAudit(audit, "catalog_validate", catalogAuditStatusError, map[string]any{"templates": len(catalog.Templates), "failures": failures}, opErr)
		if auditErr != nil {
			reportCatalogAuditFailure(stderr, "colony catalog validate", auditErr)
		}
		return 1
	}

	if err := writeCatalogAudit(audit, "catalog_validate", catalogAuditStatusSuccess, map[string]any{"templates": len(catalog.Templates), "failures": 0}, nil); err != nil {
		opErr = err
		reportCatalogAuditFailure(stderr, "colony catalog validate", err)
		return 1
	}
	measures["failures_total"] = 0
	_, _ = fmt.Fprintf(stdout, "catalog validation passed: %d template(s)\n", len(catalog.Templates))
	return 0
}

func defaultCatalogFlags() catalogFlags {
	return catalogFlags{
		catalogPath: defaultCatalogPath,
		auditPath:   defaultCatalogAuditLogPath,
		metadataDir: defaultCatalogMetadataPath,
		actor:       catalogActor(),
		emitEvents:  false,
	}
}

func registerCatalogFlags(flagSet *flag.FlagSet, flags *catalogFlags) {
	flagSet.StringVar(&flags.catalogPath, "catalog", defaultCatalogPath, "catalog registry path")
	flagSet.StringVar(&flags.auditPath, "audit-log", defaultCatalogAuditLogPath, "catalog audit log path")
	flagSet.StringVar(&flags.metadataDir, "metadata-dir", defaultCatalogMetadataPath, "catalog metadata output directory")
	flagSet.StringVar(&flags.actor, "actor", catalogActor(), "catalog actor identity")
	flagSet.BoolVar(&flags.emitEvents, "observability-json", false, "emit structured observability events as JSON lines to stderr")
}

func catalogRecorder(enabled bool, writer io.Writer) observability.Recorder {
	if !enabled {
		return observability.NoopRecorder{}
	}
	return catalogEventRecorderFactory(writer)
}

func newCatalogAuditLogger(flags catalogFlags) catalogAuditLogger {
	return catalogAuditLogger{
		path:       strings.TrimSpace(flags.auditPath),
		catalog:    strings.TrimSpace(flags.catalogPath),
		actor:      strings.TrimSpace(flags.actor),
		timestampf: catalogNowFunc,
	}
}

func withCatalogPathLock(path string, fn func() error) error {
	return withPathLock(path, fn)
}

func withPathLock(path string, fn func() error) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return errors.New("lock path must not be empty")
	}
	lockPath := path + ".lock"
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o750); err != nil {
		return fmt.Errorf("create lock directory: %w", err)
	}
	deadline := time.Now().Add(catalogLockTimeout)
	for {
		lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600) // #nosec G304: local operator-configured path
		if err == nil {
			if err := writeCatalogLockMetadata(lockFile, catalogNowFunc().UTC()); err != nil {
				_ = lockFile.Close()
				_ = os.Remove(lockPath)
				return err
			}
			defer func() {
				_ = lockFile.Close()
				_ = os.Remove(lockPath)
			}()
			return fn()
		}
		if !errors.Is(err, os.ErrExist) {
			return fmt.Errorf("acquire lock: %w", err)
		}
		stale, staleErr := isCatalogLockStale(lockPath, catalogNowFunc().UTC())
		if staleErr != nil {
			return staleErr
		}
		if stale {
			if err := os.Remove(lockPath); err != nil && !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("acquire lock: remove stale lock %s: %w", lockPath, err)
			}
			continue
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("acquire lock: timed out for %s", lockPath)
		}
		time.Sleep(catalogLockRetryInterval)
	}
}

func writeCatalogLockMetadata(lockFile *os.File, now time.Time) error {
	metadata := catalogLockMetadata{
		PID:       os.Getpid(),
		Timestamp: now,
	}
	payload, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("acquire lock: encode metadata: %w", err)
	}
	if _, err := lockFile.Write(append(payload, '\n')); err != nil {
		return fmt.Errorf("acquire lock: write metadata: %w", err)
	}
	return nil
}

func isCatalogLockStale(lockPath string, now time.Time) (bool, error) {
	payload, err := os.ReadFile(lockPath) // #nosec G304: local operator-configured path
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("acquire lock: read existing lock: %w", err)
	}

	var metadata catalogLockMetadata
	if err := json.Unmarshal(payload, &metadata); err != nil {
		fileInfo, statErr := os.Stat(lockPath)
		if statErr != nil {
			if errors.Is(statErr, os.ErrNotExist) {
				return false, nil
			}
			return false, fmt.Errorf("acquire lock: stat existing lock: %w", statErr)
		}
		return now.Sub(fileInfo.ModTime().UTC()) > catalogLockStaleAfter, nil
	}

	if metadata.Timestamp.IsZero() {
		fileInfo, statErr := os.Stat(lockPath)
		if statErr != nil {
			if errors.Is(statErr, os.ErrNotExist) {
				return false, nil
			}
			return false, fmt.Errorf("acquire lock: stat existing lock: %w", statErr)
		}
		metadata.Timestamp = fileInfo.ModTime().UTC()
	}
	if now.Sub(metadata.Timestamp.UTC()) > catalogLockStaleAfter {
		return true, nil
	}
	return !catalogProcessIsAlive(metadata.PID), nil
}

func catalogProcessIsAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	if _, err := os.Stat("/proc"); err != nil {
		return true
	}
	procPath := filepath.Join("/proc", strconv.Itoa(pid))
	_, err := os.Stat(procPath)
	return err == nil
}

func writeCatalogAudit(audit catalogAuditLogger, operation string, status catalogAuditStatus, details map[string]any, opErr error) error {
	return audit.Record(operation, status, details, opErr)
}

func reportCatalogAuditFailure(stderr io.Writer, command string, err error) {
	_, _ = fmt.Fprintf(stderr, "%s: unable to write catalog audit log: %v\n", command, err)
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
	return withPathLock(a.path, func() error {
		previousHash, err := readLastCatalogAuditHash(a.path)
		if err != nil {
			return err
		}
		entry.PrevHash = previousHash
		entry.Hash = catalogAuditHash(entry)
		return appendCatalogAuditEntry(a.path, entry)
	})
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
		} else if !oldParam.Required && newParam.Required {
			summary.ChangedParameterTypes = append(summary.ChangedParameterTypes, fmt.Sprintf("%s: optional -> required", oldParam.Name))
			summary.Breaking = true
		} else if oldParam.Required && !newParam.Required {
			summary.ChangedParameterTypes = append(summary.ChangedParameterTypes, fmt.Sprintf("%s: required -> optional", oldParam.Name))
		}
	}
	for key, newParam := range newParams {
		if _, ok := oldParams[key]; !ok {
			summary.AddedParameters = append(summary.AddedParameters, newParam.Name)
			if newParam.Required {
				summary.ChangedParameterTypes = append(summary.ChangedParameterTypes, fmt.Sprintf("%s: new required parameter", newParam.Name))
				summary.Breaking = true
			}
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

func decodeStrictJSONFile(path string, target any) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return errors.New("path must not be empty")
	}
	payload, err := os.ReadFile(path) // #nosec G304: local operator-configured path
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	if err := decoder.Decode(new(struct{})); err != io.EOF {
		return fmt.Errorf("parse %s: unexpected trailing data", path)
	}
	return nil
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

func recordCatalogOperationEvent(recorder observability.Recorder, name string, started time.Time, opErr error, labels map[string]string, measures map[string]float64) {
	if recorder == nil {
		recorder = observability.NoopRecorder{}
	}
	event := observability.Event{
		Category:   observability.CategoryCatalogOperation,
		Name:       name,
		Status:     observability.StatusSuccess,
		DurationMS: observability.DurationMS(time.Since(started)),
		Labels:     labels,
		Measures:   measures,
	}
	if opErr != nil {
		event.Status = observability.StatusError
		event.Error = opErr.Error()
	}
	recorder.Record(context.Background(), event)
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
	_, _ = fmt.Fprintln(w, "  colony catalog add [--catalog <path>] [--audit-log <path>] [--metadata-dir <path>] [--actor <id>] [--observability-json] <template.json>")
	_, _ = fmt.Fprintln(w, "  colony catalog deprecate [--catalog <path>] [--audit-log <path>] [--metadata-dir <path>] [--actor <id>] [--observability-json] --reason <text> [--window-days <n>] <key|slug>")
	_, _ = fmt.Fprintln(w, "  colony catalog migrate [--catalog <path>] [--audit-log <path>] [--metadata-dir <path>] [--actor <id>] [--observability-json] [--output <path>] <old> <new>")
	_, _ = fmt.Fprintln(w, "  colony catalog validate [--catalog <path>] [--audit-log <path>] [--metadata-dir <path>] [--actor <id>] [--observability-json]")
}
