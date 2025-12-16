package openapi

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestSpecReturnsCopyAndMatchesFile(t *testing.T) {
	want, err := os.ReadFile(filepath.Clean(filepath.Join("entity-model.yaml")))
	if err != nil {
		t.Fatalf("read entity-model.yaml: %v", err)
	}

	spec := Spec()
	if len(spec) == 0 {
		t.Fatal("Spec returned empty content")
	}
	if !bytes.Equal(spec, want) {
		t.Fatalf("Spec does not match embedded OpenAPI contents")
	}

	spec[0] ^= 0xFF
	if bytes.Equal(spec, EntityModelSpec) {
		t.Fatalf("Spec did not return a defensive copy")
	}
	if !bytes.Equal(Spec(), want) {
		t.Fatalf("Spec mutation leaked into embedded content")
	}
}
