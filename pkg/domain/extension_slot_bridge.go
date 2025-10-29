package domain

import "colonycore/pkg/domain/extension"

// EnsureAttributesSlot exposes the organism attribute bag through an
// extension.Slot while keeping the legacy map field authoritative.
func (o *Organism) EnsureAttributesSlot() *extension.Slot {
	if o.attributesSlot == nil {
		o.attributesSlot = slotFromContainer(extension.HookOrganismAttributes, o.ensureExtensionContainer())
	} else {
		_ = o.attributesSlot.BindHook(extension.HookOrganismAttributes)
	}
	return o.attributesSlot
}

// SetAttributesSlot persists the slot payload back into the legacy attribute
// map. It rejects non-core plugin payloads to avoid silent data loss.
func (o *Organism) SetAttributesSlot(slot *extension.Slot) error {
	if slot == nil {
		o.attributesSlot = nil
		o.extensions = nil
		return nil
	}
	clone := slot.Clone()
	if err := clone.BindHook(extension.HookOrganismAttributes); err != nil {
		return err
	}
	container, err := containerFromSlot(extension.HookOrganismAttributes, clone)
	if err != nil {
		return err
	}
	o.attributesSlot = clone
	o.extensions = container
	return nil
}

// EnsureEnvironmentBaselinesSlot exposes facility environment baselines through
// an extension slot without changing the persisted map representation yet.
func (f *Facility) EnsureEnvironmentBaselinesSlot() *extension.Slot {
	if f.environmentBaselinesSlot == nil {
		f.environmentBaselinesSlot = slotFromContainer(extension.HookFacilityEnvironmentBaselines, f.ensureExtensionContainer())
	} else {
		_ = f.environmentBaselinesSlot.BindHook(extension.HookFacilityEnvironmentBaselines)
	}
	return f.environmentBaselinesSlot
}

// SetEnvironmentBaselinesSlot restores the legacy environment baselines map
// from an extension slot payload.
func (f *Facility) SetEnvironmentBaselinesSlot(slot *extension.Slot) error {
	if slot == nil {
		f.environmentBaselinesSlot = nil
		f.extensions = nil
		return nil
	}
	clone := slot.Clone()
	if err := clone.BindHook(extension.HookFacilityEnvironmentBaselines); err != nil {
		return err
	}
	container, err := containerFromSlot(extension.HookFacilityEnvironmentBaselines, clone)
	if err != nil {
		return err
	}
	f.environmentBaselinesSlot = clone
	f.extensions = container
	return nil
}

// EnsurePairingAttributesSlot exposes breeding unit pairing attributes through
// the extension slot bridge.
func (b *BreedingUnit) EnsurePairingAttributesSlot() *extension.Slot {
	if b.pairingAttributesSlot == nil {
		b.pairingAttributesSlot = slotFromContainer(extension.HookBreedingUnitPairingAttributes, b.ensureExtensionContainer())
	} else {
		_ = b.pairingAttributesSlot.BindHook(extension.HookBreedingUnitPairingAttributes)
	}
	return b.pairingAttributesSlot
}

// SetPairingAttributesSlot restores the legacy pairing attributes map from a
// slot payload.
func (b *BreedingUnit) SetPairingAttributesSlot(slot *extension.Slot) error {
	if slot == nil {
		b.pairingAttributesSlot = nil
		b.extensions = nil
		return nil
	}
	clone := slot.Clone()
	if err := clone.BindHook(extension.HookBreedingUnitPairingAttributes); err != nil {
		return err
	}
	container, err := containerFromSlot(extension.HookBreedingUnitPairingAttributes, clone)
	if err != nil {
		return err
	}
	b.pairingAttributesSlot = clone
	b.extensions = container
	return nil
}

// EnsureObservationDataSlot exposes observation data through the slot bridge.
func (o *Observation) EnsureObservationDataSlot() *extension.Slot {
	if o.dataSlot == nil {
		o.dataSlot = slotFromContainer(extension.HookObservationData, o.ensureExtensionContainer())
	} else {
		_ = o.dataSlot.BindHook(extension.HookObservationData)
	}
	return o.dataSlot
}

// SetObservationDataSlot restores the observation data map from a slot payload.
func (o *Observation) SetObservationDataSlot(slot *extension.Slot) error {
	if slot == nil {
		o.dataSlot = nil
		o.extensions = nil
		return nil
	}
	clone := slot.Clone()
	if err := clone.BindHook(extension.HookObservationData); err != nil {
		return err
	}
	container, err := containerFromSlot(extension.HookObservationData, clone)
	if err != nil {
		return err
	}
	o.dataSlot = clone
	o.extensions = container
	return nil
}

// EnsureSampleAttributesSlot exposes sample attributes through the slot bridge.
func (s *Sample) EnsureSampleAttributesSlot() *extension.Slot {
	if s.attributesSlot == nil {
		s.attributesSlot = slotFromContainer(extension.HookSampleAttributes, s.ensureExtensionContainer())
	} else {
		_ = s.attributesSlot.BindHook(extension.HookSampleAttributes)
	}
	return s.attributesSlot
}

// SetSampleAttributesSlot restores the sample attributes map from a slot payload.
func (s *Sample) SetSampleAttributesSlot(slot *extension.Slot) error {
	if slot == nil {
		s.attributesSlot = nil
		s.extensions = nil
		return nil
	}
	clone := slot.Clone()
	if err := clone.BindHook(extension.HookSampleAttributes); err != nil {
		return err
	}
	container, err := containerFromSlot(extension.HookSampleAttributes, clone)
	if err != nil {
		return err
	}
	s.attributesSlot = clone
	s.extensions = container
	return nil
}

// EnsureSupplyItemAttributesSlot exposes supply item attributes through the slot bridge.
func (s *SupplyItem) EnsureSupplyItemAttributesSlot() *extension.Slot {
	if s.attributesSlot == nil {
		s.attributesSlot = slotFromContainer(extension.HookSupplyItemAttributes, s.ensureExtensionContainer())
	} else {
		_ = s.attributesSlot.BindHook(extension.HookSupplyItemAttributes)
	}
	return s.attributesSlot
}

// SetSupplyItemAttributesSlot restores the supply item attributes map from a slot payload.
func (s *SupplyItem) SetSupplyItemAttributesSlot(slot *extension.Slot) error {
	if slot == nil {
		s.attributesSlot = nil
		s.extensions = nil
		return nil
	}
	clone := slot.Clone()
	if err := clone.BindHook(extension.HookSupplyItemAttributes); err != nil {
		return err
	}
	container, err := containerFromSlot(extension.HookSupplyItemAttributes, clone)
	if err != nil {
		return err
	}
	s.attributesSlot = clone
	s.extensions = container
	return nil
}
