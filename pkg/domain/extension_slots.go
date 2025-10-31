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
// Deprecated: use SetCoreAttributes instead.
func (o *Organism) SetAttributes(attrs map[string]any) {
	if err := o.SetCoreAttributes(cloneExtensionMap(attrs)); err != nil {
		panic(fmt.Errorf("domain: set organism attributes: %w", err))
	}
	o.attributesSlot = nil
}

// AttributesMap returns a deep copy of the organism attributes map.
// Deprecated: use CoreAttributes instead.
func (o Organism) AttributesMap() map[string]any {
	return (&o).CoreAttributes()
}

// SetEnvironmentBaselines clones the provided map into the facility baselines field.
// Deprecated: use ApplyEnvironmentBaselines instead.
func (f *Facility) SetEnvironmentBaselines(baselines map[string]any) {
	if err := f.ApplyEnvironmentBaselines(cloneExtensionMap(baselines)); err != nil {
		panic(fmt.Errorf("domain: set facility environment baselines: %w", err))
	}
	f.environmentBaselinesSlot = nil
}

// EnvironmentBaselinesMap returns a deep copy of the facility environment baselines.
// Deprecated: use EnvironmentBaselines instead.
func (f Facility) EnvironmentBaselinesMap() map[string]any {
	return (&f).EnvironmentBaselines()
}

// SetPairingAttributes clones the provided map into the breeding unit pairing field.
// Deprecated: use ApplyPairingAttributes instead.
func (b *BreedingUnit) SetPairingAttributes(attrs map[string]any) {
	if err := b.ApplyPairingAttributes(cloneExtensionMap(attrs)); err != nil {
		panic(fmt.Errorf("domain: set breeding unit pairing attributes: %w", err))
	}
	b.pairingAttributesSlot = nil
}

// PairingAttributesMap returns a deep copy of the breeding unit pairing attributes.
// Deprecated: use PairingAttributes instead.
func (b BreedingUnit) PairingAttributesMap() map[string]any {
	return (&b).PairingAttributes()
}

// SetData clones the provided map into the observation data field.
// Deprecated: use ApplyObservationData instead.
func (o *Observation) SetData(data map[string]any) {
	if err := o.ApplyObservationData(cloneExtensionMap(data)); err != nil {
		panic(fmt.Errorf("domain: set observation data: %w", err))
	}
	o.dataSlot = nil
}

// DataMap returns a deep copy of the observation data map.
// Deprecated: use ObservationData instead.
func (o Observation) DataMap() map[string]any {
	return (&o).ObservationData()
}

// SetAttributes clones the provided map into the sample attributes field.
// Deprecated: use ApplySampleAttributes instead.
func (s *Sample) SetAttributes(attrs map[string]any) {
	if err := s.ApplySampleAttributes(cloneExtensionMap(attrs)); err != nil {
		panic(fmt.Errorf("domain: set sample attributes: %w", err))
	}
	s.attributesSlot = nil
}

// AttributesMap returns a deep copy of the sample attributes map.
// Deprecated: use SampleAttributes instead.
func (s Sample) AttributesMap() map[string]any {
	return (&s).SampleAttributes()
}

// SetAttributes clones the provided map into the supply item attributes field.
// Deprecated: use ApplySupplyAttributes instead.
func (s *SupplyItem) SetAttributes(attrs map[string]any) {
	if err := s.ApplySupplyAttributes(cloneExtensionMap(attrs)); err != nil {
		panic(fmt.Errorf("domain: set supply item attributes: %w", err))
	}
	s.attributesSlot = nil
}

// AttributesMap returns a deep copy of the supply item attributes map.
// Deprecated: use SupplyAttributes instead.
func (s SupplyItem) AttributesMap() map[string]any {
	return (&s).SupplyAttributes()
}
