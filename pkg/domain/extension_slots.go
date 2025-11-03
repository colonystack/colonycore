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

// SetAttributes clones the provided map into the organism attributes field.
// Deprecated: use SetCoreAttributes instead.
func (o *Organism) SetAttributes(attrs map[string]any) {
	panicOnExtension(o.SetCoreAttributes(cloneExtensionMap(attrs)), "domain: set organism attributes")
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
	panicOnExtension(f.ApplyEnvironmentBaselines(cloneExtensionMap(baselines)), "domain: set facility environment baselines")
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
	panicOnExtension(b.ApplyPairingAttributes(cloneExtensionMap(attrs)), "domain: set breeding unit pairing attributes")
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
	panicOnExtension(o.ApplyObservationData(cloneExtensionMap(data)), "domain: set observation data")
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
	panicOnExtension(s.ApplySampleAttributes(cloneExtensionMap(attrs)), "domain: set sample attributes")
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
	panicOnExtension(s.ApplySupplyAttributes(cloneExtensionMap(attrs)), "domain: set supply item attributes")
	s.attributesSlot = nil
}

// AttributesMap returns a deep copy of the supply item attributes map.
// Deprecated: use SupplyAttributes instead.
func (s SupplyItem) AttributesMap() map[string]any {
	return (&s).SupplyAttributes()
}
