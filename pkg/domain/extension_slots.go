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

// EnsureAttributes returns the strain attributes slot, initialising it if necessary.
func (s *Strain) EnsureAttributes() *extension.Slot {
	if s.attributesSlot == nil {
		s.attributesSlot = slotFromContainer(extension.HookStrainAttributes, s.ensureExtensionContainer())
	} else {
		_ = s.attributesSlot.BindHook(extension.HookStrainAttributes)
	}
	return s.attributesSlot
}

// EnsureAttributes returns the genotype marker attributes slot, initialising it if necessary.
func (g *GenotypeMarker) EnsureAttributes() *extension.Slot {
	if g.attributesSlot == nil {
		g.attributesSlot = slotFromContainer(extension.HookGenotypeMarkerAttributes, g.ensureExtensionContainer())
	} else {
		_ = g.attributesSlot.BindHook(extension.HookGenotypeMarkerAttributes)
	}
	return g.attributesSlot
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

// SetAttributesSlot persists the provided attributes slot onto the strain entity.
func (s *Strain) SetAttributesSlot(slot *extension.Slot) error {
	if slot == nil {
		s.attributesSlot = nil
		s.extensions = nil
		return nil
	}
	clone := slot.Clone()
	if err := clone.BindHook(extension.HookStrainAttributes); err != nil {
		return err
	}
	container, err := containerFromSlot(extension.HookStrainAttributes, clone)
	if err != nil {
		return err
	}
	s.attributesSlot = clone
	s.extensions = container
	return nil
}

// SetAttributesSlot persists the provided attributes slot onto the genotype marker entity.
func (g *GenotypeMarker) SetAttributesSlot(slot *extension.Slot) error {
	if slot == nil {
		g.attributesSlot = nil
		g.extensions = nil
		return nil
	}
	clone := slot.Clone()
	if err := clone.BindHook(extension.HookGenotypeMarkerAttributes); err != nil {
		return err
	}
	container, err := containerFromSlot(extension.HookGenotypeMarkerAttributes, clone)
	if err != nil {
		return err
	}
	g.attributesSlot = clone
	g.extensions = container
	return nil
}
