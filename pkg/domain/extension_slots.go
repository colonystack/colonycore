package domain

import (
	"fmt"

	"colonycore/pkg/domain/extension"
)

// EnsureDefaultAttributes returns the line default attributes slot, initialising
// it with the correct hook identifier when needed.
func (l *Line) EnsureDefaultAttributes() *extension.Slot {
	if l.DefaultAttributes == nil {
		l.DefaultAttributes = extension.NewSlot(extension.HookLineDefaultAttributes)
		return l.DefaultAttributes
	}
	_ = l.DefaultAttributes.BindHook(extension.HookLineDefaultAttributes)
	return l.DefaultAttributes
}

// EnsureExtensionOverrides returns the line extension overrides slot, ensuring
// it is bound to the correct hook.
func (l *Line) EnsureExtensionOverrides() *extension.Slot {
	if l.ExtensionOverrides == nil {
		l.ExtensionOverrides = extension.NewSlot(extension.HookLineExtensionOverrides)
		return l.ExtensionOverrides
	}
	_ = l.ExtensionOverrides.BindHook(extension.HookLineExtensionOverrides)
	return l.ExtensionOverrides
}

// EnsureAttributes returns the strain attributes slot, initialising it if necessary.
func (s *Strain) EnsureAttributes() *extension.Slot {
	if s.Attributes == nil {
		s.Attributes = extension.NewSlot(extension.HookStrainAttributes)
		return s.Attributes
	}
	_ = s.Attributes.BindHook(extension.HookStrainAttributes)
	return s.Attributes
}

// EnsureAttributes returns the genotype marker attributes slot, initialising it if necessary.
func (g *GenotypeMarker) EnsureAttributes() *extension.Slot {
	if g.Attributes == nil {
		g.Attributes = extension.NewSlot(extension.HookGenotypeMarkerAttributes)
		return g.Attributes
	}
	_ = g.Attributes.BindHook(extension.HookGenotypeMarkerAttributes)
	return g.Attributes
}

// SetAttributes clones the provided map into the organism attributes field.
func (o *Organism) SetAttributes(attrs map[string]any) {
	if attrs == nil {
		o.attributesSlot = nil
		o.extensions = nil
		return
	}
	slot := extension.NewSlot(extension.HookOrganismAttributes)
	if err := slot.Set(extension.PluginCore, cloneExtensionMap(attrs)); err != nil {
		panic(fmt.Errorf("domain: set organism attributes: %w", err))
	}
	if err := o.SetAttributesSlot(slot); err != nil {
		panic(fmt.Errorf("domain: persist organism attributes slot: %w", err))
	}
	o.attributesSlot = nil
}

// AttributesMap returns a deep copy of the organism attributes map.
func (o Organism) AttributesMap() map[string]any {
	slot := o.attributesSlot
	if slot == nil && o.extensions != nil {
		slot = slotFromContainer(extension.HookOrganismAttributes, o.extensions)
	}
	if slot == nil {
		return nil
	}
	payload, ok := slot.Get(extension.PluginCore)
	if !ok || payload == nil {
		return nil
	}
	values, ok := payload.(map[string]any)
	if !ok {
		return nil
	}
	return cloneExtensionMap(values)
}

// SetEnvironmentBaselines clones the provided map into the facility baselines field.
func (f *Facility) SetEnvironmentBaselines(baselines map[string]any) {
	if baselines == nil {
		f.environmentBaselinesSlot = nil
		f.extensions = nil
		return
	}
	slot := extension.NewSlot(extension.HookFacilityEnvironmentBaselines)
	if err := slot.Set(extension.PluginCore, cloneExtensionMap(baselines)); err != nil {
		panic(fmt.Errorf("domain: set facility environment baselines: %w", err))
	}
	if err := f.SetEnvironmentBaselinesSlot(slot); err != nil {
		panic(fmt.Errorf("domain: persist facility environment baselines slot: %w", err))
	}
	f.environmentBaselinesSlot = nil
}

// EnvironmentBaselinesMap returns a deep copy of the facility environment baselines.
func (f Facility) EnvironmentBaselinesMap() map[string]any {
	slot := f.environmentBaselinesSlot
	if slot == nil && f.extensions != nil {
		slot = slotFromContainer(extension.HookFacilityEnvironmentBaselines, f.extensions)
	}
	if slot == nil {
		return nil
	}
	payload, ok := slot.Get(extension.PluginCore)
	if !ok || payload == nil {
		return nil
	}
	values, ok := payload.(map[string]any)
	if !ok {
		return nil
	}
	return cloneExtensionMap(values)
}

// SetPairingAttributes clones the provided map into the breeding unit pairing field.
func (b *BreedingUnit) SetPairingAttributes(attrs map[string]any) {
	if attrs == nil {
		b.pairingAttributesSlot = nil
		b.extensions = nil
		return
	}
	slot := extension.NewSlot(extension.HookBreedingUnitPairingAttributes)
	if err := slot.Set(extension.PluginCore, cloneExtensionMap(attrs)); err != nil {
		panic(fmt.Errorf("domain: set breeding unit pairing attributes: %w", err))
	}
	if err := b.SetPairingAttributesSlot(slot); err != nil {
		panic(fmt.Errorf("domain: persist breeding unit pairing attributes slot: %w", err))
	}
	b.pairingAttributesSlot = nil
}

// PairingAttributesMap returns a deep copy of the breeding unit pairing attributes.
func (b BreedingUnit) PairingAttributesMap() map[string]any {
	slot := b.pairingAttributesSlot
	if slot == nil && b.extensions != nil {
		slot = slotFromContainer(extension.HookBreedingUnitPairingAttributes, b.extensions)
	}
	if slot == nil {
		return nil
	}
	payload, ok := slot.Get(extension.PluginCore)
	if !ok || payload == nil {
		return nil
	}
	values, ok := payload.(map[string]any)
	if !ok {
		return nil
	}
	return cloneExtensionMap(values)
}

// SetData clones the provided map into the observation data field.
func (o *Observation) SetData(data map[string]any) {
	if data == nil {
		o.dataSlot = nil
		o.extensions = nil
		return
	}
	slot := extension.NewSlot(extension.HookObservationData)
	if err := slot.Set(extension.PluginCore, cloneExtensionMap(data)); err != nil {
		panic(fmt.Errorf("domain: set observation data: %w", err))
	}
	if err := o.SetObservationDataSlot(slot); err != nil {
		panic(fmt.Errorf("domain: persist observation data slot: %w", err))
	}
	o.dataSlot = nil
}

// DataMap returns a deep copy of the observation data map.
func (o Observation) DataMap() map[string]any {
	slot := o.dataSlot
	if slot == nil && o.extensions != nil {
		slot = slotFromContainer(extension.HookObservationData, o.extensions)
	}
	if slot == nil {
		return nil
	}
	payload, ok := slot.Get(extension.PluginCore)
	if !ok || payload == nil {
		return nil
	}
	values, ok := payload.(map[string]any)
	if !ok {
		return nil
	}
	return cloneExtensionMap(values)
}

// SetAttributes clones the provided map into the sample attributes field.
func (s *Sample) SetAttributes(attrs map[string]any) {
	if attrs == nil {
		s.attributesSlot = nil
		s.extensions = nil
		return
	}
	slot := extension.NewSlot(extension.HookSampleAttributes)
	if err := slot.Set(extension.PluginCore, cloneExtensionMap(attrs)); err != nil {
		panic(fmt.Errorf("domain: set sample attributes: %w", err))
	}
	if err := s.SetSampleAttributesSlot(slot); err != nil {
		panic(fmt.Errorf("domain: persist sample attributes slot: %w", err))
	}
	s.attributesSlot = nil
}

// AttributesMap returns a deep copy of the sample attributes map.
func (s Sample) AttributesMap() map[string]any {
	slot := s.attributesSlot
	if slot == nil && s.extensions != nil {
		slot = slotFromContainer(extension.HookSampleAttributes, s.extensions)
	}
	if slot == nil {
		return nil
	}
	payload, ok := slot.Get(extension.PluginCore)
	if !ok || payload == nil {
		return nil
	}
	values, ok := payload.(map[string]any)
	if !ok {
		return nil
	}
	return cloneExtensionMap(values)
}

// SetAttributes clones the provided map into the supply item attributes field.
func (s *SupplyItem) SetAttributes(attrs map[string]any) {
	if attrs == nil {
		s.attributesSlot = nil
		s.extensions = nil
		return
	}
	slot := extension.NewSlot(extension.HookSupplyItemAttributes)
	if err := slot.Set(extension.PluginCore, cloneExtensionMap(attrs)); err != nil {
		panic(fmt.Errorf("domain: set supply item attributes: %w", err))
	}
	if err := s.SetSupplyItemAttributesSlot(slot); err != nil {
		panic(fmt.Errorf("domain: persist supply item attributes slot: %w", err))
	}
	s.attributesSlot = nil
}

// AttributesMap returns a deep copy of the supply item attributes map.
func (s SupplyItem) AttributesMap() map[string]any {
	slot := s.attributesSlot
	if slot == nil && s.extensions != nil {
		slot = slotFromContainer(extension.HookSupplyItemAttributes, s.extensions)
	}
	if slot == nil {
		return nil
	}
	payload, ok := slot.Get(extension.PluginCore)
	if !ok || payload == nil {
		return nil
	}
	values, ok := payload.(map[string]any)
	if !ok {
		return nil
	}
	return cloneExtensionMap(values)
}
