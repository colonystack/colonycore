package domain

import (
	"fmt"

	"colonycore/pkg/domain/extension"
)

func (o *Organism) ensureExtensionContainer() *extension.Container {
	if o.extensions != nil {
		return o.extensions
	}
	container := extension.NewContainer()
	if o.Attributes != nil {
		if err := container.Set(extension.HookOrganismAttributes, extension.PluginCore, cloneExtensionMap(o.Attributes)); err != nil {
			panic(fmt.Errorf("domain: organism extension container: %w", err))
		}
	}
	o.extensions = &container
	return o.extensions
}

func (f *Facility) ensureExtensionContainer() *extension.Container {
	if f.extensions != nil {
		return f.extensions
	}
	container := extension.NewContainer()
	if f.EnvironmentBaselines != nil {
		if err := container.Set(extension.HookFacilityEnvironmentBaselines, extension.PluginCore, cloneExtensionMap(f.EnvironmentBaselines)); err != nil {
			panic(fmt.Errorf("domain: facility extension container: %w", err))
		}
	}
	f.extensions = &container
	return f.extensions
}

func (b *BreedingUnit) ensureExtensionContainer() *extension.Container {
	if b.extensions != nil {
		return b.extensions
	}
	container := extension.NewContainer()
	if b.PairingAttributes != nil {
		if err := container.Set(extension.HookBreedingUnitPairingAttributes, extension.PluginCore, cloneExtensionMap(b.PairingAttributes)); err != nil {
			panic(fmt.Errorf("domain: breeding unit extension container: %w", err))
		}
	}
	b.extensions = &container
	return b.extensions
}

func (o *Observation) ensureExtensionContainer() *extension.Container {
	if o.extensions != nil {
		return o.extensions
	}
	container := extension.NewContainer()
	if o.Data != nil {
		if err := container.Set(extension.HookObservationData, extension.PluginCore, cloneExtensionMap(o.Data)); err != nil {
			panic(fmt.Errorf("domain: observation extension container: %w", err))
		}
	}
	o.extensions = &container
	return o.extensions
}

func (s *Sample) ensureExtensionContainer() *extension.Container {
	if s.extensions != nil {
		return s.extensions
	}
	container := extension.NewContainer()
	if s.Attributes != nil {
		if err := container.Set(extension.HookSampleAttributes, extension.PluginCore, cloneExtensionMap(s.Attributes)); err != nil {
			panic(fmt.Errorf("domain: sample extension container: %w", err))
		}
	}
	s.extensions = &container
	return s.extensions
}

func (s *SupplyItem) ensureExtensionContainer() *extension.Container {
	if s.extensions != nil {
		return s.extensions
	}
	container := extension.NewContainer()
	if s.Attributes != nil {
		if err := container.Set(extension.HookSupplyItemAttributes, extension.PluginCore, cloneExtensionMap(s.Attributes)); err != nil {
			panic(fmt.Errorf("domain: supply item extension container: %w", err))
		}
	}
	s.extensions = &container
	return s.extensions
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
		if err := slot.Set(plugin, payload); err != nil {
			panic(fmt.Errorf("domain: hydrate slot from container (hook=%s, plugin=%s): %w", hook, plugin, err))
		}
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
