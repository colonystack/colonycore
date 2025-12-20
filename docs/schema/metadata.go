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

// Entity-model fingerprint content embedded for runtime metadata exposure.
//
//go:embed entity-model.fingerprint.json
var entityModelFingerprint []byte

var (
	metaOnce sync.Once
	metaVer  string
	metaErr  error
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
