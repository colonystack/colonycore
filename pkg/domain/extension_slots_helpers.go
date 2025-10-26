package domain

import "colonycore/pkg/domain/extension"

// cloneExtensionMap wraps extension.CloneMap to provide a deep copy for legacy map accessors.
func cloneExtensionMap(values map[string]any) map[string]any {
	return extension.CloneMap(values)
}

// Assign extension maps with clone to avoid shared state.
func assignExtensionMap(values map[string]any) map[string]any {
	// Ensure we don't retain references to the input map.
	return cloneExtensionMap(values)
}
