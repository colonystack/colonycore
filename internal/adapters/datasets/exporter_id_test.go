package datasets

import (
	"context"
	"testing"

	"colonycore/internal/core"
	"colonycore/pkg/datasetapi"
)

// TestEnqueueExportGeneratesUniqueIDs validates that Worker assigns unique IDs (indirectly testing newID()).
func TestEnqueueExportGeneratesUniqueIDs(t *testing.T) {
	dialectProvider := datasetapi.GetDialectProvider()
	formatProvider := datasetapi.GetFormatProvider()
	tmpl := datasetapi.Template{
		Key:           "uniq",
		Version:       "1.0.0",
		Title:         "Uniq",
		Description:   "uniq ids",
		Dialect:       dialectProvider.SQL(),
		Query:         "SELECT 1",
		Columns:       []datasetapi.Column{{Name: "v", Type: "string"}},
		OutputFormats: []datasetapi.Format{formatProvider.JSON()},
		Binder: func(datasetapi.Environment) (datasetapi.Runner, error) {
			return func(context.Context, datasetapi.RunRequest) (datasetapi.RunResult, error) {
				return datasetapi.RunResult{Rows: []datasetapi.Row{{"v": "ok"}}}, nil
			}, nil
		},
	}
	svc := core.NewInMemoryService(core.NewDefaultRulesEngine())
	if _, err := svc.InstallPlugin(testDatasetPlugin{dataset: tmpl}); err != nil {
		t.Fatalf("install plugin: %v", err)
	}
	worker := NewWorker(svc, nil, nil)
	ids := make(map[string]struct{})
	for i := 0; i < 30; i++ {
		rec, err := worker.EnqueueExport(context.Background(), ExportInput{TemplateSlug: svc.DatasetTemplates()[0].Slug, RequestedBy: "tester", Formats: []datasetapi.Format{formatProvider.JSON()}})
		if err != nil {
			t.Fatalf("enqueue: %v", err)
		}
		if rec.ID == "" {
			t.Fatalf("expected id")
		}
		if _, dup := ids[rec.ID]; dup {
			t.Fatalf("duplicate id generated: %s", rec.ID)
		}
		ids[rec.ID] = struct{}{}
	}
}
