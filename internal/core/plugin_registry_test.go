package core

import (
	"context"
	"testing"
	"time"

	"colonycore/pkg/datasetapi"
	"colonycore/pkg/domain"
	entitymodel "colonycore/pkg/domain/entitymodel"
	"colonycore/pkg/pluginapi"
)

type pluginRuleStub struct {
	name string
}

func (r pluginRuleStub) Name() string { return r.name }

func (r pluginRuleStub) Evaluate(_ context.Context, _ pluginapi.RuleView, _ []pluginapi.Change) (pluginapi.Result, error) {
	return pluginapi.Result{}, nil
}

func TestPluginRegistryGuardsAndCopies(t *testing.T) {
	dialectProvider := datasetapi.GetDialectProvider()
	formatProvider := datasetapi.GetFormatProvider()

	registry := NewPluginRegistry()

	registry.RegisterRule(nil)
	if len(registry.Rules()) != 0 {
		t.Fatalf("expected nil rule to be ignored")
	}

	registry.RegisterSchema("", map[string]any{"ignored": true})
	registry.RegisterSchema("organism", nil)

	registry.RegisterRule(pluginRuleStub{name: "rule"})
	rules := registry.Rules()
	if len(rules) != 1 {
		t.Fatalf("expected single registered rule, got %d", len(rules))
	}
	if rules[0].Name() != "rule" {
		t.Fatalf("expected adapter to preserve rule name")
	}
	if _, err := rules[0].Evaluate(context.Background(), emptyView{}, nil); err != nil {
		t.Fatalf("expected adapted rule to evaluate: %v", err)
	}
	rules[0] = nil
	if registry.Rules()[0] == nil {
		t.Fatalf("expected registry to return copy of rules slice")
	}

	schema := map[string]any{"type": "object"}
	registry.RegisterSchema("organism", schema)
	schema["type"] = testLiteralMutated

	stored := registry.Schemas()
	if stored["organism"]["type"].(string) != "object" {
		t.Fatalf("expected schema copy to remain object")
	}

	stored["organism"]["type"] = testLiteralChanged
	if registry.Schemas()["organism"]["type"].(string) != "object" {
		t.Fatalf("expected registry to return defensive copies")
	}

	template := datasetapi.Template{
		Key:         "demo",
		Version:     "1.0.0",
		Title:       "Demo",
		Description: "Demo dataset",
		Dialect:     dialectProvider.SQL(),
		Query:       "SELECT 1",
		Parameters: []datasetapi.Parameter{{
			Name: "stage",
			Type: "string",
			Enum: []string{"adult"},
		}},
		Columns:       []datasetapi.Column{{Name: "value", Type: "number", Unit: "count"}},
		Metadata:      datasetapi.Metadata{Tags: []string{"demo"}, Annotations: map[string]string{"k": "v"}},
		OutputFormats: []datasetapi.Format{formatProvider.JSON()},
		Binder: func(datasetapi.Environment) (datasetapi.Runner, error) {
			return func(context.Context, datasetapi.RunRequest) (datasetapi.RunResult, error) {
				return datasetapi.RunResult{Rows: []datasetapi.Row{{"value": 1}}, GeneratedAt: time.Now().UTC(), Format: formatProvider.JSON()}, nil
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
	registered[0].Parameters[0].Enum[0] = testLiteralMutated
	registered[0].Metadata.Tags[0] = testLiteralChanged
	registered[0].Metadata.Annotations["k"] = testLiteralChanged

	tmplCopy := registry.DatasetTemplates()[0]
	if tmplCopy.Parameters[0].Enum[0] != "adult" {
		t.Fatalf("expected enum to remain adult")
	}
	if tmplCopy.Metadata.Tags[0] != "demo" {
		t.Fatalf("expected metadata tags copy")
	}
	if tmplCopy.Metadata.Annotations["k"] != "v" {
		t.Fatalf("expected annotation copy")
	}

	if err := registry.RegisterDatasetTemplate(template); err == nil {
		t.Fatalf("expected duplicate dataset template registration to fail")
	}
}

type emptyView struct{}

func (emptyView) ListOrganisms() []domain.Organism       { return nil }
func (emptyView) ListHousingUnits() []domain.HousingUnit { return nil }
func (emptyView) ListFacilities() []domain.Facility      { return nil }
func (emptyView) ListTreatments() []domain.Treatment     { return nil }
func (emptyView) ListObservations() []domain.Observation { return nil }
func (emptyView) ListSamples() []domain.Sample           { return nil }
func (emptyView) FindOrganism(string) (domain.Organism, bool) {
	return domain.Organism{Organism: entitymodel.Organism{}}, false
}
func (emptyView) FindHousingUnit(string) (domain.HousingUnit, bool) {
	return domain.HousingUnit{HousingUnit: entitymodel.HousingUnit{}}, false
}
func (emptyView) FindFacility(string) (domain.Facility, bool) {
	return domain.Facility{Facility: entitymodel.Facility{}}, false
}
func (emptyView) FindTreatment(string) (domain.Treatment, bool) {
	return domain.Treatment{Treatment: entitymodel.Treatment{}}, false
}
func (emptyView) FindObservation(string) (domain.Observation, bool) {
	return domain.Observation{Observation: entitymodel.Observation{}}, false
}
func (emptyView) FindSample(string) (domain.Sample, bool) {
	return domain.Sample{Sample: entitymodel.Sample{}}, false
}
func (emptyView) ListProtocols() []domain.Protocol     { return nil }
func (emptyView) ListPermits() []domain.Permit         { return nil }
func (emptyView) ListProjects() []domain.Project       { return nil }
func (emptyView) ListSupplyItems() []domain.SupplyItem { return nil }
func (emptyView) FindPermit(string) (domain.Permit, bool) {
	return domain.Permit{Permit: entitymodel.Permit{}}, false
}
func (emptyView) FindSupplyItem(string) (domain.SupplyItem, bool) {
	return domain.SupplyItem{SupplyItem: entitymodel.SupplyItem{}}, false
}

func (emptyView) FindProcedure(string) (domain.Procedure, bool) {
	return domain.Procedure{Procedure: entitymodel.Procedure{}}, false
}

func TestRulesEngineEvaluateDirect(t *testing.T) {
	engine := NewRulesEngine()
	engine.Register(staticRule{"first", domain.SeverityWarn})
	engine.Register(staticRule{"second", domain.SeverityLog})

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
