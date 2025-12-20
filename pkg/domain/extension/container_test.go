package extension

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestContainerSetGetIsolation(t *testing.T) {
	container := NewContainer()
	payload := map[string]any{
		"metrics": []int{1, 2, 3},
	}
	if err := container.Set(HookOrganismAttributes, PluginID("frog"), payload); err != nil {
		t.Fatalf("set failed: %v", err)
	}

	// Mutate the original payload to ensure the container keeps its own copy.
	payload["metrics"].([]int)[0] = 99

	raw, ok := container.Get(HookOrganismAttributes, PluginID("frog"))
	if !ok {
		t.Fatalf("expected payload for hook %q", HookOrganismAttributes)
	}
	rawMap, ok := raw.(map[string]any)
	if !ok {
		t.Fatalf("expected map payload, got %T", raw)
	}
	values, ok := rawMap["metrics"].([]int)
	if !ok {
		t.Fatalf("expected []int payload, got %T", rawMap["metrics"])
	}
	if values[0] != 1 {
		t.Fatalf("expected container copy to remain unchanged, got %v", values)
	}
}

func TestContainerCloneIndependence(t *testing.T) {
	container := NewContainer()
	if err := container.Set(HookSampleAttributes, PluginID("frog"), map[string]any{
		"tissue_type": "toe_clip",
	}); err != nil {
		t.Fatalf("set failed: %v", err)
	}

	clone, err := container.Clone()
	if err != nil {
		t.Fatalf("clone failed: %v", err)
	}

	clone.Remove(HookSampleAttributes, PluginID("frog"))

	if _, ok := clone.Get(HookSampleAttributes, PluginID("frog")); ok {
		t.Fatalf("expected clone payload to be removed")
	}
	if _, ok := container.Get(HookSampleAttributes, PluginID("frog")); !ok {
		t.Fatalf("expected original container payload to remain")
	}
}

func TestCloneWithEmptyContainer(t *testing.T) {
	clone, err := NewContainer().Clone()
	if err != nil {
		t.Fatalf("clone failed: %v", err)
	}
	if hooks := clone.Hooks(); hooks != nil {
		t.Fatalf("expected no hooks, got %v", hooks)
	}
}

func TestContainerJSONRoundTrip(t *testing.T) {
	container := NewContainer()
	if err := container.Set(HookObservationData, PluginID("frog"), map[string]any{
		"skin_score": "4",
		"notes": map[string]any{
			"context": "quarantine",
		},
	}); err != nil {
		t.Fatalf("set failed: %v", err)
	}

	bytes, err := json.Marshal(container)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded Container
	if err := json.Unmarshal(bytes, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if !reflect.DeepEqual(container.Raw(), decoded.Raw()) {
		t.Fatalf("round-trip mismatch: %#v vs %#v", container.Raw(), decoded.Raw())
	}
}

func TestFromRawRejectsUnknownHook(t *testing.T) {
	_, err := FromRaw(map[string]map[string]any{
		"entity.unknown": {
			"frog": map[string]any{"field": testValue},
		},
	})
	if err == nil {
		t.Fatalf("expected error for unknown hook")
	}
}

func TestFromRawStoresPayloads(t *testing.T) {
	container, err := FromRaw(map[string]map[string]any{
		string(HookOrganismAttributes): {
			"frog": map[string]any{"field": testValue},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	payload, ok := container.Get(HookOrganismAttributes, PluginID("frog"))
	if !ok {
		t.Fatalf("expected payload to be present")
	}
	got := payload.(map[string]any)["field"]
	if got != testValue {
		t.Fatalf("unexpected payload: %v", got)
	}
}

func TestSetRejectsInvalidInputs(t *testing.T) {
	container := NewContainer()
	if err := container.Set(Hook("entity.invalid"), PluginID("frog"), map[string]any{}); err == nil {
		t.Fatalf("expected error for unknown hook")
	}
	if err := container.Set(HookOrganismAttributes, PluginID(""), map[string]any{}); err != ErrEmptyPlugin {
		t.Fatalf("expected ErrEmptyPlugin, got %v", err)
	}
}

func TestContainerSetRejectsInvalidShape(t *testing.T) {
	container := NewContainer()
	if err := container.Set(HookOrganismAttributes, PluginID("frog"), []string{"not", "object"}); err == nil {
		t.Fatalf("expected shape validation error")
	}
}

func TestUnmarshalRejectsEmptyPlugin(t *testing.T) {
	var container Container
	data := []byte(`{"` + string(HookOrganismAttributes) + `":{"":{}}}`)
	if err := container.UnmarshalJSON(data); err != ErrEmptyPlugin {
		t.Fatalf("expected ErrEmptyPlugin, got %v", err)
	}
}

func TestContainerUnmarshalRejectsInvalidShape(t *testing.T) {
	data := []byte(`{"` + string(HookOrganismAttributes) + `":{"plugin":["not","object"]}}`)
	var container Container
	if err := container.UnmarshalJSON(data); err == nil {
		t.Fatalf("expected shape validation error during unmarshal")
	}
}

func TestHooksAndPluginsOrdering(t *testing.T) {
	container := NewContainer()
	if err := container.Set(HookSampleAttributes, PluginID("beta"), map[string]any{"ok": true}); err != nil {
		t.Fatalf("set failed: %v", err)
	}
	if err := container.Set(HookSampleAttributes, PluginID("alpha"), map[string]any{"ok": true}); err != nil {
		t.Fatalf("set failed: %v", err)
	}
	if err := container.Set(HookOrganismAttributes, PluginID("frog"), map[string]any{"ok": true}); err != nil {
		t.Fatalf("set failed: %v", err)
	}

	hooks := container.Hooks()
	expectedHooks := []Hook{HookOrganismAttributes, HookSampleAttributes}
	if !reflect.DeepEqual(hooks, expectedHooks) {
		t.Fatalf("unexpected hook order: %v", hooks)
	}

	plugins := container.Plugins(HookSampleAttributes)
	expectedPlugins := []PluginID{"alpha", "beta"}
	if !reflect.DeepEqual(plugins, expectedPlugins) {
		t.Fatalf("unexpected plugin order: %v", plugins)
	}
}

func TestRawIsImmutable(t *testing.T) {
	container := NewContainer()
	if err := container.Set(HookOrganismAttributes, PluginID("frog"), map[string]any{"field": testValue}); err != nil {
		t.Fatalf("set failed: %v", err)
	}
	raw := container.Raw()
	raw[string(HookOrganismAttributes)]["frog"] = map[string]any{"field": "mutated"}
	payload, _ := container.Get(HookOrganismAttributes, PluginID("frog"))
	if payload.(map[string]any)["field"] != testValue {
		t.Fatalf("expected container payload to remain unchanged, got %v", payload)
	}
}

func TestKnownHooksAndParse(t *testing.T) {
	hooks := KnownHooks()
	if len(hooks) == 0 {
		t.Fatalf("expected registered hooks")
	}
	for _, hook := range hooks {
		if !IsKnownHook(hook) {
			t.Fatalf("hook %q not recognised", hook)
		}
		parsed, err := ParseHook(string(hook))
		if err != nil {
			t.Fatalf("parse failed: %v", err)
		}
		if parsed != hook {
			t.Fatalf("parse returned different hook: %v", parsed)
		}
	}
}

func TestCloneMapIndependence(t *testing.T) {
	original := map[string]any{
		"slice": []any{map[string]any{"k": "v"}},
	}
	cloned := CloneMap(original)
	if cloned == nil {
		t.Fatalf("expected cloned map")
	}
	cloned["slice"].([]any)[0].(map[string]any)["k"] = "mutated"
	if original["slice"].([]any)[0].(map[string]any)["k"] != "v" {
		t.Fatalf("expected original map untouched")
	}
}

func TestSpecMetadata(t *testing.T) {
	spec, ok := Spec(HookOrganismAttributes)
	if !ok {
		t.Fatalf("expected spec for hook %q", HookOrganismAttributes)
	}
	if spec.Entity != "organism" {
		t.Fatalf("unexpected entity: %s", spec.Entity)
	}
	if spec.Field != "attributes" {
		t.Fatalf("unexpected field: %s", spec.Field)
	}
	if spec.DomainMember != "domain.Organism.CoreAttributes" {
		t.Fatalf("unexpected domain member: %s", spec.DomainMember)
	}
	if spec.Shape != ShapeObject {
		t.Fatalf("unexpected shape: %s", spec.Shape)
	}
}

func TestKnownHooksSnapshot(t *testing.T) {
	expected := []Hook{
		HookBreedingUnitPairingAttributes,
		HookFacilityEnvironmentBaselines,
		HookGenotypeMarkerAttributes,
		HookLineDefaultAttributes,
		HookLineExtensionOverrides,
		HookObservationData,
		HookOrganismAttributes,
		HookSampleAttributes,
		HookStrainAttributes,
		HookSupplyItemAttributes,
	}
	got := KnownHooks()
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("known hooks mismatch: got %v expected %v", got, expected)
	}
}

func TestMergeHookPayloadSkipsNil(t *testing.T) {
	container, err := FromRaw(map[string]map[string]any{
		string(HookOrganismAttributes): nil,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hooks := container.Hooks(); len(hooks) != 0 {
		t.Fatalf("expected no hooks, got %v", hooks)
	}
}

func TestRemoveNoopPaths(t *testing.T) {
	container := NewContainer()
	container.Remove(HookOrganismAttributes, PluginID("frog")) // ensure noop on empty

	if err := container.Set(HookOrganismAttributes, PluginID("frog"), map[string]any{"field": testValue}); err != nil {
		t.Fatalf("set failed: %v", err)
	}
	container.Remove(HookOrganismAttributes, PluginID("frog"))
	if len(container.Hooks()) != 0 {
		t.Fatalf("expected hook map to be cleared after removal")
	}
}

func TestGetAbsentPaths(t *testing.T) {
	container := NewContainer()
	if _, ok := container.Get(HookOrganismAttributes, PluginID("frog")); ok {
		t.Fatalf("expected empty container to return no payload")
	}
	if err := container.Set(HookOrganismAttributes, PluginID("frog"), map[string]any{"field": testValue}); err != nil {
		t.Fatalf("set failed: %v", err)
	}
	if _, ok := container.Get(HookOrganismAttributes, PluginID("rat")); ok {
		t.Fatalf("expected missing plugin to return false")
	}
}

func TestPluginsAndHooksNilPaths(t *testing.T) {
	container := NewContainer()
	if plugins := container.Plugins(HookOrganismAttributes); plugins != nil {
		t.Fatalf("expected nil plugins, got %v", plugins)
	}
	if hooks := container.Hooks(); hooks != nil {
		t.Fatalf("expected nil hooks, got %v", hooks)
	}
}

func TestCloneErrorPath(t *testing.T) {
	container := NewContainer()
	if err := container.Set(HookOrganismAttributes, PluginID("frog"), map[string]any{
		"bad": make(chan int),
	}); err != nil {
		t.Fatalf("set failed: %v", err)
	}
	if _, err := container.Clone(); err == nil {
		t.Fatalf("expected clone to fail for non-JSON payload")
	}
}

func TestCloneValueCoversVariants(t *testing.T) {
	container := NewContainer()
	payload := map[string]any{
		"mixed": []any{
			map[string]any{"nested": []string{"a", "b"}},
			map[string]string{"alpha": "beta"},
		},
		"floats":  []float64{1.5, 2.5},
		"bools":   []bool{true, false},
		"words":   []string{"a", "b"},
		"counts":  map[string]int{"a": 1},
		"weights": map[string]float64{"a": 1.5},
		"flags":   map[string]bool{"a": true},
		"ints":    []int{1, 2},
	}
	if err := container.Set(HookOrganismAttributes, PluginID("frog"), payload); err != nil {
		t.Fatalf("set failed: %v", err)
	}
	got, _ := container.Get(HookOrganismAttributes, PluginID("frog"))
	result := got.(map[string]any)

	// mutate clone to ensure deep copy includes all branches
	result["mixed"].([]any)[0].(map[string]any)["nested"].([]string)[0] = "z"
	result["words"].([]string)[0] = "z"
	result["counts"].(map[string]int)["a"] = 99
	result["ints"].([]int)[0] = 99
	result["floats"].([]float64)[0] = 9.9
	result["bools"].([]bool)[0] = false
	result["weights"].(map[string]float64)["a"] = 9.9
	result["flags"].(map[string]bool)["a"] = false

	original, _ := container.Get(HookOrganismAttributes, PluginID("frog"))
	orig := original.(map[string]any)
	if orig["mixed"].([]any)[0].(map[string]any)["nested"].([]string)[0] != "a" {
		t.Fatalf("expected mixed clone to be isolated")
	}
	if orig["words"].([]string)[0] != "a" {
		t.Fatalf("expected words clone to be isolated")
	}
	if orig["counts"].(map[string]int)["a"] != 1 {
		t.Fatalf("expected counts clone to be isolated")
	}
	if orig["ints"].([]int)[0] != 1 {
		t.Fatalf("expected ints clone to be isolated")
	}
	if orig["floats"].([]float64)[0] != 1.5 {
		t.Fatalf("expected floats clone to be isolated")
	}
	if orig["bools"].([]bool)[0] != true {
		t.Fatalf("expected bools clone to be isolated")
	}
	if orig["weights"].(map[string]float64)["a"] != 1.5 {
		t.Fatalf("expected weights clone to be isolated")
	}
	if orig["flags"].(map[string]bool)["a"] != true {
		t.Fatalf("expected flags clone to be isolated")
	}
}
