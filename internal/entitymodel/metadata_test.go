package entitymodel

import (
	"colonycore/docs/schema"
	"testing"
)

func TestVersionReturnsFingerprintValue(t *testing.T) {
	got := Version()
	if got == "" {
		t.Fatal("expected entity model version")
	}
	want, err := schema.EntityModelVersion()
	if err != nil {
		t.Fatalf("schema.EntityModelVersion: %v", err)
	}
	if got != want {
		t.Fatalf("version mismatch: got %q want %q", got, want)
	}
}
