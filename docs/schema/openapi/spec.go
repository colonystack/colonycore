// Package openapi embeds the generated Entity Model OpenAPI components for
// runtime distribution.
package openapi

import _ "embed"

// EntityModelSpec contains the generated OpenAPI components for the Entity Model.
//
//go:embed entity-model.yaml
var EntityModelSpec []byte

// Spec returns a defensive copy of the embedded Entity Model OpenAPI YAML.
func Spec() []byte {
	return append([]byte(nil), EntityModelSpec...)
}
