package datasetapi

// ExtensionHookContext exposes opaque references for known dataset extension hooks.
// Use this context instead of comparing against raw string constants.
type ExtensionHookContext interface {
	OrganismAttributes() HookRef
	FacilityEnvironmentBaselines() HookRef
	BreedingUnitPairingAttributes() HookRef
	ObservationData() HookRef
	SampleAttributes() HookRef
	SupplyItemAttributes() HookRef
}

// ExtensionContributorContext exposes opaque plugin identifiers for extension payloads.
type ExtensionContributorContext interface {
	Core() PluginRef
	Custom(id string) PluginRef
}

// HookRef represents an opaque reference to an extension hook identifier.
type HookRef interface {
	String() string
	Equals(HookRef) bool
	value() string
	isHookRef()
}

// PluginRef represents an opaque reference to a plugin identifier contributing extension data.
type PluginRef interface {
	String() string
	Equals(PluginRef) bool
	value() string
	isPluginRef()
}

// NewExtensionHookContext constructs a context for accessing known extension hooks.
func NewExtensionHookContext() ExtensionHookContext {
	return extensionHookContext{}
}

// NewExtensionContributorContext constructs a context for accessing plugin identifiers.
func NewExtensionContributorContext() ExtensionContributorContext {
	return extensionContributorContext{}
}

type extensionHookContext struct{}

const (
	hookOrganismAttributes           = "entity.organism.attributes"
	hookFacilityEnvironmentBaselines = "entity.facility.environment_baselines"
	hookBreedingUnitPairing          = "entity.breeding_unit.pairing_attributes"
	hookObservationData              = "entity.observation.data"
	hookSampleAttributes             = "entity.sample.attributes"
	hookSupplyItemAttributes         = "entity.supply_item.attributes"
)

func (extensionHookContext) OrganismAttributes() HookRef {
	return hookRef{identifier: hookOrganismAttributes}
}

func (extensionHookContext) FacilityEnvironmentBaselines() HookRef {
	return hookRef{identifier: hookFacilityEnvironmentBaselines}
}

func (extensionHookContext) BreedingUnitPairingAttributes() HookRef {
	return hookRef{identifier: hookBreedingUnitPairing}
}

func (extensionHookContext) ObservationData() HookRef {
	return hookRef{identifier: hookObservationData}
}

func (extensionHookContext) SampleAttributes() HookRef {
	return hookRef{identifier: hookSampleAttributes}
}

func (extensionHookContext) SupplyItemAttributes() HookRef {
	return hookRef{identifier: hookSupplyItemAttributes}
}

type extensionContributorContext struct{}

const pluginCoreIdentifier = "core"

func (extensionContributorContext) Core() PluginRef {
	return pluginRef{identifier: pluginCoreIdentifier}
}

func (extensionContributorContext) Custom(id string) PluginRef {
	return pluginRef{identifier: id}
}

type hookRef struct {
	identifier string
}

func (r hookRef) String() string {
	return r.identifier
}

func (r hookRef) Equals(other HookRef) bool {
	switch o := other.(type) {
	case hookRef:
		return r.identifier == o.identifier
	case *hookRef:
		if o == nil {
			return false
		}
		return r.identifier == o.identifier
	default:
		return false
	}
}

func (r hookRef) value() string {
	return r.identifier
}

func (r hookRef) isHookRef() {}

type pluginRef struct {
	identifier string
}

func (r pluginRef) String() string {
	return r.identifier
}

func (r pluginRef) Equals(other PluginRef) bool {
	switch o := other.(type) {
	case pluginRef:
		return r.identifier == o.identifier
	case *pluginRef:
		if o == nil {
			return false
		}
		return r.identifier == o.identifier
	default:
		return false
	}
}

func (r pluginRef) value() string {
	return r.identifier
}

func (r pluginRef) isPluginRef() {}
