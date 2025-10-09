package datasets

import (
	"colonycore/internal/core"
	"colonycore/pkg/datasetapi"
	"context"
	"testing"
	"time"
)

// TestWorkerStopTwice covers branch where Stop is invoked multiple times (second call should be a no-op).
func TestWorkerStopTwice(t *testing.T) {
	formatProvider := datasetapi.GetFormatProvider()
	tpl := buildTemplate()
	w := NewWorker(fakeCatalog{tpl: core.DatasetTemplateRuntimeForTests(tpl)}, nil, nil)
	w.Start()
	// enqueue one export to ensure worker loop started
	_, _ = w.EnqueueExport(context.Background(), ExportInput{TemplateSlug: tpl.Descriptor().Slug, Formats: []datasetapi.Format{formatProvider.JSON()}})
	// stop worker first time
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := w.Stop(ctx); err != nil {
		t.Fatalf("first stop error: %v", err)
	}
	// second stop should return quickly without error
	if err := w.Stop(ctx); err != nil {
		t.Fatalf("second stop error: %v", err)
	}
}
