package core

import (
	"colonycore/pkg/domain"
	"testing"
)

// strPtr is a lightweight helper for pointer fields in core package tests.
func strPtr(v string) *string {
	return &v
}

func mustChangePayload[T any](t *testing.T, value T) domain.ChangePayload {
	t.Helper()
	payload, err := domain.NewChangePayloadFromValue(value)
	if err != nil {
		t.Fatalf("build change payload: %v", err)
	}
	return payload
}
