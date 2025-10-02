package frog

import (
	"testing"
	"time"

	"colonycore/pkg/datasetapi"
)

// TestFrogPopulationBinderMissingStore checks binder error when Store is nil.
func TestFrogPopulationBinderMissingStore(t *testing.T) {
	env := datasetapi.Environment{Now: func() time.Time { return time.Unix(0, 0).UTC() }}
	_, err := frogPopulationBinder(env)
	if err == nil {
		t.Fatalf("expected error for missing store")
	}
}
