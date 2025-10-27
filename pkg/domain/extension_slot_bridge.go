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
	return slotFromMap(extension.HookOrganismAttributes, o.Attributes)
}

// SetAttributesSlot persists the slot payload back into the legacy attribute
// map. It rejects non-core plugin payloads to avoid silent data loss.
func (o *Organism) SetAttributesSlot(slot *extension.Slot) error {
	attrs, err := mapFromSlot(extension.HookOrganismAttributes, slot)
	if err != nil {
		return err
	}
	o.Attributes = attrs
	return nil
}

// EnsureEnvironmentBaselinesSlot exposes facility environment baselines through
// an extension slot without changing the persisted map representation yet.
func (f *Facility) EnsureEnvironmentBaselinesSlot() *extension.Slot {
	return slotFromMap(extension.HookFacilityEnvironmentBaselines, f.EnvironmentBaselines)
}

// SetEnvironmentBaselinesSlot restores the legacy environment baselines map
// from an extension slot payload.
func (f *Facility) SetEnvironmentBaselinesSlot(slot *extension.Slot) error {
	baselines, err := mapFromSlot(extension.HookFacilityEnvironmentBaselines, slot)
	if err != nil {
		return err
	}
	f.EnvironmentBaselines = baselines
	return nil
}

// EnsurePairingAttributesSlot exposes breeding unit pairing attributes through
// the extension slot bridge.
func (b *BreedingUnit) EnsurePairingAttributesSlot() *extension.Slot {
	return slotFromMap(extension.HookBreedingUnitPairingAttributes, b.PairingAttributes)
}

// SetPairingAttributesSlot restores the legacy pairing attributes map from a
// slot payload.
func (b *BreedingUnit) SetPairingAttributesSlot(slot *extension.Slot) error {
	attrs, err := mapFromSlot(extension.HookBreedingUnitPairingAttributes, slot)
	if err != nil {
		return err
	}
	b.PairingAttributes = attrs
	return nil
}

// EnsureObservationDataSlot exposes observation data through the slot bridge.
func (o *Observation) EnsureObservationDataSlot() *extension.Slot {
	return slotFromMap(extension.HookObservationData, o.Data)
}

// SetObservationDataSlot restores the observation data map from a slot payload.
func (o *Observation) SetObservationDataSlot(slot *extension.Slot) error {
	data, err := mapFromSlot(extension.HookObservationData, slot)
	if err != nil {
		return err
	}
	o.Data = data
	return nil
}

// EnsureSampleAttributesSlot exposes sample attributes through the slot bridge.
func (s *Sample) EnsureSampleAttributesSlot() *extension.Slot {
	return slotFromMap(extension.HookSampleAttributes, s.Attributes)
}

// SetSampleAttributesSlot restores the sample attributes map from a slot payload.
func (s *Sample) SetSampleAttributesSlot(slot *extension.Slot) error {
	attrs, err := mapFromSlot(extension.HookSampleAttributes, slot)
	if err != nil {
		return err
	}
	s.Attributes = attrs
	return nil
}

// EnsureSupplyItemAttributesSlot exposes supply item attributes through the slot bridge.
func (s *SupplyItem) EnsureSupplyItemAttributesSlot() *extension.Slot {
	return slotFromMap(extension.HookSupplyItemAttributes, s.Attributes)
}

// SetSupplyItemAttributesSlot restores the supply item attributes map from a slot payload.
func (s *SupplyItem) SetSupplyItemAttributesSlot(slot *extension.Slot) error {
	attrs, err := mapFromSlot(extension.HookSupplyItemAttributes, slot)
	if err != nil {
		return err
	}
	s.Attributes = attrs
	return nil
}
