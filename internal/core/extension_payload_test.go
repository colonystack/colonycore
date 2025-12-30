package core

import "testing"

func TestMapExtensionPayloads(t *testing.T) {
	if got := mapExtensionPayloads(nil); got != nil {
		t.Fatalf("expected nil for nil input, got %+v", got)
	}
	if got := mapExtensionPayloads(map[string]map[string]any{}); got != nil {
		t.Fatalf("expected nil for empty input, got %+v", got)
	}

	raw := map[string]map[string]any{
		"hook.one": {
			"core": map[string]any{"k": "v"},
			"nil":  nil,
			"bad":  "scalar",
		},
		"hook.empty": {},
	}

	out := mapExtensionPayloads(raw)
	if out["hook.one"]["core"]["k"] != "v" {
		t.Fatalf("expected payload value to map through, got %+v", out["hook.one"]["core"])
	}
	if out["hook.one"]["nil"] != nil {
		t.Fatalf("expected nil payload to stay nil, got %+v", out["hook.one"]["nil"])
	}
	if out["hook.one"]["bad"] != nil {
		t.Fatalf("expected non-map payload to map to nil, got %+v", out["hook.one"]["bad"])
	}
	if out["hook.empty"] != nil {
		t.Fatalf("expected empty hook to map to nil, got %+v", out["hook.empty"])
	}
}
