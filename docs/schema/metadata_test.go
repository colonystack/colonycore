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

func TestEntityModelMetadata(t *testing.T) {
	got, err := EntityModelMetadata()
	if err != nil {
		t.Fatalf("EntityModelMetadata: %v", err)
	}
	if got.Status == "" || got.Source == "" {
		t.Fatalf("expected status and source, got %+v", got)
	}

	var doc metadataDoc
	if err := json.Unmarshal(entityModelSchema, &doc); err != nil {
		t.Fatalf("unmarshal schema: %v", err)
	}
	if got.Status != doc.Metadata.Status || got.Source != doc.Metadata.Source {
		t.Fatalf("metadata mismatch: got %+v want %+v", got, doc.Metadata)
	}
}
