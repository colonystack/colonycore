package extension

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestFromRawHandlesNilPluginMap(t *testing.T) {
	container, err := FromRaw(map[string]map[string]any{
		string(HookOrganismAttributes): nil,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hooks := container.Hooks(); hooks != nil {
		t.Fatalf("expected no hooks to be registered, got %v", hooks)
	}
}

func TestContainerHooksAndPluginsEmpty(t *testing.T) {
	container := NewContainer()
	if hooks := container.Hooks(); hooks != nil {
		t.Fatalf("expected nil hooks for empty container")
	}
	if plugins := container.Plugins(HookOrganismAttributes); plugins != nil {
		t.Fatalf("expected nil plugins for unknown hook")
	}
}

func TestContainerUnmarshalJSONNull(t *testing.T) {
	var container Container
	if err := json.Unmarshal([]byte("null"), &container); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if hooks := container.Hooks(); hooks != nil {
		t.Fatalf("expected nil hooks after null unmarshal")
	}
}

func TestCloneValueBranches(t *testing.T) {
	// Map with non-string keys should be returned as-is.
	originalMap := map[int]string{1: "one"}
	clonedMap := cloneValue(originalMap).(map[int]string)
	clonedMap[2] = "two"
	if len(originalMap) != 2 {
		t.Fatalf("expected original map to reflect changes when keys are non-strings")
	}

	// Array should be deep-copied.
	array := [2]int{1, 2}
	clonedArray := cloneValue(array).([2]int)
	clonedArray[0] = 99
	if array[0] != 1 {
		t.Fatalf("expected array clone to be independent")
	}

	// Slice clone should be independent.
	slice := []string{"a", "b"}
	clonedSlice := cloneValue(slice).([]string)
	clonedSlice[0] = "z"
	if slice[0] != "a" {
		t.Fatalf("expected slice clone to be independent")
	}
}

func TestCloneIntoTypeBranches(t *testing.T) {
	// Invalid value should return zero of target type.
	zero := cloneIntoType(reflect.Value{}, reflect.TypeOf(int(0)))
	if zero.Int() != 0 {
		t.Fatalf("expected zero value for invalid input")
	}

	// Convertible value should convert to target type.
	converted := cloneIntoType(reflect.ValueOf(int32(7)), reflect.TypeOf(int64(0)))
	if converted.Int() != 7 {
		t.Fatalf("expected converted value 7, got %d", converted.Int())
	}

	// Non-convertible value should return original.
	value := reflect.ValueOf("text")
	result := cloneIntoType(value, reflect.TypeOf(struct{}{}))
	if result.Interface() != value.Interface() {
		t.Fatalf("expected original value when conversion not possible")
	}
}

func TestCloneMapExportedHelpers(t *testing.T) {
	if CloneMap(nil) != nil {
		t.Fatalf("expected nil clone for nil input")
	}

	source := map[string]any{"nested": []string{"a"}}
	cloned := CloneMap(source)
	if cloned == nil {
		t.Fatalf("expected cloned map")
	}
	cloned["nested"].([]string)[0] = "b"
	if source["nested"].([]string)[0] != "a" {
		t.Fatalf("expected original map unchanged")
	}

	value := CloneValue(42).(int)
	if value != 42 {
		t.Fatalf("expected primitive clone to retain value")
	}
}

func TestContainerSetOnZeroValueInitialisesMap(t *testing.T) {
	var container Container // zero value to exercise ensurePayload
	if err := container.Set(HookSampleAttributes, PluginID("frog"), map[string]any{"k": "v"}); err != nil {
		t.Fatalf("set failed: %v", err)
	}
	payload, ok := container.Get(HookSampleAttributes, PluginID("frog"))
	if !ok {
		t.Fatalf("expected payload after set")
	}
	if payload.(map[string]any)["k"] != "v" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

// TestIsJSONArray tests the isJSONArray helper function with 0% coverage
func TestIsJSONArray(t *testing.T) {
	// Test with nil
	if isJSONArray(nil) {
		t.Errorf("Expected isJSONArray(nil) to be false")
	}

	// Test with slice
	if !isJSONArray([]string{"a", "b"}) {
		t.Errorf("Expected isJSONArray(slice) to be true")
	}

	// Test with array
	arr := [3]int{1, 2, 3}
	if !isJSONArray(arr) {
		t.Errorf("Expected isJSONArray(array) to be true")
	}

	// Test with non-array types
	if isJSONArray("string") {
		t.Errorf("Expected isJSONArray(string) to be false")
	}

	if isJSONArray(42) {
		t.Errorf("Expected isJSONArray(int) to be false")
	}

	if isJSONArray(map[string]int{"k": 1}) {
		t.Errorf("Expected isJSONArray(map) to be false")
	}
}

// TestPluginsEdgeCases tests additional edge cases for the Plugins method
func TestPluginsEdgeCases(t *testing.T) {
	// Test with multiple plugins to ensure sorting works
	container := NewContainer()

	if err := container.Set(HookOrganismAttributes, PluginID("zebra"), map[string]any{"z": 1}); err != nil {
		t.Fatalf("unexpected Set error: %v", err)
	}
	if err := container.Set(HookOrganismAttributes, PluginID("alpha"), map[string]any{"a": 1}); err != nil {
		t.Fatalf("unexpected Set error: %v", err)
	}
	if err := container.Set(HookOrganismAttributes, PluginID("beta"), map[string]any{"b": 1}); err != nil {
		t.Fatalf("unexpected Set error: %v", err)
	}

	plugins := container.Plugins(HookOrganismAttributes)
	if len(plugins) != 3 {
		t.Fatalf("Expected 3 plugins, got %d", len(plugins))
	}

	// Verify sorting
	expected := []PluginID{"alpha", "beta", "zebra"}
	for i, plugin := range plugins {
		if plugin != expected[i] {
			t.Errorf("Expected plugin %d to be %s, got %s", i, expected[i], plugin)
		}
	}
}

// TestContainerMarshalJSONEdgeCases tests edge cases in JSON marshaling
func TestContainerMarshalJSONEdgeCases(t *testing.T) {
	// Test marshaling empty container
	container := NewContainer()
	data, err := json.Marshal(container)
	if err != nil {
		t.Fatalf("Expected empty container to marshal successfully: %v", err)
	}

	// Should produce empty object
	if string(data) != "{}" {
		t.Errorf("Expected empty container to marshal to '{}', got %s", string(data))
	}

	// Test marshaling container with data
	if err := container.Set(HookOrganismAttributes, PluginCore, map[string]any{"test": true}); err != nil {
		t.Fatalf("Expected container with data to set successfully: %v", err)
	}

	data2, err := json.Marshal(container)
	if err != nil {
		t.Fatalf("Expected container with data to marshal successfully: %v", err)
	}

	// Should contain the hook and plugin data
	var result map[string]any
	if err := json.Unmarshal(data2, &result); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	if result[string(HookOrganismAttributes)] == nil {
		t.Errorf("Expected marshaled container to include hook data")
	}
}

// TestValidateHookPayloadCoverage tests validateHookPayload function coverage
func TestValidateHookPayloadCoverage(t *testing.T) {
	container := NewContainer()

	// Test setting invalid payload types to trigger validateHookPayload error paths
	// These should be rejected by the validation

	// Test with string instead of map[string]any
	err := container.Set(HookOrganismAttributes, PluginCore, "invalid-string")
	if err == nil {
		t.Error("Expected error when setting string payload")
	}

	// Test with integer
	err = container.Set(HookOrganismAttributes, PluginCore, 42)
	if err == nil {
		t.Error("Expected error when setting integer payload")
	}

	// Test with slice instead of map
	err = container.Set(HookOrganismAttributes, PluginCore, []string{"invalid"})
	if err == nil {
		t.Error("Expected error when setting slice payload")
	}

	// Test with nil (this should be allowed)
	err = container.Set(HookOrganismAttributes, PluginCore, nil)
	if err != nil {
		t.Errorf("Expected nil payload to be allowed, got error: %v", err)
	}

	// Test with valid map[string]any (should work)
	err = container.Set(HookOrganismAttributes, PluginCore, map[string]any{"valid": true})
	if err != nil {
		t.Errorf("Expected valid map payload to work, got error: %v", err)
	}
}
