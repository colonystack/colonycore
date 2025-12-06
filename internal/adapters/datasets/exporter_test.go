package datasets

import (
	"context"
	"testing"
	"time"

	"colonycore/internal/adapters/testutil"
	"colonycore/internal/core"
	"colonycore/pkg/datasetapi"
	"colonycore/pkg/domain"
)

func TestWorkerProcessesExport(t *testing.T) {
	formatProvider := datasetapi.GetFormatProvider()
	svc := core.NewInMemoryService(core.NewDefaultRulesEngine())
	meta, err := testutil.InstallFrogPlugin(svc)
	if err != nil {
		t.Fatalf("install plugin: %v", err)
	}
	descriptor := meta.Datasets[0]

	store := NewMemoryObjectStore()
	audit := &MemoryAuditLog{}
	worker := NewWorker(svc, store, audit)
	worker.Start()
	t.Cleanup(func() { _ = worker.Stop(context.Background()) })

	ctx := context.Background()
	facility, _, err := svc.CreateFacility(ctx, domain.Facility{Name: "Worker Facility"})
	if err != nil {
		t.Fatalf("create facility: %v", err)
	}
	project, _, err := svc.CreateProject(ctx, domain.Project{Code: "PRJ-WORK", Title: "Worker", FacilityIDs: []string{facility.ID}})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	projectID := project.ID
	if _, _, err := svc.CreateOrganism(ctx, domain.Organism{Name: "Frog", Species: "Tree Frog", Stage: domain.StageAdult, ProjectID: &projectID}); err != nil {
		t.Fatalf("create organism: %v", err)
	}

	record, err := worker.EnqueueExport(ctx, ExportInput{TemplateSlug: descriptor.Slug, Parameters: map[string]any{"include_retired": true}, Formats: []datasetapi.Format{formatProvider.JSON()}, Scope: datasetapi.Scope{ProjectIDs: []string{project.ID}, Requestor: "worker@colonycore"}, RequestedBy: "worker@colonycore"})
	if err != nil {
		t.Fatalf("enqueue export: %v", err)
	}
	if record.Status != ExportStatusQueued {
		t.Fatalf("expected queued status, got %s", record.Status)
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		current, _ := worker.GetExport(record.ID)
		if current.Status == ExportStatusSucceeded {
			if len(current.Artifacts) == 0 {
				t.Fatalf("expected artifacts on completion")
			}
			break
		}
		if current.Status == ExportStatusFailed {
			t.Fatalf("export failed: %s", current.Error)
		}
		if time.Now().After(deadline) {
			t.Fatalf("timeout waiting for worker completion")
		}
		time.Sleep(20 * time.Millisecond)
	}
	if len(store.Objects()) == 0 {
		t.Fatalf("expected object store to contain artifacts")
	}
	if len(audit.Entries()) == 0 {
		t.Fatalf("expected audit entries")
	}
}

func TestWorkerRejectsUnsupportedFormat(t *testing.T) {
	svc := core.NewInMemoryService(core.NewDefaultRulesEngine())
	meta, err := testutil.InstallFrogPlugin(svc)
	if err != nil {
		t.Fatalf("install plugin: %v", err)
	}
	descriptor := meta.Datasets[0]
	worker := NewWorker(svc, NewMemoryObjectStore(), &MemoryAuditLog{})
	ctx := context.Background()
	_, err = worker.EnqueueExport(ctx, ExportInput{TemplateSlug: descriptor.Slug, Formats: []datasetapi.Format{datasetapi.Format("xml")}})
	if err == nil {
		t.Fatalf("expected error for unsupported format")
	}
}
