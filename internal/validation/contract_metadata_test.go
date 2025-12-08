package validation

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadContractMetadata(t *testing.T) {
	schema := `{
        "version": "0.0.1",
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
	tmp := filepath.Join(t.TempDir(), "schema.json")
	if err := os.WriteFile(tmp, []byte(schema), 0o600); err != nil {
		t.Fatalf("write schema: %v", err)
	}
	meta, err := LoadContractMetadata(tmp)
	if err != nil {
		t.Fatalf("load metadata: %v", err)
	}
	if meta.Version != "0.0.1" {
		t.Fatalf("expected version 0.0.1, got %s", meta.Version)
	}
	organism, ok := meta.Entities["Organism"]
	if !ok {
		t.Fatalf("expected organism entity")
	}
	if !organism.HasProperty("name") {
		t.Fatalf("expected name property")
	}
	if !organism.IsExtensionHook("attributes") {
		t.Fatalf("expected attributes hook")
	}
	if organism.IsExtensionHook("name") {
		t.Fatalf("name should not be treated as extension hook")
	}
}

func TestLoadContractMetadataErrors(t *testing.T) {
	if _, err := LoadContractMetadata("/does/not/exist.json"); err == nil {
		t.Fatalf("expected error for missing schema file")
	}
	path := filepath.Join(t.TempDir(), "schema.json")
	if err := os.WriteFile(path, []byte("invalid"), 0o600); err != nil {
		t.Fatalf("write invalid schema: %v", err)
	}
	if _, err := LoadContractMetadata(path); err == nil {
		t.Fatalf("expected parse error for invalid schema")
	}
}

func TestContractEntityHelpersHandleEmptyData(t *testing.T) {
	var entity ContractEntity
	if entity.HasProperty("id") {
		t.Fatalf("expected HasProperty to return false when properties unset")
	}
	if entity.IsExtensionHook("attributes") {
		t.Fatalf("expected IsExtensionHook to return false when hooks unset")
	}
}
