package entitymodel

import "colonycore/docs/schema"

// Version returns the canonical Entity Model schema version derived from the
// generated fingerprint.
func Version() string {
	version, err := schema.EntityModelVersion()
	if err != nil {
		return ""
	}
	return version
}
