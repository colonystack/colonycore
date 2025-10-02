package frog

import (
	"context"
	"testing"
	"time"

	"colonycore/pkg/datasetapi"
)

// TestFrogPopulationBinderMetadata covers metadata enrichment branches (stage_filter, project_scope, protocol_scope, as_of).
func TestFrogPopulationBinderMetadata(t *testing.T) {
	plugin := New()
	reg := newStubRegistry()
	if err := plugin.Register(reg); err != nil {
		t.Fatalf("register: %v", err)
	}
	template := reg.datasets[0]

	now := time.Date(2024, 5, 6, 7, 8, 9, 0, time.UTC)
	proj := "proj"
	protA := "protA"
	protB := "protB"
	stageAdult := datasetapi.StageAdult
	stageRetired := datasetapi.StageRetired
	asOf := now.Add(30 * time.Minute)

	// organisms: include adult with protA, retired with protB (filtered by stage parameter), other species ignored
	adult := datasetapi.Organism{Base: datasetapi.Base{ID: "adult", UpdatedAt: now}, Species: "Frog", Stage: stageAdult, ProjectID: &proj, ProtocolID: &protA}
	retired := datasetapi.Organism{Base: datasetapi.Base{ID: "retired", UpdatedAt: now}, Species: "Frog", Stage: stageRetired, ProjectID: &proj, ProtocolID: &protB}
	other := datasetapi.Organism{Base: datasetapi.Base{ID: "other", UpdatedAt: now}, Species: "Gecko", Stage: stageAdult}
	view := newStubView()
	view.organisms = []datasetapi.Organism{adult, retired, other}
	store := stubStore{view: view}
	env := datasetapi.Environment{Store: store, Now: func() time.Time { return now }}
	runner, err := frogPopulationBinder(env)
	if err != nil {
		t.Fatalf("binder: %v", err)
	}

	req := datasetapi.RunRequest{
		Template:   datasetapi.TemplateDescriptor{Columns: template.Columns},
		Parameters: map[string]any{"stage": string(stageAdult), "as_of": asOf},
		Scope:      datasetapi.Scope{ProjectIDs: []string{proj}, ProtocolIDs: []string{protA}},
	}
	result, err := runner(context.Background(), req)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(result.Rows) != 1 {
		t.Fatalf("expected 1 filtered row, got %d", len(result.Rows))
	}
	meta := result.Metadata
	if meta["stage_filter"] != string(stageAdult) {
		t.Fatalf("expected stage_filter metadata, got %+v", meta)
	}
	if _, ok := meta["as_of"]; !ok {
		t.Fatalf("expected as_of metadata")
	}
	if projScope, ok := meta["project_scope"].([]string); !ok || len(projScope) != 1 || projScope[0] != proj {
		t.Fatalf("expected project_scope metadata, got %+v", meta["project_scope"])
	}
	if protScope, ok := meta["protocol_scope"].([]string); !ok || len(protScope) != 1 || protScope[0] != protA {
		t.Fatalf("expected protocol_scope metadata, got %+v", meta["protocol_scope"])
	}
}
