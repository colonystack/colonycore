package datasets

import (
	"colonycore/internal/core"
	"colonycore/plugins/frog"
	"context"
	"testing"
	"time"
)

// Smoke test verifying worker can enqueue & complete default export (JSON).
func TestWorkerSmoke(t *testing.T) {
	svc := core.NewInMemoryService(core.NewDefaultRulesEngine())
	meta, err := svc.InstallPlugin(frog.New())
	if err != nil {
		t.Fatalf("install plugin: %v", err)
	}
	store := NewMemoryObjectStore()
	worker := NewWorker(svc, store, &MemoryAuditLog{})
	worker.Start()
	t.Cleanup(func() { _ = worker.Stop(context.Background()) })
	rec, err := worker.EnqueueExport(context.Background(), ExportInput{TemplateSlug: meta.Datasets[0].Slug, RequestedBy: "tester"})
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		cur, _ := worker.GetExport(rec.ID)
		if cur.Status == ExportStatusSucceeded {
			break
		}
		if cur.Status == ExportStatusFailed {
			t.Fatalf("failed: %s", cur.Error)
		}
		if time.Now().After(deadline) {
			t.Fatalf("timeout waiting for export")
		}
		time.Sleep(20 * time.Millisecond)
	}
	if len(store.Objects()) == 0 {
		t.Fatalf("expected artifact")
	}
}
