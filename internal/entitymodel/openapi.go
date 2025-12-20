// Package entitymodel exposes runtime helpers for serving the generated entity
// model OpenAPI components.
package entitymodel

import (
	entitymodelopenapi "colonycore/docs/schema/openapi"
	"net/http"
)

// OpenAPISpec returns a defensive copy of the embedded Entity Model OpenAPI
// components so callers can safely modify the slice.
func OpenAPISpec() []byte {
	return entitymodelopenapi.Spec()
}

// NewOpenAPIHandler returns an http.Handler that serves the embedded Entity
// Model OpenAPI YAML with a static content-type. It is intended for wiring
// into admin/debug endpoints so downstream clients can fetch the canonical
// contract.
func NewOpenAPIHandler() http.Handler {
	spec := OpenAPISpec()
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(spec)
	})
}
