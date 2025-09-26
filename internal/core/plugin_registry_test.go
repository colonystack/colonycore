package core

import (
	"context"
	"testing"
	"time"

	"colonycore/pkg/datasetapi"
)

func TestPluginRegistryGuardsAndCopies(t *testing.T) {
	registry := NewPluginRegistry()

	registry.RegisterRule(nil)
	if len(registry.Rules()) != 0 {
		t.Fatalf("expected nil rule to be ignored")
	}

	registry.RegisterSchema("", map[string]any{"ignored": true})
	registry.RegisterSchema("organism", nil)

	registry.RegisterRule(staticRule{"rule", SeverityLog})
	rules := registry.Rules()
	if len(rules) != 1 {
		t.Fatalf("expected single registered rule, got %d", len(rules))
	}
	rules[0] = nil
	if registry.Rules()[0] == nil {
		t.Fatalf("expected registry to return copy of rules slice")
	}

	schema := map[string]any{"type": "object"}
	registry.RegisterSchema("organism", schema)
	schema["type"] = "mutated"

	stored := registry.Schemas()
	if stored["organism"]["type"].(string) != "object" {
		t.Fatalf("expected schema copy to remain object")
	}

	stored["organism"]["type"] = "changed"
	if registry.Schemas()["organism"]["type"].(string) != "object" {
		t.Fatalf("expected registry to return defensive copies")
	}

	template := datasetapi.Template{
		Key:         "demo",
		Version:     "1.0.0",
		Title:       "Demo",
		Description: "Demo dataset",
		Dialect:     datasetapi.DialectSQL,
		Query:       "SELECT 1",
		Parameters: []datasetapi.Parameter{{
			Name: "stage",
			Type: "string",
			Enum: []string{"adult"},
		}},
		Columns:       []datasetapi.Column{{Name: "value", Type: "number", Unit: "count"}},
		Metadata:      datasetapi.Metadata{Tags: []string{"demo"}, Annotations: map[string]string{"k": "v"}},
		OutputFormats: []datasetapi.Format{datasetapi.FormatJSON},
		Binder: func(datasetapi.Environment) (datasetapi.Runner, error) {
			return func(context.Context, datasetapi.RunRequest) (datasetapi.RunResult, error) {
				return datasetapi.RunResult{Rows: []map[string]any{{"value": 1}}, GeneratedAt: time.Now().UTC(), Format: datasetapi.FormatJSON}, nil
			}, nil
		},
	}
	if err := registry.RegisterDatasetTemplate(template); err != nil {
		t.Fatalf("register dataset: %v", err)
	}
	registered := registry.DatasetTemplates()
	if len(registered) != 1 {
		t.Fatalf("expected dataset to be registered")
	}
	registered[0].Parameters[0].Enum[0] = "mutated"
	registered[0].Metadata.Tags[0] = "changed"
	registered[0].Metadata.Annotations["k"] = "changed"

	copy := registry.DatasetTemplates()[0]
	if copy.Parameters[0].Enum[0] != "adult" {
		t.Fatalf("expected enum to remain adult")
	}
	if copy.Metadata.Tags[0] != "demo" {
		t.Fatalf("expected metadata tags copy")
	}
	if copy.Metadata.Annotations["k"] != "v" {
		t.Fatalf("expected annotation copy")
	}

	if err := registry.RegisterDatasetTemplate(template); err == nil {
		t.Fatalf("expected duplicate dataset template registration to fail")
	}
}

type emptyView struct{}

func (emptyView) ListOrganisms() []Organism                  { return nil }
func (emptyView) ListHousingUnits() []HousingUnit            { return nil }
func (emptyView) FindOrganism(string) (Organism, bool)       { return Organism{}, false }
func (emptyView) FindHousingUnit(string) (HousingUnit, bool) { return HousingUnit{}, false }
func (emptyView) ListProtocols() []Protocol                  { return nil }

func TestRulesEngineEvaluateDirect(t *testing.T) {
	engine := NewRulesEngine()
	engine.Register(staticRule{"first", SeverityWarn})
	engine.Register(staticRule{"second", SeverityLog})

	view := emptyView{}
	result, err := engine.Evaluate(context.Background(), view, nil)
	if err != nil {
		t.Fatalf("unexpected evaluate error: %v", err)
	}
	if len(result.Violations) != 2 {
		t.Fatalf("expected 2 violations, got %d", len(result.Violations))
	}
	if result.Violations[0].Rule != "first" {
		t.Fatalf("unexpected rule order")
	}
}

func TestRuleNameCoverage(t *testing.T) {
	if name := NewHousingCapacityRule().Name(); name == "" {
		t.Fatalf("expected housing rule name")
	}
	if name := NewProtocolSubjectCapRule().Name(); name == "" {
		t.Fatalf("expected protocol rule name")
	}
}
