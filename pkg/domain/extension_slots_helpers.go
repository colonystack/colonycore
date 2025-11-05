package domain

import (
	"fmt"

	"colonycore/pkg/domain/extension"
)

// cloneExtensionMap wraps extension.CloneMap to provide a deep copy for legacy map accessors.
func cloneExtensionMap(values map[string]any) map[string]any {
	return extension.CloneMap(values)
}

// Assign extension maps with clone to avoid shared state.
func assignExtensionMap(values map[string]any) map[string]any {
	// Ensure we don't retain references to the input map.
	return cloneExtensionMap(values)
}

// panicOnExtension panics with a formatted message when the provided error is non-nil.
func panicOnExtension(err error, format string, args ...any) {
	if err == nil {
		return
	}
	panic(fmt.Errorf(format+": %w", append(args, err)...))
}

func slotRaw(slot *extension.Slot) map[string]any {
	if slot == nil {
		return map[string]any{}
	}
	return slot.Raw()
}

func populateContainerFromMap(container *extension.Container, hook extension.Hook, label string, payloads map[string]any) {
	if container == nil {
		return
	}
	for plugin, value := range payloads {
		panicOnExtension(container.Set(hook, extension.PluginID(plugin), value), label)
	}
}

func slotFromPluginPayloads(hook extension.Hook, payloads map[string]any) (*extension.Slot, error) {
	if len(payloads) == 0 {
		return nil, nil
	}
	slot := extension.NewSlot(hook)
	for plugin, value := range payloads {
		if err := slot.Set(extension.PluginID(plugin), value); err != nil {
			return nil, err
		}
	}
	return slot, nil
}
