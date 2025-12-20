package pluginapi

// HousingContext provides contextual access to housing-related constants
// without exposing raw constants. This promotes hexagonal architecture
// by decoupling plugins from internal constant definitions.
type HousingContext interface {
	// EnvironmentTypes returns contextual references to environment types
	Aquatic() EnvironmentTypeRef
	Terrestrial() EnvironmentTypeRef
	Arboreal() EnvironmentTypeRef
	Humid() EnvironmentTypeRef
}

// EnvironmentTypeRef represents an opaque reference to an environment type.
// This interface prevents direct constant access while providing contextual methods.
type EnvironmentTypeRef interface {
	// String returns the string representation of the environment type.
	String() string

	// IsAquatic returns true if this environment supports aquatic species.
	IsAquatic() bool

	// IsHumid returns true if this environment has high humidity.
	IsHumid() bool

	// Equals compares two EnvironmentTypeRef instances for equality.
	Equals(other EnvironmentTypeRef) bool

	// Internal marker method to prevent external implementations
	isEnvironmentTypeRef()
}

// environmentTypeRef is the internal implementation of EnvironmentTypeRef.
type environmentTypeRef struct {
	value string
}

func (e environmentTypeRef) String() string {
	return e.value
}

func (e environmentTypeRef) IsAquatic() bool {
	return e.value == environmentTypeAquatic || e.value == "semi-aquatic"
}

func (e environmentTypeRef) IsHumid() bool {
	return e.value == environmentTypeHumid || e.value == environmentTypeAquatic || e.value == "tropical"
}

func (e environmentTypeRef) Equals(other EnvironmentTypeRef) bool {
	if otherRef, ok := other.(environmentTypeRef); ok {
		return e.value == otherRef.value
	}
	return false
}

func (e environmentTypeRef) isEnvironmentTypeRef() {}

// DefaultHousingContext provides the default housing implementation.
type DefaultHousingContext struct{}

// Aquatic returns the aquatic environment type reference.
func (DefaultHousingContext) Aquatic() EnvironmentTypeRef {
	return environmentTypeRef{value: environmentTypeAquatic}
}

// Terrestrial returns the terrestrial environment type reference.
func (DefaultHousingContext) Terrestrial() EnvironmentTypeRef {
	return environmentTypeRef{value: environmentTypeTerrestrial}
}

// Arboreal returns the arboreal environment type reference.
func (DefaultHousingContext) Arboreal() EnvironmentTypeRef {
	return environmentTypeRef{value: environmentTypeArboreal}
}

// Humid returns the humid environment type reference.
func (DefaultHousingContext) Humid() EnvironmentTypeRef {
	return environmentTypeRef{value: environmentTypeHumid}
}

// NewHousingContext creates a new housing context instance.
func NewHousingContext() HousingContext {
	return DefaultHousingContext{}
}
