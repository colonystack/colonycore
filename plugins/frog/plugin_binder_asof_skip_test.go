package frog

import (
	"context"
	"testing"
	"time"

	"colonycore/pkg/datasetapi"
)

// TestFrogPopulationBinderAsOfSkip covers branch skipping rows updated after as_of timestamp.
func TestFrogPopulationBinderAsOfSkip(t *testing.T) {
	plugin := New()
	reg := newStubRegistry()
	if err := plugin.Register(reg); err != nil {
		t.Fatalf("register: %v", err)
	}
	template := reg.datasets[0]

	baseTime := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	asOf := baseTime.Add(-time.Minute) // earlier than organism update time so it will skip
	adult := datasetapi.Organism{Base: datasetapi.Base{ID: "alpha", UpdatedAt: baseTime}, Species: "Frog", Stage: datasetapi.StageAdult}
	view := newStubView()
	view.organisms = []datasetapi.Organism{adult}
	store := stubStore{view: view}
	env := datasetapi.Environment{Store: store, Now: func() time.Time { return baseTime }}
	runner, err := frogPopulationBinder(env)
	if err != nil {
		t.Fatalf("binder: %v", err)
	}
	request := datasetapi.RunRequest{Template: datasetapi.TemplateDescriptor{Columns: template.Columns}, Parameters: map[string]any{"as_of": asOf}}
	result, err := runner(context.Background(), request)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(result.Rows) != 0 {
		t.Fatalf("expected 0 rows due to as_of filter, got %d", len(result.Rows))
	}
}
