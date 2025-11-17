package domain

import (
	"fmt"

	"colonycore/pkg/domain/extension"
)

// panicOnExtension panics with a formatted message when the provided error is non-nil.
func panicOnExtension(err error, format string, args ...any) {
	if err == nil {
		return
	}
	panic(fmt.Errorf(format+": %w", append(args, err)...))
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
