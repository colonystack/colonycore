package schema

import (
	"encoding/json"
	"testing"
)

func TestEntityModelVersion(t *testing.T) {
	got, err := EntityModelVersion()
	if err != nil {
		t.Fatalf("EntityModelVersion: %v", err)
	}
	if got == "" {
		t.Fatal("expected non-empty entity model version")
	}

	var doc fingerprintDoc
	if err := json.Unmarshal(entityModelFingerprint, &doc); err != nil {
		t.Fatalf("unmarshal fingerprint: %v", err)
	}
	if got != doc.Version {
		t.Fatalf("version mismatch: got %q want %q", got, doc.Version)
	}
}
