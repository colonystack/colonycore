package datasetapi

const (
	strategyNatural    = "natural"
	strategyArtificial = "artificial"
	strategyControlled = "controlled"
	strategySelective  = "selective"
)

// BreedingContext provides contextual access to breeding-related constants
// without exposing raw constants. This promotes hexagonal architecture
// by decoupling plugins from internal constant definitions.
type BreedingContext interface {
	// StrategyTypes returns contextual references to breeding strategy types
	Natural() BreedingStrategyRef
	Artificial() BreedingStrategyRef
	Controlled() BreedingStrategyRef
	Selective() BreedingStrategyRef
}

// BreedingStrategyRef represents an opaque reference to a breeding strategy.
// This interface prevents direct constant access while providing contextual methods.
type BreedingStrategyRef interface {
	// String returns the string representation of the breeding strategy.
	String() string

	// IsNatural returns true if this is a natural breeding strategy.
	IsNatural() bool

	// RequiresIntervention returns true if this strategy requires human intervention.
	RequiresIntervention() bool

	// Equals compares two BreedingStrategyRef instances for equality.
	Equals(other BreedingStrategyRef) bool

	// Internal marker method to prevent external implementations
	isBreedingStrategyRef()
}

// breedingStrategyRef is the internal implementation of BreedingStrategyRef.
type breedingStrategyRef struct {
	value string
}

func (b breedingStrategyRef) String() string {
	return b.value
}

func (b breedingStrategyRef) IsNatural() bool {
	return b.value == strategyNatural
}

func (b breedingStrategyRef) RequiresIntervention() bool {
	return b.value == strategyArtificial || b.value == strategyControlled || b.value == strategySelective
}

func (b breedingStrategyRef) Equals(other BreedingStrategyRef) bool {
	if otherRef, ok := other.(breedingStrategyRef); ok {
		return b.value == otherRef.value
	}
	return false
}

func (b breedingStrategyRef) isBreedingStrategyRef() {}

// DefaultBreedingContext provides the default breeding implementation.
type DefaultBreedingContext struct{}

// Natural returns the natural breeding strategy reference.
func (DefaultBreedingContext) Natural() BreedingStrategyRef {
	return breedingStrategyRef{value: strategyNatural}
}

// Artificial returns the artificial breeding strategy reference.
func (DefaultBreedingContext) Artificial() BreedingStrategyRef {
	return breedingStrategyRef{value: strategyArtificial}
}

// Controlled returns the controlled breeding strategy reference.
func (DefaultBreedingContext) Controlled() BreedingStrategyRef {
	return breedingStrategyRef{value: strategyControlled}
}

// Selective returns the selective breeding strategy reference.
func (DefaultBreedingContext) Selective() BreedingStrategyRef {
	return breedingStrategyRef{value: strategySelective}
}

// NewBreedingContext creates a new breeding context instance.
func NewBreedingContext() BreedingContext {
	return DefaultBreedingContext{}
}
