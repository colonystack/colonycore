package frog

import "testing"

// TestPluginNameVersion ensures the trivial Name and Version accessors are covered.
func TestPluginNameVersion(t *testing.T) {
	p := New()
	if p.Name() != "frog" {
		t.Fatalf("expected name frog, got %s", p.Name())
	}
	if p.Version() == "" {
		t.Fatalf("expected non-empty version")
	}
}
