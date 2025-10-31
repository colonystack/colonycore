package pluginapi

import "testing"

func TestExtensionSetBasic(t *testing.T) {
	hooks := NewExtensionHookContext()
	contributors := NewExtensionContributorContext()
	organismHook := hooks.OrganismAttributes()
	facilityHook := hooks.FacilityEnvironmentBaselines()
	corePlugin := contributors.Core()
	raw := map[string]map[string]any{
		organismHook.String(): {
			"alpha": map[string]any{"flag": true},
		},
		facilityHook.String(): {
			corePlugin.String(): map[string]any{"temp": 21},
		},
	}

	set := NewExtensionSet(raw)

	resolvedHooks := set.Hooks()
	if len(resolvedHooks) != 2 || !resolvedHooks[0].Equals(facilityHook) || !resolvedHooks[1].Equals(organismHook) {
		t.Fatalf("unexpected hooks ordering: %+v", resolvedHooks)
	}

	plugins := set.Plugins(facilityHook)
	if len(plugins) != 1 || !plugins[0].Equals(corePlugin) {
		t.Fatalf("expected core plugin, got %+v", plugins)
	}

	value, ok := set.Core(facilityHook)
	if !ok {
		t.Fatalf("expected core payload")
	}
	payload := value.(map[string]any)
	payload["temp"] = 30

	fresh, ok := set.Core(facilityHook)
	if !ok {
		t.Fatalf("expected payload after mutation")
	}
	if fresh.(map[string]any)["temp"] != 21 {
		t.Fatalf("expected deep copy protection, got %v", fresh)
	}

	if _, ok := set.Get(organismHook, contributors.Custom("missing")); ok {
		t.Fatal("expected missing plugin lookup to return false")
	}

	rawCopy := set.Raw()
	rawCopy[organismHook.String()]["alpha"].(map[string]any)["flag"] = false
	value, ok = set.Get(organismHook, contributors.Custom("alpha"))
	if !ok || value.(map[string]any)["flag"] != true {
		t.Fatalf("expected original payload unaffected, got %+v", value)
	}
}

func TestExtensionSetNil(t *testing.T) {
	set := NewExtensionSet(nil)
	if hooks := set.Hooks(); hooks != nil {
		t.Fatalf("expected nil hooks, got %+v", hooks)
	}
	if _, ok := set.Get(NewExtensionHookContext().OrganismAttributes(), NewExtensionContributorContext().Core()); ok {
		t.Fatal("expected missing payload for nil set")
	}
}
