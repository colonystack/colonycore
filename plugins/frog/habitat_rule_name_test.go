package frog

import "testing"

func TestFrogHabitatRuleName(t *testing.T) {
	r := frogHabitatRule{}
	if r.Name() != "frog_habitat_warning" {
		t.Fatalf("unexpected name %s", r.Name())
	}
}
