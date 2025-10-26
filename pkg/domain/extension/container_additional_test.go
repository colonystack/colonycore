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
