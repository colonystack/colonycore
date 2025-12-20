package entitymodel

import (
	"colonycore/docs/schema"
	"errors"
	"strconv"
	"strings"
	"testing"
)

func TestMetadataInfo(t *testing.T) {
	info := MetadataInfo()
	if info.Version != Version() {
		t.Fatalf("version mismatch: got %q want %q", info.Version, Version())
	}
	meta, err := schema.EntityModelMetadata()
	if err != nil {
		t.Fatalf("schema metadata: %v", err)
	}
	if info.Status != meta.Status || info.Source != meta.Source {
		t.Fatalf("metadata mismatch: got %+v want %+v", info, meta)
	}
}

func TestIsCompatibleMajor(t *testing.T) {
	v := Version()
	if v == "" {
		t.Skip("version not set")
	}
	majorPart := strings.SplitN(v, ".", 2)[0]
	major, err := strconv.Atoi(majorPart)
	if err != nil {
		t.Fatalf("parse major: %v", err)
	}
	if !IsCompatibleMajor(major) {
		t.Fatalf("expected compatibility for major %d", major)
	}
	if IsCompatibleMajor(major + 1) {
		t.Fatalf("unexpected compatibility for major %d", major+1)
	}
}

func TestIsCompatibleMajorHandlesInvalidVersion(t *testing.T) {
	orig := schemaVersionFn
	schemaVersionFn = func() (string, error) { return "bad.version", nil }
	t.Cleanup(func() { schemaVersionFn = orig })

	if IsCompatibleMajor(1) {
		t.Fatalf("expected incompatible result for invalid version")
	}
}

func TestIsCompatibleMajorHandlesEmptyVersion(t *testing.T) {
	orig := schemaVersionFn
	schemaVersionFn = func() (string, error) { return "", nil }
	t.Cleanup(func() { schemaVersionFn = orig })

	if IsCompatibleMajor(1) {
		t.Fatalf("expected incompatible result for empty version")
	}
}

func TestMajorVersion(t *testing.T) {
	v := Version()
	if v == "" {
		t.Skip("version not set")
	}
	major, ok := MajorVersion()
	if !ok {
		t.Fatalf("expected to parse major from %q", v)
	}
	parsed, err := strconv.Atoi(strings.SplitN(v, ".", 2)[0])
	if err != nil {
		t.Fatalf("parse major: %v", err)
	}
	if major != parsed {
		t.Fatalf("expected major %d, got %d", parsed, major)
	}
}

func TestMajorVersionHandlesInvalidVersion(t *testing.T) {
	orig := schemaVersionFn
	schemaVersionFn = func() (string, error) { return "bad.version", nil }
	t.Cleanup(func() { schemaVersionFn = orig })

	if major, ok := MajorVersion(); ok || major != 0 {
		t.Fatalf("expected invalid parse to report false, got %d", major)
	}
}

func TestMajorVersionHandlesEmptyVersion(t *testing.T) {
	orig := schemaVersionFn
	schemaVersionFn = func() (string, error) { return "", nil }
	t.Cleanup(func() { schemaVersionFn = orig })

	if major, ok := MajorVersion(); ok || major != 0 {
		t.Fatalf("expected empty version to report false, got %d", major)
	}
}

func TestVersionFallbackOnError(t *testing.T) {
	orig := schemaVersionFn
	schemaVersionFn = func() (string, error) { return "", errors.New("boom") }
	t.Cleanup(func() { schemaVersionFn = orig })

	if got := Version(); got != "" {
		t.Fatalf("expected empty version on error, got %q", got)
	}
}

func TestMetadataInfoHandlesErrors(t *testing.T) {
	origVersion := schemaVersionFn
	origMetadata := schemaMetadataFn
	schemaVersionFn = func() (string, error) { return "", errors.New("boom") }
	schemaMetadataFn = func() (schema.Metadata, error) {
		return schema.Metadata{}, errors.New("boom")
	}
	t.Cleanup(func() {
		schemaVersionFn = origVersion
		schemaMetadataFn = origMetadata
	})

	info := MetadataInfo()
	if info.Version != "" || info.Status != "" || info.Source != "" {
		t.Fatalf("expected empty metadata on error, got %+v", info)
	}
}
