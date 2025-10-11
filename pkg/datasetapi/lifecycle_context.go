package datasetapi

// LifecycleStageContext provides contextual access to lifecycle stage identifiers
// without exposing raw constants. This promotes hexagonal architecture
// by keeping business logic independent of specific stage representations.
type LifecycleStageContext interface {
	// Planned returns an opaque reference to the planned lifecycle stage.
	Planned() LifecycleStageRef
	// Larva returns an opaque reference to the embryo/larva lifecycle stage.
	Larva() LifecycleStageRef
	// Juvenile returns an opaque reference to the juvenile lifecycle stage.
	Juvenile() LifecycleStageRef
	// Adult returns an opaque reference to the adult lifecycle stage.
	Adult() LifecycleStageRef
	// Retired returns an opaque reference to the retired lifecycle stage.
	Retired() LifecycleStageRef
	// Deceased returns an opaque reference to the deceased lifecycle stage.
	Deceased() LifecycleStageRef
}

// LifecycleStageRef represents an opaque reference to a lifecycle stage.
// Dataset plugins should not inspect or manipulate the underlying value directly.
type LifecycleStageRef interface {
	// String returns the string representation for debugging/logging purposes only.
	// Do not use this value for business logic comparisons.
	String() string
	// IsActive returns true if this lifecycle stage represents an active organism.
	IsActive() bool
	// Equals compares two LifecycleStageRef instances for equality.
	Equals(other LifecycleStageRef) bool
	// Value returns the underlying LifecycleStage value - INTERNAL USE ONLY
	Value() LifecycleStage
	// internal marker to prevent external implementations
	isLifecycleStageRef()
}

// lifecycleStageRef is the internal implementation of LifecycleStageRef.
type lifecycleStageRef struct {
	value LifecycleStage
}

func (s lifecycleStageRef) String() string {
	return string(s.value)
}

func (s lifecycleStageRef) IsActive() bool {
	return s.value != stageDeceased && s.value != stageRetired
}

func (s lifecycleStageRef) Equals(other LifecycleStageRef) bool {
	if otherRef, ok := other.(lifecycleStageRef); ok {
		return s.value == otherRef.value
	}
	return false
}

func (s lifecycleStageRef) Value() LifecycleStage {
	return s.value
}

func (s lifecycleStageRef) isLifecycleStageRef() {}

// newLifecycleStageRef creates a new lifecycle stage reference from the internal LifecycleStage.
func newLifecycleStageRef(stage LifecycleStage) LifecycleStageRef {
	return lifecycleStageRef{value: stage}
}

// lifecycleStageContext is the default implementation of LifecycleStageContext.
type lifecycleStageContext struct{}

func (lifecycleStageContext) Planned() LifecycleStageRef  { return newLifecycleStageRef(stagePlanned) }
func (lifecycleStageContext) Larva() LifecycleStageRef    { return newLifecycleStageRef(stageLarva) }
func (lifecycleStageContext) Juvenile() LifecycleStageRef { return newLifecycleStageRef(stageJuvenile) }
func (lifecycleStageContext) Adult() LifecycleStageRef    { return newLifecycleStageRef(stageAdult) }
func (lifecycleStageContext) Retired() LifecycleStageRef  { return newLifecycleStageRef(stageRetired) }
func (lifecycleStageContext) Deceased() LifecycleStageRef { return newLifecycleStageRef(stageDeceased) }

// NewLifecycleStageContext creates a new lifecycle stage context for accessing stage references.
func NewLifecycleStageContext() LifecycleStageContext {
	return lifecycleStageContext{}
}
