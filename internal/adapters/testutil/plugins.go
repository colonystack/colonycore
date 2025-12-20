// Package testutil hosts helper utilities for dataset adapter tests.
// It intentionally encapsulates access to runtime plugins so the production
// adapter package never depends on plugin implementations directly.
package testutil

import (
	"colonycore/internal/core"
	"colonycore/plugins/frog"
)

// InstallFrogPlugin installs the reference frog plugin and returns its metadata.
// Tests rely on this helper to access dataset templates without importing
// runtime plugin packages, preserving the adapter-layer boundary.
func InstallFrogPlugin(svc *core.Service) (core.PluginMetadata, error) {
	return svc.InstallPlugin(frog.New())
}
