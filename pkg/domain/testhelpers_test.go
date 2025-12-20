package domain

import (
	"testing"

	"colonycore/pkg/domain/extension"
)

// mustNoError simplifies tests that expect helper methods to succeed.
func mustNoError(t *testing.T, label string, err error) {
	t.Helper()
	if err != nil {
		if label == "" {
			t.Fatalf("unexpected error: %v", err)
		}
		t.Fatalf("%s: %v", label, err)
	}
}

func assertSlotEmpty(t *testing.T, slot *extension.Slot, msg string) {
	t.Helper()
	if slot != nil && len(slot.Plugins()) != 0 {
		if msg == "" {
			t.Fatalf("expected slot to be empty")
		}
		t.Fatalf("%s", msg)
	}
}

func assertContainerEmpty(t *testing.T, container *extension.Container, msg string) {
	t.Helper()
	if container != nil && len(container.Hooks()) != 0 {
		if msg == "" {
			t.Fatalf("expected container to be empty")
		}
		t.Fatalf("%s", msg)
	}
}
