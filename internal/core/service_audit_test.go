package core

import (
	"context"
	"strings"
	"testing"
	"time"

	memory "colonycore/internal/infra/persistence/memory"
	"colonycore/pkg/datasetapi"
	"colonycore/pkg/domain"
	"colonycore/pkg/pluginapi"
)

func TestRecordAuditSuccessUsesMetadata(t *testing.T) {
	fixed := time.Date(2024, 10, 1, 8, 30, 0, 0, time.UTC)
	recorder := &auditRecorderStub{}
	store := clockOverrideStore{Store: NewMemoryStore(NewDefaultRulesEngine())}
	svc := NewService(
		store,
		WithAuditRecorder(recorder),
		WithClock(ClockFunc(func() time.Time { return fixed })),
	)

	entityID := "project-123"
	duration := 42 * time.Millisecond
	svc.recordAuditSuccess(context.Background(), "create_project", entityID, duration)

	if len(recorder.entries) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(recorder.entries))
	}
	entry := recorder.entries[0]
	if entry.Operation != "create_project" {
		t.Fatalf("unexpected operation: %s", entry.Operation)
	}
	if entry.Entity != domain.EntityProject {
		t.Fatalf("expected entity project, got %s", entry.Entity)
	}
	if entry.Action != domain.ActionCreate {
		t.Fatalf("expected create action, got %s", entry.Action)
	}
	if entry.EntityID != entityID {
		t.Fatalf("expected entity id %s, got %s", entityID, entry.EntityID)
	}
	if entry.Status != AuditStatusSuccess {
		t.Fatalf("expected success status, got %s", entry.Status)
	}
	if entry.Duration != duration {
		t.Fatalf("expected duration %v, got %v", duration, entry.Duration)
	}
	if !entry.Timestamp.Equal(fixed) {
		t.Fatalf("expected timestamp %v, got %v", fixed, entry.Timestamp)
	}
}

func TestRecordAuditSuccessIgnoresUnknownOperation(t *testing.T) {
	recorder := &auditRecorderStub{}
	store := clockOverrideStore{Store: NewMemoryStore(NewDefaultRulesEngine())}
	svc := NewService(
		store,
		WithAuditRecorder(recorder),
	)

	svc.recordAuditSuccess(context.Background(), "unknown_operation", "entity", time.Second)

	if len(recorder.entries) != 0 {
		t.Fatalf("expected no audit entries for unknown operation, got %d", len(recorder.entries))
	}
}

func TestResolveDatasetTemplateMissing(t *testing.T) {
	store := clockOverrideStore{Store: NewMemoryStore(NewDefaultRulesEngine())}
	svc := NewService(store)

	runtime, ok := svc.ResolveDatasetTemplate("missing")
	if ok {
		t.Fatalf("expected missing template to return ok=false")
	}
	if runtime != nil {
		t.Fatalf("expected nil runtime for missing template")
	}
}

func TestInstallPluginDatasetConflict(t *testing.T) {
	store := clockOverrideStore{Store: NewMemoryStore(NewDefaultRulesEngine())}
	svc := NewService(store)
	if _, err := svc.InstallPlugin(datasetConflictPlugin{}); err == nil {
		t.Fatalf("expected conflict plugin installation to fail")
	} else if !strings.Contains(err.Error(), "already registered") {
		t.Fatalf("expected dataset conflict error, got %v", err)
	}
}

func TestNoopImplementations(t *testing.T) {
	var logger noopLogger
	logger.Debug("noop")
	logger.Info("noop")
	logger.Warn("noop")
	logger.Error("noop")

	var audit noopAuditRecorder
	audit.Record(context.Background(), AuditEntry{})

	var metrics noopMetricsRecorder
	metrics.Observe(context.Background(), "noop", true, 0)

	tracer := noopTracer{}
	ctx, span := tracer.Start(context.Background(), "op")
	if ctx == nil {
		t.Fatalf("expected context from tracer")
	}
	span.End(nil)
}

type auditRecorderStub struct {
	entries []AuditEntry
}

func (r *auditRecorderStub) Record(_ context.Context, entry AuditEntry) {
	r.entries = append(r.entries, entry)
}

type clockOverrideStore struct {
	*memory.Store
}

func (clockOverrideStore) NowFunc() func() time.Time {
	return nil
}

type datasetConflictPlugin struct{}

func (datasetConflictPlugin) Name() string    { return "conflict" }
func (datasetConflictPlugin) Version() string { return "v1" }

func (datasetConflictPlugin) Register(reg pluginapi.Registry) error {
	tpl := testDatasetTemplate()
	if err := reg.RegisterDatasetTemplate(tpl); err != nil {
		return err
	}
	return reg.RegisterDatasetTemplate(tpl)
}

func testDatasetTemplate() datasetapi.Template {
	dialect := datasetapi.GetDialectProvider()
	formats := datasetapi.GetFormatProvider()
	return datasetapi.Template{
		Key:         "summary",
		Version:     "v1",
		Title:       "Summary",
		Description: "Test dataset",
		Dialect:     dialect.SQL(),
		Query:       "SELECT 1",
		Columns: []datasetapi.Column{{
			Name: "value",
			Type: "integer",
		}},
		OutputFormats: []datasetapi.Format{formats.JSON()},
		Binder: func(datasetapi.Environment) (datasetapi.Runner, error) {
			return func(context.Context, datasetapi.RunRequest) (datasetapi.RunResult, error) {
				return datasetapi.RunResult{
					Schema:      []datasetapi.Column{{Name: "value", Type: "integer"}},
					Rows:        []datasetapi.Row{{"value": 1}},
					GeneratedAt: time.Now().UTC(),
					Format:      formats.JSON(),
				}, nil
			}, nil
		},
	}
}
