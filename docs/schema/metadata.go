// Package schema exposes embedded Entity Model metadata (version) for runtime use.
package schema

import (
	_ "embed"
	"encoding/json"
	"sync"
)

type fingerprintDoc struct {
	Version string `json:"version"`
}

// Metadata captures the high-level metadata block from the canonical
// entity-model JSON.
type Metadata struct {
	Source string `json:"source"`
	Status string `json:"status"`
}

type metadataDoc struct {
	Metadata Metadata `json:"metadata"`
}

// Entity-model fingerprint content embedded for runtime metadata exposure.
//
//go:embed entity-model.fingerprint.json
var entityModelFingerprint []byte

// Canonical entity-model JSON content embedded for accessing schema metadata.
//
//go:embed entity-model.json
var entityModelSchema []byte

var (
	metaOnce sync.Once
	metaVer  string
	metaErr  error

	schemaOnce sync.Once
	schemaMeta Metadata
	schemaErr  error
)

// EntityModelVersion returns the canonical schema version declared in the
// generated fingerprint (source of truth: docs/schema/entity-model.json).
func EntityModelVersion() (string, error) {
	metaOnce.Do(func() {
		var fp fingerprintDoc
		metaErr = json.Unmarshal(entityModelFingerprint, &fp)
		if metaErr == nil {
			metaVer = fp.Version
		}
	})
	return metaVer, metaErr
}

// EntityModelMetadata returns the schema metadata (status, source) declared in
// the canonical entity-model JSON.
func EntityModelMetadata() (Metadata, error) {
	schemaOnce.Do(func() {
		var doc metadataDoc
		schemaErr = json.Unmarshal(entityModelSchema, &doc)
		if schemaErr == nil {
			schemaMeta = doc.Metadata
		}
	})
	return schemaMeta, schemaErr
}
