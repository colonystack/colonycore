package core

import (
	"context"
	"strings"
	"testing"
)

// TestServiceRunErrorLogging triggers an operation failure to exercise the logger.Error branch in Service.run.
func TestServiceRunErrorLogging(t *testing.T) {
	log := &captureLogger{}
	svc := NewInMemoryService(NewRulesEngine(), WithLogger(log))
	// Update missing organism to force tx.UpdateOrganism error path.
	if _, _, err := svc.UpdateOrganism(context.Background(), "missing", func(_ *Organism) error { return nil }); err == nil {
		t.Fatalf("expected error updating missing organism")
	}
	// Ensure an error log was recorded.
	var found bool
	for _, c := range log.calls {
		if strings.HasPrefix(c, "e:") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected error log entry, got %v", log.calls)
	}
}
