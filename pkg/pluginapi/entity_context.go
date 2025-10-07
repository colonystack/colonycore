package pluginapi

// EntityContext provides contextual access to entity type identifiers
// without exposing raw constants. This promotes hexagonal architecture
// by keeping business logic independent of specific entity representations.
type EntityContext interface {
	// Organism returns an opaque reference to an organism entity type.
	Organism() EntityTypeRef
	// Housing returns an opaque reference to a housing unit entity type.
	Housing() EntityTypeRef
	// Protocol returns an opaque reference to a protocol entity type.
	Protocol() EntityTypeRef
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
	return e.value == EntityOrganism || e.value == EntityHousingUnit
}

func (e entityTypeRef) Equals(other EntityTypeRef) bool {
	if otherRef, ok := other.(entityTypeRef); ok {
		return e.value == otherRef.value
	}
	return false
}

func (e entityTypeRef) isEntityTypeRef() {}

// newEntityTypeRef creates a new entity type reference from the internal EntityType.
func newEntityTypeRef(entityType EntityType) EntityTypeRef {
	return entityTypeRef{value: entityType}
}

// entityContext is the default implementation of EntityContext.
type entityContext struct{}

func (entityContext) Organism() EntityTypeRef { return newEntityTypeRef(EntityOrganism) }
func (entityContext) Housing() EntityTypeRef  { return newEntityTypeRef(EntityHousingUnit) }
func (entityContext) Protocol() EntityTypeRef { return newEntityTypeRef(EntityProtocol) }

// NewEntityContext creates a new entity context for accessing entity type references.
func NewEntityContext() EntityContext {
	return entityContext{}
}
