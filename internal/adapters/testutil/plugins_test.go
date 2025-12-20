package testutil

import (
	"colonycore/internal/core"
	"colonycore/pkg/domain"
	"testing"
)

// TestInstallFrogPlugin tests the helper function for installing the frog plugin
func TestInstallFrogPlugin(t *testing.T) {
	engine := domain.NewRulesEngine()
	store := core.NewMemoryStore(engine)
	svc := core.NewService(store)
	metadata, err := InstallFrogPlugin(svc)

	if err != nil {
		t.Errorf("Expected no error installing frog plugin, got: %v", err)
	}

	if metadata.Name == "" {
		t.Error("Expected plugin metadata to have a name")
	}

	if metadata.Version == "" {
		t.Error("Expected plugin metadata to have a version")
	}
}
