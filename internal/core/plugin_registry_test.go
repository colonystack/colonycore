package core

import (
	"context"
	"testing"
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
}

func TestRulesEngineEvaluateDirect(t *testing.T) {
	engine := NewRulesEngine()
	engine.Register(staticRule{"first", SeverityWarn})
	engine.Register(staticRule{"second", SeverityLog})

	view := TransactionView{state: &memoryState{organisms: map[string]Organism{}}}
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
