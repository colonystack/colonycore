package pluginapi

import "testing"

func TestNewObjectPayloadClonesInput(t *testing.T) {
	source := map[string]any{
		"nested": map[string]any{
			"value": "alpha",
		},
	}
	payload := NewObjectPayload(source)
	if !payload.Defined() {
		t.Fatalf("expected payload to be defined")
	}
	cloned := payload.Map()
	if cloned["nested"].(map[string]any)["value"] != "alpha" {
		t.Fatalf("expected cloned payload to retain value, got %+v", cloned)
	}
	source["nested"].(map[string]any)["value"] = "mutated"
	if cloned["nested"].(map[string]any)["value"] != "alpha" {
		t.Fatalf("payload map should be immutable from source mutations")
	}
}

func TestObjectPayloadEmptyStates(t *testing.T) {
	var zero ObjectPayload
	if zero.Defined() {
		t.Fatalf("zero payload should not be defined")
	}
	if !zero.IsEmpty() {
		t.Fatalf("zero payload should be empty")
	}
	if zero.Map() != nil {
		t.Fatalf("expected nil map for zero payload")
	}

	payload := NewObjectPayload(nil)
	if !payload.Defined() {
		t.Fatalf("nil input should still mark payload as defined")
	}
	if !payload.IsEmpty() {
		t.Fatalf("nil input should result in empty payload")
	}
	if payload.Map() != nil {
		t.Fatalf("expected nil map for empty payload")
	}
}
