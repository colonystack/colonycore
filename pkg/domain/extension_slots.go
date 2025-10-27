package domain

import (
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
	o.Attributes = assignExtensionMap(attrs)
	o.attributesSlot = nil
	o.extensions = nil
}

// AttributesMap returns a deep copy of the organism attributes map.
func (o Organism) AttributesMap() map[string]any {
	return cloneExtensionMap(o.Attributes)
}

// SetEnvironmentBaselines clones the provided map into the facility baselines field.
func (f *Facility) SetEnvironmentBaselines(baselines map[string]any) {
	f.EnvironmentBaselines = assignExtensionMap(baselines)
	f.environmentBaselinesSlot = nil
	f.extensions = nil
}

// EnvironmentBaselinesMap returns a deep copy of the facility environment baselines.
func (f Facility) EnvironmentBaselinesMap() map[string]any {
	return cloneExtensionMap(f.EnvironmentBaselines)
}

// SetPairingAttributes clones the provided map into the breeding unit pairing field.
func (b *BreedingUnit) SetPairingAttributes(attrs map[string]any) {
	b.PairingAttributes = assignExtensionMap(attrs)
	b.pairingAttributesSlot = nil
	b.extensions = nil
}

// PairingAttributesMap returns a deep copy of the breeding unit pairing attributes.
func (b BreedingUnit) PairingAttributesMap() map[string]any {
	return cloneExtensionMap(b.PairingAttributes)
}

// SetData clones the provided map into the observation data field.
func (o *Observation) SetData(data map[string]any) {
	o.Data = assignExtensionMap(data)
	o.dataSlot = nil
	o.extensions = nil
}

// DataMap returns a deep copy of the observation data map.
func (o Observation) DataMap() map[string]any {
	return cloneExtensionMap(o.Data)
}

// SetAttributes clones the provided map into the sample attributes field.
func (s *Sample) SetAttributes(attrs map[string]any) {
	s.Attributes = assignExtensionMap(attrs)
	s.attributesSlot = nil
	s.extensions = nil
}

// AttributesMap returns a deep copy of the sample attributes map.
func (s Sample) AttributesMap() map[string]any {
	return cloneExtensionMap(s.Attributes)
}

// SetAttributes clones the provided map into the supply item attributes field.
func (s *SupplyItem) SetAttributes(attrs map[string]any) {
	s.Attributes = assignExtensionMap(attrs)
	s.attributesSlot = nil
	s.extensions = nil
}

// AttributesMap returns a deep copy of the supply item attributes map.
func (s SupplyItem) AttributesMap() map[string]any {
	return cloneExtensionMap(s.Attributes)
}
