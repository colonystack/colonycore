package domain

import (
	"fmt"

	"colonycore/pkg/domain/extension"
)

// slotFromMap initialises an extension slot bound to the provided hook and
// seeds it with the flattened legacy attribute map as the core payload.
func slotFromMap(hook extension.Hook, values map[string]any) *extension.Slot {
	slot := extension.NewSlot(hook)
	if values == nil {
		return slot
	}
	cloned := assignExtensionMap(values)
	// store the legacy payload under the synthetic "core" plugin identifier so
	// future phases can promote real plugin payloads without losing data.
	_ = slot.Set(extension.PluginCore, cloned)
	return slot
}

// mapFromSlot extracts the legacy attribute map from an extension slot. The
// bridge expects the payload to live under the synthetic "core" plugin; any
// additional plugin entries are rejected so we do not silently drop data while
// the legacy map representation remains in use.
func mapFromSlot(hook extension.Hook, slot *extension.Slot) (map[string]any, error) {
	if slot == nil {
		return nil, nil
	}
	if err := slot.BindHook(hook); err != nil {
		return nil, err
	}
	for _, plugin := range slot.Plugins() {
		if plugin != extension.PluginCore {
			return nil, fmt.Errorf("domain: hook %s: unsupported plugin payload %q while legacy maps are active", hook, plugin)
		}
	}
	payload, ok := slot.Get(extension.PluginCore)
	if !ok || payload == nil {
		return nil, nil
	}
	result, ok := payload.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("domain: hook %s: core payload must be JSON object, got %T", hook, payload)
	}
	return assignExtensionMap(result), nil
}

// EnsureAttributesSlot exposes the organism attribute bag through an
// extension.Slot while keeping the legacy map field authoritative.
func (o *Organism) EnsureAttributesSlot() *extension.Slot {
	if o.attributesSlot == nil {
		o.attributesSlot = slotFromMap(extension.HookOrganismAttributes, o.Attributes)
		return o.attributesSlot
	}
	if err := o.attributesSlot.BindHook(extension.HookOrganismAttributes); err != nil {
		o.attributesSlot = slotFromMap(extension.HookOrganismAttributes, o.Attributes)
		return o.attributesSlot
	}
	return o.attributesSlot
}

// SetAttributesSlot persists the slot payload back into the legacy attribute
// map. It rejects non-core plugin payloads to avoid silent data loss.
func (o *Organism) SetAttributesSlot(slot *extension.Slot) error {
	if slot == nil {
		o.attributesSlot = nil
		o.Attributes = nil
		return nil
	}
	clone := slot.Clone()
	attrs, err := mapFromSlot(extension.HookOrganismAttributes, clone)
	if err != nil {
		return err
	}
	o.attributesSlot = clone
	o.Attributes = attrs
	return nil
}

// EnsureEnvironmentBaselinesSlot exposes facility environment baselines through
// an extension slot without changing the persisted map representation yet.
func (f *Facility) EnsureEnvironmentBaselinesSlot() *extension.Slot {
	if f.environmentBaselinesSlot == nil {
		f.environmentBaselinesSlot = slotFromMap(extension.HookFacilityEnvironmentBaselines, f.EnvironmentBaselines)
		return f.environmentBaselinesSlot
	}
	if err := f.environmentBaselinesSlot.BindHook(extension.HookFacilityEnvironmentBaselines); err != nil {
		f.environmentBaselinesSlot = slotFromMap(extension.HookFacilityEnvironmentBaselines, f.EnvironmentBaselines)
		return f.environmentBaselinesSlot
	}
	return f.environmentBaselinesSlot
}

// SetEnvironmentBaselinesSlot restores the legacy environment baselines map
// from an extension slot payload.
func (f *Facility) SetEnvironmentBaselinesSlot(slot *extension.Slot) error {
	if slot == nil {
		f.environmentBaselinesSlot = nil
		f.EnvironmentBaselines = nil
		return nil
	}
	clone := slot.Clone()
	baselines, err := mapFromSlot(extension.HookFacilityEnvironmentBaselines, clone)
	if err != nil {
		return err
	}
	f.environmentBaselinesSlot = clone
	f.EnvironmentBaselines = baselines
	return nil
}

// EnsurePairingAttributesSlot exposes breeding unit pairing attributes through
// the extension slot bridge.
func (b *BreedingUnit) EnsurePairingAttributesSlot() *extension.Slot {
	if b.pairingAttributesSlot == nil {
		b.pairingAttributesSlot = slotFromMap(extension.HookBreedingUnitPairingAttributes, b.PairingAttributes)
		return b.pairingAttributesSlot
	}
	if err := b.pairingAttributesSlot.BindHook(extension.HookBreedingUnitPairingAttributes); err != nil {
		b.pairingAttributesSlot = slotFromMap(extension.HookBreedingUnitPairingAttributes, b.PairingAttributes)
		return b.pairingAttributesSlot
	}
	return b.pairingAttributesSlot
}

// SetPairingAttributesSlot restores the legacy pairing attributes map from a
// slot payload.
func (b *BreedingUnit) SetPairingAttributesSlot(slot *extension.Slot) error {
	if slot == nil {
		b.pairingAttributesSlot = nil
		b.PairingAttributes = nil
		return nil
	}
	clone := slot.Clone()
	attrs, err := mapFromSlot(extension.HookBreedingUnitPairingAttributes, clone)
	if err != nil {
		return err
	}
	b.pairingAttributesSlot = clone
	b.PairingAttributes = attrs
	return nil
}

// EnsureObservationDataSlot exposes observation data through the slot bridge.
func (o *Observation) EnsureObservationDataSlot() *extension.Slot {
	if o.dataSlot == nil {
		o.dataSlot = slotFromMap(extension.HookObservationData, o.Data)
		return o.dataSlot
	}
	if err := o.dataSlot.BindHook(extension.HookObservationData); err != nil {
		o.dataSlot = slotFromMap(extension.HookObservationData, o.Data)
		return o.dataSlot
	}
	return o.dataSlot
}

// SetObservationDataSlot restores the observation data map from a slot payload.
func (o *Observation) SetObservationDataSlot(slot *extension.Slot) error {
	if slot == nil {
		o.dataSlot = nil
		o.Data = nil
		return nil
	}
	clone := slot.Clone()
	data, err := mapFromSlot(extension.HookObservationData, clone)
	if err != nil {
		return err
	}
	o.dataSlot = clone
	o.Data = data
	return nil
}

// EnsureSampleAttributesSlot exposes sample attributes through the slot bridge.
func (s *Sample) EnsureSampleAttributesSlot() *extension.Slot {
	if s.attributesSlot == nil {
		s.attributesSlot = slotFromMap(extension.HookSampleAttributes, s.Attributes)
		return s.attributesSlot
	}
	if err := s.attributesSlot.BindHook(extension.HookSampleAttributes); err != nil {
		s.attributesSlot = slotFromMap(extension.HookSampleAttributes, s.Attributes)
		return s.attributesSlot
	}
	return s.attributesSlot
}

// SetSampleAttributesSlot restores the sample attributes map from a slot payload.
func (s *Sample) SetSampleAttributesSlot(slot *extension.Slot) error {
	if slot == nil {
		s.attributesSlot = nil
		s.Attributes = nil
		return nil
	}
	clone := slot.Clone()
	attrs, err := mapFromSlot(extension.HookSampleAttributes, clone)
	if err != nil {
		return err
	}
	s.attributesSlot = clone
	s.Attributes = attrs
	return nil
}

// EnsureSupplyItemAttributesSlot exposes supply item attributes through the slot bridge.
func (s *SupplyItem) EnsureSupplyItemAttributesSlot() *extension.Slot {
	if s.attributesSlot == nil {
		s.attributesSlot = slotFromMap(extension.HookSupplyItemAttributes, s.Attributes)
		return s.attributesSlot
	}
	if err := s.attributesSlot.BindHook(extension.HookSupplyItemAttributes); err != nil {
		s.attributesSlot = slotFromMap(extension.HookSupplyItemAttributes, s.Attributes)
		return s.attributesSlot
	}
	return s.attributesSlot
}

// SetSupplyItemAttributesSlot restores the supply item attributes map from a slot payload.
func (s *SupplyItem) SetSupplyItemAttributesSlot(slot *extension.Slot) error {
	if slot == nil {
		s.attributesSlot = nil
		s.Attributes = nil
		return nil
	}
	clone := slot.Clone()
	attrs, err := mapFromSlot(extension.HookSupplyItemAttributes, clone)
	if err != nil {
		return err
	}
	s.attributesSlot = clone
	s.Attributes = attrs
	return nil
}
