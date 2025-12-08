package validation

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateContractFlowsSuccess(t *testing.T) {
	dir, schema := writeFlowFixtures(t, map[string]any{
		"entity": "Organism",
		"action": "create",
		"payload": map[string]any{
			"id":         "123",
			"name":       "Specimen",
			"attributes": map[string]any{"color": "green"},
		},
	})
	errs := ValidateContractFlows(dir, schema)
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %+v", errs)
	}
}

func TestValidateContractFlowsMissingRequired(t *testing.T) {
	dir, schema := writeFlowFixtures(t, map[string]any{
		"entity":  "Organism",
		"action":  "create",
		"payload": map[string]any{"id": "123"},
	})
	errs := ValidateContractFlows(dir, schema)
	if len(errs) == 0 {
		t.Fatalf("expected missing field error")
	}
}

func TestValidateContractFlowsUnknownField(t *testing.T) {
	dir, schema := writeFlowFixtures(t, map[string]any{
		"entity": "Organism",
		"action": "update",
		"payload": map[string]any{
			"id":           "123",
			"name":         "Specimen",
			"unexpected":   true,
			"attributes":   map[string]any{"color": "green"},
			"unknown_hook": map[string]any{},
		},
	})
	errs := ValidateContractFlows(dir, schema)
	if len(errs) == 0 {
		t.Fatalf("expected error for unknown fields")
	}
}

func TestValidateContractFlowsInvalidAction(t *testing.T) {
	dir, schema := writeFlowFixtures(t, map[string]any{
		"entity":  "Organism",
		"action":  "delete",
		"payload": map[string]any{"id": "1", "name": "specimen"},
	})
	errs := ValidateContractFlows(dir, schema)
	if len(errs) == 0 {
		t.Fatalf("expected action error")
	}
}

func TestValidateContractFlowsUnknownEntity(t *testing.T) {
	dir, schema := writeFlowFixtures(t, map[string]any{
		"entity":  "Unknown",
		"action":  "create",
		"payload": map[string]any{"id": "1", "name": "specimen"},
	})
	errs := ValidateContractFlows(dir, schema)
	if len(errs) == 0 {
		t.Fatalf("expected unknown entity error")
	}
}

func TestValidateContractFlowsPayloadRequired(t *testing.T) {
	dir, schema := writeFlowFixtures(t, map[string]any{
		"entity": "Organism",
		"action": "create",
	})
	errs := ValidateContractFlows(dir, schema)
	if len(errs) == 0 {
		t.Fatalf("expected payload error")
	}
	if !strings.Contains(errs[0].Message, "payload is required") {
		t.Fatalf("unexpected error: %v", errs[0].Message)
	}
}

func TestValidateContractFlowsMissingEntity(t *testing.T) {
	dir, schema := writeFlowFixtures(t, map[string]any{
		"action":  "create",
		"payload": map[string]any{"id": "123"},
	})
	errs := ValidateContractFlows(dir, schema)
	if len(errs) == 0 {
		t.Fatalf("expected entity error")
	}
	if !strings.Contains(errs[0].Message, "entity is required") {
		t.Fatalf("unexpected error: %v", errs[0].Message)
	}
}

func TestValidateContractFlowsMissingAction(t *testing.T) {
	dir, schema := writeFlowFixtures(t, map[string]any{
		"entity":  "Organism",
		"payload": map[string]any{"id": "123", "name": "Specimen"},
	})
	errs := ValidateContractFlows(dir, schema)
	if len(errs) == 0 {
		t.Fatalf("expected action error")
	}
	if !strings.Contains(errs[0].Message, "action is required") {
		t.Fatalf("unexpected error: %v", errs[0].Message)
	}
}

func TestValidateContractFlowsSchemaLoadFailure(t *testing.T) {
	dir, _ := writeFlowFixtures(t, map[string]any{
		"entity":  "Organism",
		"action":  "create",
		"payload": map[string]any{"id": "123", "name": "Specimen"},
	})
	missingSchema := filepath.Join(dir, "missing.json")
	errs := ValidateContractFlows(dir, missingSchema)
	if len(errs) == 0 {
		t.Fatalf("expected schema error")
	}
	if !strings.Contains(errs[0].Message, "load contract metadata") {
		t.Fatalf("unexpected error: %v", errs[0].Message)
	}
}

func TestValidateContractFlowsInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	flowsDir := filepath.Join(dir, "contract_flows")
	if err := os.MkdirAll(flowsDir, 0o750); err != nil {
		t.Fatalf("mkdir flows: %v", err)
	}
	if err := os.WriteFile(filepath.Join(flowsDir, "bad.json"), []byte("not json"), 0o600); err != nil {
		t.Fatalf("write bad flow: %v", err)
	}
	schema := writeSchema(t)
	errs := ValidateContractFlows(dir, schema)
	if len(errs) == 0 {
		t.Fatalf("expected parse error")
	}
}

func TestValidateContractFlowsMissingDir(t *testing.T) {
	schema := writeSchema(t)
	errs := ValidateContractFlows(t.TempDir(), schema)
	if len(errs) != 0 {
		t.Fatalf("expected no errors when directory missing")
	}
}

func TestMissingRequiredHelpers(t *testing.T) {
	if missing := missingRequired(nil, map[string]any{"id": "1"}); len(missing) != 0 {
		t.Fatalf("expected no missing fields when requirements empty")
	}
	out := missingRequired([]string{"name", "id"}, map[string]any{"name": "frog"})
	if len(out) != 1 || out[0] != "id" {
		t.Fatalf("expected id to be reported missing, got %v", out)
	}
	unsorted := missingRequired([]string{"b", "a"}, map[string]any{})
	if len(unsorted) != 2 || unsorted[0] != "a" || unsorted[1] != "b" {
		t.Fatalf("expected sorted missing fields, got %v", unsorted)
	}
}

func writeFlowFixtures(t *testing.T, payload map[string]any) (string, string) {
	t.Helper()
	dir := t.TempDir()
	flowsDir := filepath.Join(dir, "contract_flows")
	if err := os.MkdirAll(flowsDir, 0o750); err != nil {
		t.Fatalf("mkdir flows: %v", err)
	}
	flowPath := filepath.Join(flowsDir, "flow.json")
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal flow: %v", err)
	}
	if err := os.WriteFile(flowPath, data, 0o600); err != nil {
		t.Fatalf("write flow: %v", err)
	}
	schema := writeSchema(t)
	return dir, schema
}

func writeSchema(t *testing.T) string {
	t.Helper()
	schema := `{
        "version": "test",
        "entities": {
            "Organism": {
                "required": ["id", "name"],
                "properties": {
                    "id": {},
                    "name": {},
                    "attributes": {"$ref": "#/definitions/extension_attributes"}
                }
            }
        }
    }`
	path := filepath.Join(t.TempDir(), "schema.json")
	if err := os.WriteFile(path, []byte(schema), 0o600); err != nil {
		t.Fatalf("write schema: %v", err)
	}
	return path
}
