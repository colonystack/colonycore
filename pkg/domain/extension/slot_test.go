package extension

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
)

const (
	testValue = "value"
	testNull  = "null"
)

func TestSlotSetGet(t *testing.T) {
	slot := NewSlot(HookLineDefaultAttributes)
	if err := slot.Set(PluginID("frog"), map[string]any{"field": testValue}); err != nil {
		t.Fatalf("set failed: %v", err)
	}
	payload, ok := slot.Get(PluginID("frog"))
	if !ok {
		t.Fatalf("expected payload")
	}
	if payload.(map[string]any)["field"] != testValue {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

func TestSlotSetRejectsInvalidShape(t *testing.T) {
	slot := NewSlot(HookOrganismAttributes)
	if err := slot.Set(PluginID("frog"), []string{"invalid"}); err == nil {
		t.Fatalf("expected shape validation error")
	}
}

func TestSlotSetRequiresHookBinding(t *testing.T) {
	var slot Slot
	if err := slot.Set(PluginID("frog"), map[string]any{"field": testValue}); err != ErrUnboundSlot {
		t.Fatalf("expected ErrUnboundSlot, got %v", err)
	}
}

func TestSlotClone(t *testing.T) {
	slot := NewSlot(HookStrainAttributes)
	_ = slot.Set(PluginID("frog"), map[string]any{"field": testValue})

	clone := slot.Clone()
	if clone == nil {
		t.Fatalf("expected clone")
	}
	if err := clone.BindHook(HookStrainAttributes); err != nil {
		t.Fatalf("bind hook failed: %v", err)
	}
	if err := clone.Set(PluginID("frog"), map[string]any{"field": "mutated"}); err != nil {
		t.Fatalf("mutate clone failed: %v", err)
	}
	payload, _ := slot.Get(PluginID("frog"))
	if payload.(map[string]any)["field"] != testValue {
		t.Fatalf("expected original payload unchanged")
	}
}

func TestSlotMarshalRoundTrip(t *testing.T) {
	slot := NewSlot(HookGenotypeMarkerAttributes)
	_ = slot.Set(PluginID("frog"), map[string]any{"field": testValue})

	data, err := json.Marshal(slot)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded Slot
	if err := decoded.BindHook(HookGenotypeMarkerAttributes); err != nil {
		t.Fatalf("bind hook failed: %v", err)
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	payload, ok := decoded.Get(PluginID("frog"))
	if !ok {
		t.Fatalf("expected payload")
	}
	if payload.(map[string]any)["field"] != testValue {
		t.Fatalf("unexpected payload after round-trip")
	}
}

func TestSlotHookAndBindErrors(t *testing.T) {
	slot := NewSlot(HookLineDefaultAttributes)
	if slot.Hook() != HookLineDefaultAttributes {
		t.Fatalf("expected hook %q, got %q", HookLineDefaultAttributes, slot.Hook())
	}

	var empty Slot
	if err := empty.BindHook(Hook("entity.invalid")); !errors.Is(err, ErrUnknownHook) {
		t.Fatalf("expected ErrUnknownHook, got %v", err)
	}
	if err := empty.BindHook(HookStrainAttributes); err != nil {
		t.Fatalf("bind hook failed: %v", err)
	}
	if empty.Hook() != HookStrainAttributes {
		t.Fatalf("expected updated hook")
	}
}

func TestSlotRemovePluginsAndGetMissing(t *testing.T) {
	uninitialised := &Slot{}
	if err := uninitialised.BindHook(HookSampleAttributes); err != nil {
		t.Fatalf("bind hook failed: %v", err)
	}
	uninitialised.Remove(PluginID("frog"))

	slot := NewSlot(HookSampleAttributes)
	if _, ok := slot.Get(PluginID("missing")); ok {
		t.Fatalf("expected missing payload")
	}
	if err := slot.Set(PluginID("beta"), map[string]any{"ok": true}); err != nil {
		t.Fatalf("set failed: %v", err)
	}
	if err := slot.Set(PluginID("alpha"), map[string]any{"ok": true}); err != nil {
		t.Fatalf("set failed: %v", err)
	}
	plugins := slot.Plugins()
	expected := []PluginID{"alpha", "beta"}
	if !reflect.DeepEqual(plugins, expected) {
		t.Fatalf("unexpected plugin order: %v", plugins)
	}

	slot.Remove(PluginID("beta"))
	slot.Remove(PluginID("alpha"))
	if slot.values != nil {
		t.Fatalf("expected slot values to be nil after removing all entries")
	}
	if len(slot.Raw()) != 0 {
		t.Fatalf("expected empty raw map")
	}
}

func TestSlotMarshalNilPointer(t *testing.T) {
	var slot *Slot
	bytes, err := slot.MarshalJSON()
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if string(bytes) != testNull {
		t.Fatalf("expected null JSON, got %s", bytes)
	}
}

func TestSlotUnmarshalEdgeCases(t *testing.T) {
	slot := NewSlot(HookLineExtensionOverrides)
	if err := json.Unmarshal([]byte(testNull), slot); err != nil {
		t.Fatalf("unmarshal null failed: %v", err)
	}
	if len(slot.Raw()) != 0 {
		t.Fatalf("expected empty raw map after null unmarshal")
	}

	if err := json.Unmarshal([]byte(`{"":{}}`), slot); !errors.Is(err, ErrEmptyPlugin) {
		t.Fatalf("expected ErrEmptyPlugin, got %v", err)
	}

	if err := json.Unmarshal([]byte(`{}`), slot); err != nil {
		t.Fatalf("unmarshal empty object failed: %v", err)
	}
	if len(slot.Raw()) != 0 {
		t.Fatalf("expected empty raw map after empty object")
	}
}

func TestSlotUnmarshalRequiresHookBinding(t *testing.T) {
	var slot Slot
	if err := json.Unmarshal([]byte(`{"plugin":{}}`), &slot); err != ErrUnboundSlot {
		t.Fatalf("expected ErrUnboundSlot, got %v", err)
	}
}

func TestSlotUnmarshalRejectsInvalidShape(t *testing.T) {
	slot := NewSlot(HookSampleAttributes)
	if err := json.Unmarshal([]byte(`{"plugin":["invalid"]}`), slot); err == nil {
		t.Fatalf("expected shape validation error")
	}
}

func TestSlotSetOnZeroValueInitialisesMap(t *testing.T) {
	var slot Slot // zero value without constructor to exercise ensureMap
	if err := slot.BindHook(HookOrganismAttributes); err != nil {
		t.Fatalf("bind hook failed: %v", err)
	}
	if err := slot.Set(PluginID("frog"), map[string]any{"k": "v"}); err != nil {
		t.Fatalf("set failed: %v", err)
	}
	payload, ok := slot.Get(PluginID("frog"))
	if !ok || payload.(map[string]any)["k"] != "v" {
		t.Fatalf("expected stored payload after set, got %v", payload)
	}
}
