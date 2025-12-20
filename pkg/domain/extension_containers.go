package domain

import "colonycore/pkg/domain/extension"

func (o *Organism) ensureExtensionContainer() *extension.Container {
	if o.extensions != nil {
		return o.extensions
	}
	container := extension.NewContainer()
	o.extensions = &container
	return o.extensions
}

func (f *Facility) ensureExtensionContainer() *extension.Container {
	if f.extensions != nil {
		return f.extensions
	}
	container := extension.NewContainer()
	f.extensions = &container
	return f.extensions
}

func (b *BreedingUnit) ensureExtensionContainer() *extension.Container {
	if b.extensions != nil {
		return b.extensions
	}
	container := extension.NewContainer()
	b.extensions = &container
	return b.extensions
}

func (o *Observation) ensureExtensionContainer() *extension.Container {
	if o.extensions != nil {
		return o.extensions
	}
	container := extension.NewContainer()
	o.extensions = &container
	return o.extensions
}

func (s *Sample) ensureExtensionContainer() *extension.Container {
	if s.extensions != nil {
		return s.extensions
	}
	container := extension.NewContainer()
	s.extensions = &container
	return s.extensions
}

func (s *SupplyItem) ensureExtensionContainer() *extension.Container {
	if s.extensions != nil {
		return s.extensions
	}
	container := extension.NewContainer()
	s.extensions = &container
	return s.extensions
}

func (l *Line) ensureExtensionContainer() *extension.Container {
	if l.extensions != nil {
		return l.extensions
	}
	container := extension.NewContainer()
	l.extensions = &container
	return l.extensions
}

func (l *Line) rebindLineSlots() {
	if l.extensions == nil {
		l.defaultAttributesSlot = nil
		l.extensionOverridesSlot = nil
		return
	}

	defaultSlot := slotFromContainer(extension.HookLineDefaultAttributes, l.extensions)
	if len(defaultSlot.Plugins()) == 0 {
		l.defaultAttributesSlot = nil
	} else {
		l.defaultAttributesSlot = defaultSlot
	}

	overrideSlot := slotFromContainer(extension.HookLineExtensionOverrides, l.extensions)
	if len(overrideSlot.Plugins()) == 0 {
		l.extensionOverridesSlot = nil
	} else {
		l.extensionOverridesSlot = overrideSlot
	}
}

func (s *Strain) ensureExtensionContainer() *extension.Container {
	if s.extensions != nil {
		return s.extensions
	}
	container := extension.NewContainer()
	s.extensions = &container
	return s.extensions
}

func (g *GenotypeMarker) ensureExtensionContainer() *extension.Container {
	if g.extensions != nil {
		return g.extensions
	}
	container := extension.NewContainer()
	g.extensions = &container
	return g.extensions
}

func slotFromContainer(hook extension.Hook, container *extension.Container) *extension.Slot {
	slot := extension.NewSlot(hook)
	if container == nil {
		return slot
	}
	for _, plugin := range container.Plugins(hook) {
		payload, ok := container.Get(hook, plugin)
		if !ok {
			continue
		}
		panicOnExtension(slot.Set(plugin, payload), "domain: hydrate slot from container (hook=%s, plugin=%s)", hook, plugin)
	}
	return slot
}

func containerFromSlot(hook extension.Hook, slot *extension.Slot) (*extension.Container, error) {
	if slot == nil {
		return nil, nil
	}
	container := extension.NewContainer()
	for _, plugin := range slot.Plugins() {
		payload, ok := slot.Get(plugin)
		if !ok {
			continue
		}
		if err := container.Set(hook, plugin, payload); err != nil {
			return nil, err
		}
	}
	if len(container.Plugins(hook)) == 0 {
		return nil, nil
	}
	return &container, nil
}
