package extension

import (
	"encoding/json"
	"testing"
)

func TestNewObjectPayloadClonesInput(t *testing.T) {
	input := map[string]any{
		"k": []any{
			map[string]any{"nested": "value"},
		},
	}
	payload, err := NewObjectPayload(HookOrganismAttributes, input)
	if err != nil {
		t.Fatalf("NewObjectPayload: %v", err)
	}
	if payload.Hook() != HookOrganismAttributes {
		t.Fatalf("expected hook %q, got %q", HookOrganismAttributes, payload.Hook())
	}
	cloned := payload.Map()
	if cloned["k"].([]any)[0].(map[string]any)["nested"] != "value" {
		t.Fatalf("expected cloned payload to match input")
	}
	// Mutate the original to assert defensive cloning.
	input["k"].([]any)[0].(map[string]any)["nested"] = "mutated"
	if cloned["k"].([]any)[0].(map[string]any)["nested"] != "value" {
		t.Fatalf("payload retained reference to input after mutation")
	}
}

func TestObjectFromContainerMissingPayload(t *testing.T) {
	payload, err := ObjectFromContainer(nil, HookOrganismAttributes, PluginCore)
	if err != nil {
		t.Fatalf("ObjectFromContainer nil: %v", err)
	}
	if !payload.Defined() {
		t.Fatalf("expected payload to be initialised for hook")
	}
	if !payload.IsEmpty() {
		t.Fatalf("expected empty payload for missing hook data")
	}
}

func TestObjectPayloadExpectHookAndMarshal(t *testing.T) {
	container := NewContainer()
	if err := container.Set(HookOrganismAttributes, PluginCore, map[string]any{"tag": "alpha"}); err != nil {
		t.Fatalf("container.Set: %v", err)
	}
	payload, err := ObjectFromContainer(&container, HookOrganismAttributes, PluginCore)
	if err != nil {
		t.Fatalf("ObjectFromContainer: %v", err)
	}
	if err := payload.ExpectHook(HookOrganismAttributes); err != nil {
		t.Fatalf("ExpectHook: %v", err)
	}
	if err := payload.ExpectHook(HookFacilityEnvironmentBaselines); err == nil {
		t.Fatalf("expected hook mismatch to error")
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	if string(encoded) != `{"tag":"alpha"}` {
		t.Fatalf("unexpected JSON encoding: %s", encoded)
	}
}

func TestNewObjectPayloadValidations(t *testing.T) {
	if _, err := NewObjectPayload(HookOrganismAttributes, []string{"bad"}); err == nil {
		t.Fatalf("expected validation error for non-object payload")
	}
	payload, err := NewObjectPayload(HookOrganismAttributes, nil)
	if err != nil {
		t.Fatalf("NewObjectPayload nil: %v", err)
	}
	if !payload.Defined() {
		t.Fatalf("expected payload to be initialised")
	}
	if !payload.IsEmpty() {
		t.Fatalf("expected nil payload to report empty")
	}
	if payload.Map() != nil {
		t.Fatalf("expected nil map from empty payload")
	}
	zero := ObjectPayload{}
	if zero.Defined() {
		t.Fatalf("zero value payload should not be initialised")
	}
	if !zero.IsEmpty() {
		t.Fatalf("zero value payload should be empty")
	}
	if m := zero.Map(); m != nil {
		t.Fatalf("expected nil map for zero payload, got %v", m)
	}
	if data, err := json.Marshal(zero); err != nil || string(data) != "null" {
		t.Fatalf("expected zero payload to encode as null, got %q (err=%v)", data, err)
	}
	if err := zero.ExpectHook(HookOrganismAttributes); err == nil {
		t.Fatalf("expected ExpectHook to fail for uninitialised payload")
	}
	if err := payload.ExpectHook(""); err == nil {
		t.Fatalf("expected empty hook validation to fail")
	}
}

func TestObjectFromContainerExistingPayload(t *testing.T) {
	container := NewContainer()
	if err := container.Set(HookOrganismAttributes, PluginID("external"), map[string]any{"flag": true}); err != nil {
		t.Fatalf("container.Set external: %v", err)
	}
	if err := container.Set(HookOrganismAttributes, PluginCore, map[string]any{"energy": 5}); err != nil {
		t.Fatalf("container.Set core: %v", err)
	}
	payload, err := ObjectFromContainer(&container, HookOrganismAttributes, PluginCore)
	if err != nil {
		t.Fatalf("ObjectFromContainer core: %v", err)
	}
	if payload.IsEmpty() {
		t.Fatalf("expected core payload to be non-empty")
	}
	if got := payload.Map()["energy"]; got != 5 {
		t.Fatalf("unexpected payload contents: %v", got)
	}
}
