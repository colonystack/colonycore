package entitymodel

import (
	"colonycore/docs/schema"
	"strconv"
	"strings"
)

// Metadata aggregates the canonical Entity Model metadata exposed at runtime.
type Metadata struct {
	Version string
	Status  string
	Source  string
}

var (
	schemaVersionFn  = schema.EntityModelVersion
	schemaMetadataFn = schema.EntityModelMetadata
)

// Version returns the canonical Entity Model schema version derived from the
// generated fingerprint.
func Version() string {
	version, err := schemaVersionFn()
	if err != nil {
		return ""
	}
	return version
}

// MetadataInfo returns the canonical Entity Model metadata. Errors are
// swallowed to avoid coupling callers to schema parsing; fields are empty on
// failure.
func MetadataInfo() Metadata {
	version, _ := schemaVersionFn()
	meta, _ := schemaMetadataFn()
	return Metadata{
		Version: version,
		Status:  meta.Status,
		Source:  meta.Source,
	}
}

// IsCompatibleMajor reports whether the embedded Entity Model major version
// matches the expected value. It returns false when the version is missing or
// unparsable.
func IsCompatibleMajor(expected int) bool {
	v := Version()
	if v == "" {
		return false
	}
	parts := strings.SplitN(v, ".", 2)
	if len(parts) == 0 {
		return false
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return false
	}
	return major == expected
}
