package pluginapi

// EntityContext provides contextual access to entity type identifiers
// without exposing raw constants. This promotes hexagonal architecture
// by keeping business logic independent of specific entity representations.
type EntityContext interface {
	// Organism returns an opaque reference to an organism entity type.
	Organism() EntityTypeRef
	// Housing returns an opaque reference to a housing unit entity type.
	Housing() EntityTypeRef
	// Facility returns an opaque reference to a facility entity type.
	Facility() EntityTypeRef
	// Protocol returns an opaque reference to a protocol entity type.
	Protocol() EntityTypeRef
	// Procedure returns an opaque reference to a procedure entity type.
	Procedure() EntityTypeRef
	// Treatment returns an opaque reference to a treatment entity type.
	Treatment() EntityTypeRef
	// Observation returns an opaque reference to an observation entity type.
	Observation() EntityTypeRef
	// Sample returns an opaque reference to a sample entity type.
	Sample() EntityTypeRef
	// Permit returns an opaque reference to a permit entity type.
	Permit() EntityTypeRef
	// Project returns an opaque reference to a project entity type.
	Project() EntityTypeRef
	// SupplyItem returns an opaque reference to a supply item entity type.
	SupplyItem() EntityTypeRef
}

// EntityTypeRef represents an opaque reference to an entity type.
// Plugin rules should not inspect or manipulate the underlying value directly.
type EntityTypeRef interface {
	// String returns the string representation for debugging/logging purposes only.
	// Do not use this value for business logic comparisons.
	String() string
	// IsCore returns true if this entity type represents core colony data.
	IsCore() bool
	// Equals compares two EntityTypeRef instances for equality.
	Equals(other EntityTypeRef) bool
	// Value returns the underlying EntityType value - INTERNAL USE ONLY
	Value() EntityType
	// internal marker to prevent external implementations
	isEntityTypeRef()
}

// entityTypeRef is the internal implementation of EntityTypeRef.
type entityTypeRef struct {
	value EntityType
}

func (e entityTypeRef) String() string {
	return string(e.value)
}

func (e entityTypeRef) IsCore() bool {
	switch e.value {
	case entityOrganism,
		entityHousingUnit,
		entityFacility,
		entityProcedure,
		entityTreatment,
		entityObservation,
		entitySample,
		entityProtocol,
		entityPermit,
		entityProject,
		entitySupplyItem:
		return true
	default:
		return false
	}
}

func (e entityTypeRef) Equals(other EntityTypeRef) bool {
	if otherRef, ok := other.(entityTypeRef); ok {
		return e.value == otherRef.value
	}
	return false
}

func (e entityTypeRef) Value() EntityType {
	return e.value
}

func (e entityTypeRef) isEntityTypeRef() {}

// newEntityTypeRef creates a new entity type reference from the internal EntityType.
func newEntityTypeRef(entityType EntityType) EntityTypeRef {
	return entityTypeRef{value: entityType}
}

// entityContext is the default implementation of EntityContext.
type entityContext struct{}

func (entityContext) Organism() EntityTypeRef  { return newEntityTypeRef(entityOrganism) }
func (entityContext) Housing() EntityTypeRef   { return newEntityTypeRef(entityHousingUnit) }
func (entityContext) Facility() EntityTypeRef  { return newEntityTypeRef(entityFacility) }
func (entityContext) Protocol() EntityTypeRef  { return newEntityTypeRef(entityProtocol) }
func (entityContext) Procedure() EntityTypeRef { return newEntityTypeRef(entityProcedure) }
func (entityContext) Treatment() EntityTypeRef { return newEntityTypeRef(entityTreatment) }
func (entityContext) Observation() EntityTypeRef {
	return newEntityTypeRef(entityObservation)
}
func (entityContext) Sample() EntityTypeRef  { return newEntityTypeRef(entitySample) }
func (entityContext) Permit() EntityTypeRef  { return newEntityTypeRef(entityPermit) }
func (entityContext) Project() EntityTypeRef { return newEntityTypeRef(entityProject) }
func (entityContext) SupplyItem() EntityTypeRef {
	return newEntityTypeRef(entitySupplyItem)
}

// NewEntityContext creates a new entity context for accessing entity type references.
func NewEntityContext() EntityContext {
	return entityContext{}
}
