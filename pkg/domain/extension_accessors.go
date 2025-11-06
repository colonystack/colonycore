package domain

import (
	"fmt"

	"colonycore/pkg/domain/extension"
)

// cloneHookMap retrieves a defensive copy of the payload stored for the given
// hook and plugin combination. The second return value reports whether a
// payload was present.
func cloneHookMap(container *extension.Container, hook extension.Hook, plugin extension.PluginID) (map[string]any, bool) {
	if container == nil {
		return nil, false
	}
	payload, ok := container.Get(hook, plugin)
	if !ok || payload == nil {
		return nil, false
	}
	values, ok := payload.(map[string]any)
	if !ok {
		return nil, false
	}
	return extension.CloneMap(values), true
}

// updateHookPayload mutates the underlying container and slot references for a
// single hook/plugin combination. When payload is nil the entry is removed.
func updateHookPayload(
	ensure func() *extension.Container,
	containerRef **extension.Container,
	slotRef **extension.Slot,
	hook extension.Hook,
	plugin extension.PluginID,
	payload map[string]any,
) error {
	if payload == nil {
		if *containerRef == nil {
			*slotRef = nil
			return nil
		}
		(*containerRef).Remove(hook, plugin)
		if len((*containerRef).Hooks()) == 0 {
			*containerRef = nil
			*slotRef = nil
			return nil
		}
		clone, err := cloneContainer(*containerRef)
		if err != nil {
			return err
		}
		*containerRef = &clone
		slot := slotFromContainer(hook, *containerRef)
		if len(slot.Plugins()) == 0 {
			*slotRef = nil
		} else {
			*slotRef = slot
		}
		return nil
	}

	container := ensure()
	if err := container.Set(hook, plugin, payload); err != nil {
		return err
	}
	clone, err := cloneContainer(container)
	if err != nil {
		return err
	}
	*containerRef = &clone
	slot := slotFromContainer(hook, *containerRef)
	if len(slot.Plugins()) == 0 {
		*slotRef = nil
	} else {
		*slotRef = slot
	}
	return nil
}

// replaceExtensionContainer installs a cloned copy of the provided container
// onto the target entity and synchronises the backing slot representation. A
// container with no hooks clears both references.
func replaceExtensionContainer(
	targetRef **extension.Container,
	slotRef **extension.Slot,
	hook extension.Hook,
	container extension.Container,
) error {
	clone, err := cloneContainer(&container)
	if err != nil {
		return err
	}
	if len(clone.Hooks()) == 0 {
		*targetRef = nil
		*slotRef = nil
		return nil
	}
	*targetRef = &clone
	slot := slotFromContainer(hook, *targetRef)
	if len(slot.Plugins()) == 0 {
		*slotRef = nil
	} else {
		*slotRef = slot
	}
	return nil
}

func cloneContainer(container *extension.Container) (extension.Container, error) {
	if container == nil {
		return extension.NewContainer(), nil
	}
	raw := container.Raw()
	return extension.FromRaw(raw)
}

// OrganismExtensions returns a deep copy of the organism extension container.
func (o *Organism) OrganismExtensions() (extension.Container, error) {
	container := o.ensureExtensionContainer()
	return cloneContainer(container)
}

// SetOrganismExtensions replaces the organism extension container with the
// provided payload.
func (o *Organism) SetOrganismExtensions(container extension.Container) error {
	return replaceExtensionContainer(&o.extensions, &o.attributesSlot, extension.HookOrganismAttributes, container)
}

// CoreAttributes returns the payload stored in the core plugin slot for an
// organism. The returned map is a defensive copy and may be nil when unset.
func (o *Organism) CoreAttributes() map[string]any {
	return o.CoreAttributesPayload().Map()
}

// CoreAttributesPayload returns the organism core attributes payload as a typed
// wrapper.
func (o *Organism) CoreAttributesPayload() extension.ObjectPayload {
	payload, err := extension.ObjectFromContainer(
		o.ensureExtensionContainer(),
		extension.HookOrganismAttributes,
		extension.PluginCore,
	)
	panicOnExtension(err, "domain: load organism core attributes")
	return payload
}

// SetCoreAttributes stores a payload for the organism core attributes slot. A
// nil payload removes the entry.
func (o *Organism) SetCoreAttributes(attrs map[string]any) error {
	payload, err := extension.NewObjectPayload(extension.HookOrganismAttributes, attrs)
	if err != nil {
		return err
	}
	return o.SetCoreAttributesPayload(payload)
}

// SetCoreAttributesPayload stores a typed payload for the organism core hook.
func (o *Organism) SetCoreAttributesPayload(payload extension.ObjectPayload) error {
	if err := payload.ExpectHook(extension.HookOrganismAttributes); err != nil {
		return err
	}
	return updateHookPayload(
		func() *extension.Container { return o.ensureExtensionContainer() },
		&o.extensions,
		&o.attributesSlot,
		extension.HookOrganismAttributes,
		extension.PluginCore,
		payload.Map(),
	)
}

// FacilityExtensions returns a deep copy of the facility extension container.
func (f *Facility) FacilityExtensions() (extension.Container, error) {
	container := f.ensureExtensionContainer()
	return cloneContainer(container)
}

// SetFacilityExtensions replaces the facility extension container with the
// provided payload.
func (f *Facility) SetFacilityExtensions(container extension.Container) error {
	return replaceExtensionContainer(&f.extensions, &f.environmentBaselinesSlot, extension.HookFacilityEnvironmentBaselines, container)
}

// EnvironmentBaselines returns a copy of the core environment baselines payload.
func (f *Facility) EnvironmentBaselines() map[string]any {
	return f.EnvironmentBaselinesPayload().Map()
}

// EnvironmentBaselinesPayload returns the environment baselines payload as a
// typed wrapper.
func (f *Facility) EnvironmentBaselinesPayload() extension.ObjectPayload {
	payload, err := extension.ObjectFromContainer(
		f.ensureExtensionContainer(),
		extension.HookFacilityEnvironmentBaselines,
		extension.PluginCore,
	)
	panicOnExtension(err, "domain: load facility environment baselines")
	return payload
}

// ApplyEnvironmentBaselines stores environment baselines for the facility. A
// nil payload clears the entry.
func (f *Facility) ApplyEnvironmentBaselines(baselines map[string]any) error {
	payload, err := extension.NewObjectPayload(extension.HookFacilityEnvironmentBaselines, baselines)
	if err != nil {
		return err
	}
	return f.ApplyEnvironmentBaselinesPayload(payload)
}

// ApplyEnvironmentBaselinesPayload persists a typed payload for the facility hook.
func (f *Facility) ApplyEnvironmentBaselinesPayload(payload extension.ObjectPayload) error {
	if err := payload.ExpectHook(extension.HookFacilityEnvironmentBaselines); err != nil {
		return err
	}
	return updateHookPayload(
		func() *extension.Container { return f.ensureExtensionContainer() },
		&f.extensions,
		&f.environmentBaselinesSlot,
		extension.HookFacilityEnvironmentBaselines,
		extension.PluginCore,
		payload.Map(),
	)
}

// BreedingUnitExtensions returns a deep copy of the breeding unit extension container.
func (b *BreedingUnit) BreedingUnitExtensions() (extension.Container, error) {
	container := b.ensureExtensionContainer()
	return cloneContainer(container)
}

// SetBreedingUnitExtensions replaces the breeding unit extension container with
// the provided payload.
func (b *BreedingUnit) SetBreedingUnitExtensions(container extension.Container) error {
	return replaceExtensionContainer(&b.extensions, &b.pairingAttributesSlot, extension.HookBreedingUnitPairingAttributes, container)
}

// PairingAttributes returns a copy of the core pairing attributes payload.
func (b *BreedingUnit) PairingAttributes() map[string]any {
	return b.PairingAttributesPayload().Map()
}

// PairingAttributesPayload returns the pairing attributes payload as a typed wrapper.
func (b *BreedingUnit) PairingAttributesPayload() extension.ObjectPayload {
	payload, err := extension.ObjectFromContainer(
		b.ensureExtensionContainer(),
		extension.HookBreedingUnitPairingAttributes,
		extension.PluginCore,
	)
	panicOnExtension(err, "domain: load breeding unit pairing attributes")
	return payload
}

// LineExtensions returns a deep copy of the line extension container.
func (l *Line) LineExtensions() (extension.Container, error) {
	container := l.ensureExtensionContainer()
	return cloneContainer(container)
}

// SetLineExtensions replaces all line extension hooks and rebinds the slots.
func (l *Line) SetLineExtensions(container extension.Container) error {
	clone, err := cloneContainer(&container)
	if err != nil {
		return err
	}
	for _, hook := range clone.Hooks() {
		if hook != extension.HookLineDefaultAttributes && hook != extension.HookLineExtensionOverrides {
			return fmt.Errorf("domain: unsupported hook %q for line extensions", hook)
		}
	}
	if len(clone.Hooks()) == 0 {
		l.extensions = nil
		l.defaultAttributesSlot = nil
		l.extensionOverridesSlot = nil
		return nil
	}
	l.extensions = &clone
	l.rebindLineSlots()
	return nil
}

// DefaultAttributes returns a defensive copy of the plugin-scoped default attributes payload.
func (l *Line) DefaultAttributes() map[string]any {
	slot := l.EnsureDefaultAttributes()
	payload := slot.Raw()
	if len(payload) == 0 {
		return nil
	}
	return payload
}

// ApplyDefaultAttributes replaces the default attribute payloads for all plugins.
func (l *Line) ApplyDefaultAttributes(attrs map[string]any) error {
	slot, err := slotFromPluginPayloads(extension.HookLineDefaultAttributes, attrs)
	if err != nil {
		return err
	}
	return l.SetDefaultAttributesSlot(slot)
}

// ExtensionOverrides returns a defensive copy of the plugin-scoped extension overrides payload.
func (l *Line) ExtensionOverrides() map[string]any {
	slot := l.EnsureExtensionOverrides()
	payload := slot.Raw()
	if len(payload) == 0 {
		return nil
	}
	return payload
}

// ApplyExtensionOverrides replaces the extension override payloads for all plugins.
func (l *Line) ApplyExtensionOverrides(overrides map[string]any) error {
	slot, err := slotFromPluginPayloads(extension.HookLineExtensionOverrides, overrides)
	if err != nil {
		return err
	}
	return l.SetExtensionOverridesSlot(slot)
}

// StrainExtensions returns a deep copy of the strain extension container.
func (s *Strain) StrainExtensions() (extension.Container, error) {
	container := s.ensureExtensionContainer()
	return cloneContainer(container)
}

// SetStrainExtensions replaces the strain extension container.
func (s *Strain) SetStrainExtensions(container extension.Container) error {
	return replaceExtensionContainer(&s.extensions, &s.attributesSlot, extension.HookStrainAttributes, container)
}

// GenotypeMarkerExtensions returns a deep copy of the genotype marker extension container.
func (g *GenotypeMarker) GenotypeMarkerExtensions() (extension.Container, error) {
	container := g.ensureExtensionContainer()
	return cloneContainer(container)
}

// SetGenotypeMarkerExtensions replaces the genotype marker extension container.
func (g *GenotypeMarker) SetGenotypeMarkerExtensions(container extension.Container) error {
	return replaceExtensionContainer(&g.extensions, &g.attributesSlot, extension.HookGenotypeMarkerAttributes, container)
}

// ApplyPairingAttributes stores the provided pairing attributes payload. A nil
// payload clears the entry.
func (b *BreedingUnit) ApplyPairingAttributes(attrs map[string]any) error {
	payload, err := extension.NewObjectPayload(extension.HookBreedingUnitPairingAttributes, attrs)
	if err != nil {
		return err
	}
	return b.ApplyPairingAttributesPayload(payload)
}

// ApplyPairingAttributesPayload persists a typed payload for the pairing hook.
func (b *BreedingUnit) ApplyPairingAttributesPayload(payload extension.ObjectPayload) error {
	if err := payload.ExpectHook(extension.HookBreedingUnitPairingAttributes); err != nil {
		return err
	}
	return updateHookPayload(
		func() *extension.Container { return b.ensureExtensionContainer() },
		&b.extensions,
		&b.pairingAttributesSlot,
		extension.HookBreedingUnitPairingAttributes,
		extension.PluginCore,
		payload.Map(),
	)
}

// ObservationExtensions returns a deep copy of the observation extension container.
func (o *Observation) ObservationExtensions() (extension.Container, error) {
	container := o.ensureExtensionContainer()
	return cloneContainer(container)
}

// SetObservationExtensions replaces the observation extension container with
// the provided payload.
func (o *Observation) SetObservationExtensions(container extension.Container) error {
	return replaceExtensionContainer(&o.extensions, &o.dataSlot, extension.HookObservationData, container)
}

// ObservationData returns a copy of the core observation data payload.
func (o *Observation) ObservationData() map[string]any {
	return o.ObservationDataPayload().Map()
}

// ObservationDataPayload returns the observation data payload as a typed wrapper.
func (o *Observation) ObservationDataPayload() extension.ObjectPayload {
	payload, err := extension.ObjectFromContainer(
		o.ensureExtensionContainer(),
		extension.HookObservationData,
		extension.PluginCore,
	)
	panicOnExtension(err, "domain: load observation data payload")
	return payload
}

// ApplyObservationData stores the provided observation data payload. A nil
// payload clears the entry.
func (o *Observation) ApplyObservationData(data map[string]any) error {
	payload, err := extension.NewObjectPayload(extension.HookObservationData, data)
	if err != nil {
		return err
	}
	return o.ApplyObservationDataPayload(payload)
}

// ApplyObservationDataPayload persists a typed payload for the observation hook.
func (o *Observation) ApplyObservationDataPayload(payload extension.ObjectPayload) error {
	if err := payload.ExpectHook(extension.HookObservationData); err != nil {
		return err
	}
	return updateHookPayload(
		func() *extension.Container { return o.ensureExtensionContainer() },
		&o.extensions,
		&o.dataSlot,
		extension.HookObservationData,
		extension.PluginCore,
		payload.Map(),
	)
}

// SampleExtensions returns a deep copy of the sample extension container.
func (s *Sample) SampleExtensions() (extension.Container, error) {
	container := s.ensureExtensionContainer()
	return cloneContainer(container)
}

// SetSampleExtensions replaces the sample extension container with the provided payload.
func (s *Sample) SetSampleExtensions(container extension.Container) error {
	return replaceExtensionContainer(&s.extensions, &s.attributesSlot, extension.HookSampleAttributes, container)
}

// SampleAttributes returns a copy of the core sample attributes payload.
func (s *Sample) SampleAttributes() map[string]any {
	return s.SampleAttributesPayload().Map()
}

// SampleAttributesPayload returns the sample attributes payload as a typed wrapper.
func (s *Sample) SampleAttributesPayload() extension.ObjectPayload {
	payload, err := extension.ObjectFromContainer(
		s.ensureExtensionContainer(),
		extension.HookSampleAttributes,
		extension.PluginCore,
	)
	panicOnExtension(err, "domain: load sample attributes payload")
	return payload
}

// ApplySampleAttributes stores the provided sample attributes payload. A nil
// payload clears the entry.
func (s *Sample) ApplySampleAttributes(attrs map[string]any) error {
	payload, err := extension.NewObjectPayload(extension.HookSampleAttributes, attrs)
	if err != nil {
		return err
	}
	return s.ApplySampleAttributesPayload(payload)
}

// ApplySampleAttributesPayload persists a typed payload for the sample hook.
func (s *Sample) ApplySampleAttributesPayload(payload extension.ObjectPayload) error {
	if err := payload.ExpectHook(extension.HookSampleAttributes); err != nil {
		return err
	}
	return updateHookPayload(
		func() *extension.Container { return s.ensureExtensionContainer() },
		&s.extensions,
		&s.attributesSlot,
		extension.HookSampleAttributes,
		extension.PluginCore,
		payload.Map(),
	)
}

// SupplyItemExtensions returns a deep copy of the supply item extension container.
func (s *SupplyItem) SupplyItemExtensions() (extension.Container, error) {
	container := s.ensureExtensionContainer()
	return cloneContainer(container)
}

// SetSupplyItemExtensions replaces the supply item extension container with the
// provided payload.
func (s *SupplyItem) SetSupplyItemExtensions(container extension.Container) error {
	return replaceExtensionContainer(&s.extensions, &s.attributesSlot, extension.HookSupplyItemAttributes, container)
}

// SupplyAttributes returns a copy of the core supply item attributes payload.
func (s *SupplyItem) SupplyAttributes() map[string]any {
	return s.SupplyAttributesPayload().Map()
}

// SupplyAttributesPayload returns the supply item attributes payload as a typed wrapper.
func (s *SupplyItem) SupplyAttributesPayload() extension.ObjectPayload {
	payload, err := extension.ObjectFromContainer(
		s.ensureExtensionContainer(),
		extension.HookSupplyItemAttributes,
		extension.PluginCore,
	)
	panicOnExtension(err, "domain: load supply item attributes payload")
	return payload
}

// ApplySupplyAttributes stores the provided supply item attributes payload. A
// nil payload clears the entry.
func (s *SupplyItem) ApplySupplyAttributes(attrs map[string]any) error {
	payload, err := extension.NewObjectPayload(extension.HookSupplyItemAttributes, attrs)
	if err != nil {
		return err
	}
	return s.ApplySupplyAttributesPayload(payload)
}

// ApplySupplyAttributesPayload persists a typed payload for the supply hook.
func (s *SupplyItem) ApplySupplyAttributesPayload(payload extension.ObjectPayload) error {
	if err := payload.ExpectHook(extension.HookSupplyItemAttributes); err != nil {
		return err
	}
	return updateHookPayload(
		func() *extension.Container { return s.ensureExtensionContainer() },
		&s.extensions,
		&s.attributesSlot,
		extension.HookSupplyItemAttributes,
		extension.PluginCore,
		payload.Map(),
	)
}
