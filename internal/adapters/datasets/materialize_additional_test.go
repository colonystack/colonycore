package datasets

import (
	"context"
	"testing"
	"time"

	"colonycore/internal/core"
	"colonycore/pkg/datasetapi"
)

func TestWorkerMaterializePNGParquet(t *testing.T) {
	tmpl := datasetapi.Template{Key: "allfmts", Version: "1.0.0", Title: "All Formats", Description: "cover png/parquet", Dialect: datasetapi.DialectSQL, Query: "SELECT 1", Columns: []datasetapi.Column{{Name: "value", Type: "string"}}, OutputFormats: []datasetapi.Format{datasetapi.FormatJSON, datasetapi.FormatCSV, datasetapi.FormatHTML, datasetapi.FormatPNG, datasetapi.FormatParquet}, Binder: func(datasetapi.Environment) (datasetapi.Runner, error) {
		return func(context.Context, datasetapi.RunRequest) (datasetapi.RunResult, error) {
			return datasetapi.RunResult{Rows: []map[string]any{{"value": "alpha"}}, Format: datasetapi.FormatJSON}, nil
		}, nil
	}}
	svc := core.NewInMemoryService(core.NewDefaultRulesEngine())
	if _, err := svc.InstallPlugin(testDatasetPlugin{dataset: tmpl}); err != nil {
		t.Fatalf("install plugin: %v", err)
	}
	store := NewMemoryObjectStore()
	worker := NewWorker(svc, store, &MemoryAuditLog{})
	worker.Start()
	t.Cleanup(func() { _ = worker.Stop(context.Background()) })
	slug := svc.DatasetTemplates()[0].Slug
	formats := []core.DatasetFormat{core.FormatJSON, core.FormatPNG, core.FormatParquet}
	rec, err := worker.EnqueueExport(context.Background(), ExportInput{TemplateSlug: slug, RequestedBy: "tester", Formats: formats})
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for {
		cur, _ := worker.GetExport(rec.ID)
		if cur.Status == ExportStatusSucceeded {
			if len(cur.Artifacts) != len(formats) {
				t.Fatalf("expected %d artifacts, got %d", len(formats), len(cur.Artifacts))
			}
			seen := map[core.DatasetFormat]bool{}
			for _, a := range cur.Artifacts {
				seen[a.Format] = true
			}
			for _, f := range formats {
				if !seen[f] {
					t.Fatalf("missing artifact for %s", f)
				}
			}
			return
		}
		if cur.Status == ExportStatusFailed {
			t.Fatalf("unexpected failure: %s", cur.Error)
		}
		if time.Now().After(deadline) {
			t.Fatalf("timeout waiting for export completion; status=%s", cur.Status)
		}
		time.Sleep(25 * time.Millisecond)
	}
}
