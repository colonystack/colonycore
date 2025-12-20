package domain

import "colonycore/pkg/domain/extension"

// EnsureDefaultAttributes returns the line default attributes slot, initialising
// it with the correct hook identifier when needed.
func (l *Line) EnsureDefaultAttributes() *extension.Slot {
	if l.defaultAttributesSlot == nil {
		l.defaultAttributesSlot = slotFromContainer(extension.HookLineDefaultAttributes, l.ensureExtensionContainer())
	} else {
		_ = l.defaultAttributesSlot.BindHook(extension.HookLineDefaultAttributes)
	}
	return l.defaultAttributesSlot
}

// EnsureExtensionOverrides returns the line extension overrides slot, ensuring
// it is bound to the correct hook.
func (l *Line) EnsureExtensionOverrides() *extension.Slot {
	if l.extensionOverridesSlot == nil {
		l.extensionOverridesSlot = slotFromContainer(extension.HookLineExtensionOverrides, l.ensureExtensionContainer())
	} else {
		_ = l.extensionOverridesSlot.BindHook(extension.HookLineExtensionOverrides)
	}
	return l.extensionOverridesSlot
}

// SetDefaultAttributesSlot installs the provided slot as the default attributes hook payload.
func (l *Line) SetDefaultAttributesSlot(slot *extension.Slot) error {
	return l.setLineSlot(extension.HookLineDefaultAttributes, slot)
}

// SetExtensionOverridesSlot installs the provided slot as the extension overrides hook payload.
func (l *Line) SetExtensionOverridesSlot(slot *extension.Slot) error {
	return l.setLineSlot(extension.HookLineExtensionOverrides, slot)
}

func (l *Line) setLineSlot(hook extension.Hook, incoming *extension.Slot) error {
	if incoming == nil {
		if l.extensions != nil {
			container := l.extensions
			for _, plugin := range container.Plugins(hook) {
				container.Remove(hook, plugin)
			}
			clone, err := cloneContainer(container)
			if err != nil {
				return err
			}
			if len(clone.Hooks()) == 0 {
				l.extensions = nil
			} else {
				l.extensions = &clone
			}
		}
		l.rebindLineSlots()
		return nil
	}

	clone := incoming.Clone()
	if err := clone.BindHook(hook); err != nil {
		return err
	}
	container := l.ensureExtensionContainer()
	for _, plugin := range container.Plugins(hook) {
		container.Remove(hook, plugin)
	}
	for _, plugin := range clone.Plugins() {
		payload, ok := clone.Get(plugin)
		if !ok {
			continue
		}
		if err := container.Set(hook, plugin, payload); err != nil {
			return err
		}
	}
	containerClone, err := cloneContainer(container)
	if err != nil {
		return err
	}
	if len(containerClone.Hooks()) == 0 {
		l.extensions = nil
	} else {
		l.extensions = &containerClone
	}
	l.rebindLineSlots()
	return nil
}
