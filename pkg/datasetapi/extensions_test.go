package datasetapi

import "testing"

const (
	extTestMutated = "mutated"
	extTestValue   = "value"
)

func TestExtensionSetBasic(t *testing.T) {
	hooks := NewExtensionHookContext()
	contributors := NewExtensionContributorContext()
	sampleHook := hooks.SampleAttributes()
	corePlugin := contributors.Core()
	raw := map[string]map[string]map[string]any{
		sampleHook.value(): {
			corePlugin.value(): map[string]any{"volume": "5ml"},
			"external":         map[string]any{"notes": "custom"},
		},
	}

	set := NewExtensionSet(raw)

	if got := set.Hooks(); len(got) != 1 || !got[0].Equals(sampleHook) {
		t.Fatalf("unexpected hooks: %+v", got)
	}
	plugins := set.Plugins(sampleHook)
	if len(plugins) != 2 || !plugins[0].Equals(corePlugin) {
		t.Fatalf("unexpected plugins ordering: %+v", plugins)
	}

	core, ok := set.Core(sampleHook)
	if !ok {
		t.Fatalf("expected core payload")
	}
	payload := core.Map()
	payload["volume"] = "10ml"

	fresh, ok := set.Core(sampleHook)
	if !ok || fresh.Map()["volume"] != "5ml" {
		t.Fatalf("expected deep cloned core payload, got %+v", fresh)
	}

	if _, ok := set.Get(sampleHook, contributors.Custom("missing")); ok {
		t.Fatalf("expected missing plugin lookup to return false")
	}

	rawCopy := set.Raw()
	rawCopy[sampleHook.value()][corePlugin.value()]["volume"] = extTestMutated
	fresh, ok = set.Core(sampleHook)
	if !ok || fresh.Map()["volume"] != "5ml" {
		t.Fatalf("raw mutation should not affect stored payload")
	}
}

func TestExtractCoreMap(t *testing.T) {
	hooks := NewExtensionHookContext()
	sampleHook := hooks.SampleAttributes()
	set := newCoreExtensionSet(sampleHook, map[string]any{"key": extTestValue})
	core := extractCoreMap(set, sampleHook)
	if core["key"] != extTestValue {
		t.Fatalf("expected value 'value', got %v", core["key"])
	}
	core["key"] = "mut"
	if extractCoreMap(set, sampleHook)["key"] != extTestValue {
		t.Fatalf("expected extractCoreMap to clone payload")
	}

	if value := extractCoreMap(nil, sampleHook); value != nil {
		t.Fatalf("expected nil for nil set")
	}

	empty := NewExtensionSet(nil)
	if value := extractCoreMap(empty, sampleHook); value != nil {
		t.Fatalf("expected nil for empty set")
	}
}

func TestExtensionSetEmptyPayload(t *testing.T) {
	set := NewExtensionSet(map[string]map[string]map[string]any{})
	if hooks := set.Hooks(); hooks != nil {
		t.Fatalf("expected nil hooks for empty payload, got %+v", hooks)
	}
	hookRef := NewExtensionHookContext().SampleAttributes()
	if plugins := set.Plugins(hookRef); plugins != nil {
		t.Fatalf("expected nil plugins for empty payload, got %+v", plugins)
	}
}

func TestExtractCoreMapNilPayload(t *testing.T) {
	hookRef := NewExtensionHookContext().SampleAttributes()
	corePlugin := NewExtensionContributorContext().Core()
	set := NewExtensionSet(map[string]map[string]map[string]any{
		hookRef.value(): {
			corePlugin.value(): nil,
		},
	})
	if value := extractCoreMap(set, hookRef); value != nil {
		t.Fatalf("expected nil when core payload is nil, got %+v", value)
	}
}
