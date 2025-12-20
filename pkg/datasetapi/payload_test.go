package datasetapi

import "testing"

func TestNewExtensionPayloadClonesInput(t *testing.T) {
	source := map[string]any{
		"nested": map[string]any{"value": "alpha"},
	}
	payload := NewExtensionPayload(source)
	if !payload.Defined() {
		t.Fatalf("expected payload to be defined")
	}
	cloned := payload.Map()
	if cloned["nested"].(map[string]any)["value"] != "alpha" {
		t.Fatalf("expected cloned payload to match input")
	}
	source["nested"].(map[string]any)["value"] = mutatedLiteral
	if payload.Map()["nested"].(map[string]any)["value"] != "alpha" {
		t.Fatalf("expected payload clone to remain immutable after source mutation")
	}
}

func TestExtensionPayloadEmptyStates(t *testing.T) {
	var zero ExtensionPayload
	if zero.Defined() {
		t.Fatalf("zero payload should not be defined")
	}
	if !zero.IsEmpty() {
		t.Fatalf("zero payload should be empty")
	}
	if zero.Map() != nil {
		t.Fatalf("zero payload should return nil map")
	}

	payload := NewExtensionPayload(nil)
	if !payload.Defined() {
		t.Fatalf("nil payload should still be defined")
	}
	if !payload.IsEmpty() {
		t.Fatalf("nil payload should be empty")
	}
	if payload.Map() != nil {
		t.Fatalf("expected nil map for empty payload")
	}
}
