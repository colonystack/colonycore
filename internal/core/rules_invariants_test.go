package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestEntityModelInvariantsAlignWithDefaultRules(t *testing.T) {
	schemaPath := filepath.Join("..", "..", "docs", "schema", "entity-model.json")
	data, err := os.ReadFile(schemaPath) //nolint:gosec // repository-local schema path
	if err != nil {
		t.Fatalf("read entity-model schema: %v", err)
	}

	var doc struct {
		Entities map[string]struct {
			Invariants []string `json:"invariants"`
		} `json:"entities"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("parse entity-model schema: %v", err)
	}
	if len(doc.Entities) == 0 {
		t.Fatal("entity-model schema contained no entities")
	}

	schemaInvariants := make(map[string]struct{})
	for name, entity := range doc.Entities {
		for _, inv := range entity.Invariants {
			if inv == "" {
				t.Fatalf("entity %s declares empty invariant name", name)
			}
			schemaInvariants[inv] = struct{}{}
		}
	}
	if len(schemaInvariants) == 0 {
		t.Fatal("entity-model schema declares no invariants")
	}

	ruleNames := make(map[string]struct{})
	for _, rule := range defaultRules() {
		name := rule.Name()
		if name == "" {
			t.Fatalf("encountered default rule with empty name: %#v", rule)
		}
		if _, exists := ruleNames[name]; exists {
			t.Fatalf("duplicate default rule name detected: %s", name)
		}
		ruleNames[name] = struct{}{}
	}

	if !equalSets(schemaInvariants, ruleNames) {
		t.Fatalf("default rules %v must match schema invariants %v", sortedKeys(ruleNames), sortedKeys(schemaInvariants))
	}
}

func equalSets(a, b map[string]struct{}) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if _, ok := b[k]; !ok {
			return false
		}
	}
	return true
}

func sortedKeys(set map[string]struct{}) []string {
	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
