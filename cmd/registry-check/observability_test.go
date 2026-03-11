package main

import (
	"context"
	"testing"

	"colonycore/internal/observability"
)

type captureRegistryEvents struct {
	events []observability.Event
}

func (c *captureRegistryEvents) Record(_ context.Context, event observability.Event) {
	c.events = append(c.events, event)
}

func (c *captureRegistryEvents) has(name, status string) bool {
	for _, event := range c.events {
		if event.Name == name && event.Status == status {
			return true
		}
	}
	return false
}

func TestRunWithRecorderEmitsSuccessEvent(t *testing.T) {
	docPath := writeTestFile(t, "obs-success-doc.md", "# Test\n- Status: Draft\n")
	regPath := writeTestFile(t, "obs-success-registry.yaml", "documents:\n  - id: RFC-200\n    type: RFC\n    title: Success\n    status: Draft\n    path: "+docPath+"\n")
	events := &captureRegistryEvents{}

	if err := runWithRecorder(context.Background(), regPath, events); err != nil {
		t.Fatalf("runWithRecorder success: %v", err)
	}
	if !events.has("registry.validate", observability.StatusSuccess) {
		t.Fatalf("expected registry.validate success event, got %+v", events.events)
	}
	if !events.has("registry.document.validate", observability.StatusSuccess) {
		t.Fatalf("expected registry.document.validate success event, got %+v", events.events)
	}
}

func TestRunWithRecorderEmitsFailureEvent(t *testing.T) {
	docPath := writeTestFile(t, "obs-fail-doc.md", "# Test\n- Status: Accepted\n")
	regPath := writeTestFile(t, "obs-fail-registry.yaml", "documents:\n  - id: RFC-201\n    type: RFC\n    title: Failure\n    status: Draft\n    path: "+docPath+"\n")
	events := &captureRegistryEvents{}

	if err := runWithRecorder(context.Background(), regPath, events); err == nil {
		t.Fatalf("expected validation failure")
	}
	if !events.has("registry.document.status", observability.StatusError) {
		t.Fatalf("expected registry.document.status error event, got %+v", events.events)
	}
	if !events.has("registry.validate", observability.StatusError) {
		t.Fatalf("expected registry.validate error event, got %+v", events.events)
	}
}
