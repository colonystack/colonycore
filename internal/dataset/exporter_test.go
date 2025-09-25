package dataset_test

import (
	"context"
	"testing"
	"time"

	"colonycore/internal/core"
	"colonycore/internal/dataset"
	domain "colonycore/pkg/domain"
	"colonycore/plugins/frog"
)

func TestWorkerProcessesExport(t *testing.T) {
	svc := core.NewInMemoryService(core.NewDefaultRulesEngine())
	meta, err := svc.InstallPlugin(frog.New())
	if err != nil {
		t.Fatalf("install plugin: %v", err)
	}
	descriptor := meta.Datasets[0]

	store := dataset.NewMemoryObjectStore()
	audit := &dataset.MemoryAuditLog{}
	worker := dataset.NewWorker(svc, store, audit)
	worker.Start()
	t.Cleanup(func() {
		_ = worker.Stop(context.Background())
	})

	ctx := context.Background()
	project, _, err := svc.CreateProject(ctx, domain.Project{Code: "PRJ-WORK", Title: "Worker"})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	projectID := project.ID
	if _, _, err := svc.CreateOrganism(ctx, domain.Organism{Name: "Frog", Species: "Tree Frog", Stage: domain.StageAdult, ProjectID: &projectID}); err != nil {
		t.Fatalf("create organism: %v", err)
	}

	record, err := worker.EnqueueExport(ctx, dataset.ExportInput{
		TemplateSlug: descriptor.Slug,
		Parameters:   map[string]any{"include_retired": true},
		Formats:      []core.DatasetFormat{core.FormatJSON},
		Scope:        core.DatasetScope{ProjectIDs: []string{project.ID}, Requestor: "worker@colonycore"},
		RequestedBy:  "worker@colonycore",
	})
	if err != nil {
		t.Fatalf("enqueue export: %v", err)
	}
	if record.Status != dataset.ExportStatusQueued {
		t.Fatalf("expected queued status, got %s", record.Status)
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		current, _ := worker.GetExport(record.ID)
		if current.Status == dataset.ExportStatusSucceeded {
			if len(current.Artifacts) == 0 {
				t.Fatalf("expected artifacts on completion")
			}
			break
		}
		if current.Status == dataset.ExportStatusFailed {
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
	meta, err := svc.InstallPlugin(frog.New())
	if err != nil {
		t.Fatalf("install plugin: %v", err)
	}
	descriptor := meta.Datasets[0]

	worker := dataset.NewWorker(svc, dataset.NewMemoryObjectStore(), &dataset.MemoryAuditLog{})
	ctx := context.Background()

	_, err = worker.EnqueueExport(ctx, dataset.ExportInput{
		TemplateSlug: descriptor.Slug,
		Formats:      []core.DatasetFormat{core.DatasetFormat("xml")},
	})
	if err == nil {
		t.Fatalf("expected error for unsupported format")
	}
}
