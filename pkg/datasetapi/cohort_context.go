package datasetapi

const (
	purposeResearch     = "research"
	purposeBreeding     = "breeding"
	purposeTeaching     = "teaching"
	purposeConservation = "conservation"
	purposeProduction   = "production"
)

// CohortContext provides contextual access to cohort-related constants
// without exposing raw constants. This promotes hexagonal architecture
// by decoupling plugins from internal constant definitions.
type CohortContext interface {
	// PurposeTypes returns contextual references to cohort purpose types
	Research() CohortPurposeRef
	Breeding() CohortPurposeRef
	Teaching() CohortPurposeRef
	Conservation() CohortPurposeRef
	Production() CohortPurposeRef
}

// CohortPurposeRef represents an opaque reference to a cohort purpose.
// This interface prevents direct constant access while providing contextual methods.
type CohortPurposeRef interface {
	// String returns the string representation of the cohort purpose.
	String() string

	// IsResearch returns true if this is a research purpose.
	IsResearch() bool

	// RequiresProtocol returns true if this purpose requires protocol compliance.
	RequiresProtocol() bool

	// Equals compares two CohortPurposeRef instances for equality.
	Equals(other CohortPurposeRef) bool

	// Internal marker method to prevent external implementations
	isCohortPurposeRef()
}

// cohortPurposeRef is the internal implementation of CohortPurposeRef.
type cohortPurposeRef struct {
	value string
}

func (c cohortPurposeRef) String() string {
	return c.value
}

func (c cohortPurposeRef) IsResearch() bool {
	return c.value == purposeResearch
}

func (c cohortPurposeRef) RequiresProtocol() bool {
	return c.value == purposeResearch || c.value == purposeTeaching
}

func (c cohortPurposeRef) Equals(other CohortPurposeRef) bool {
	if otherRef, ok := other.(cohortPurposeRef); ok {
		return c.value == otherRef.value
	}
	return false
}

func (c cohortPurposeRef) isCohortPurposeRef() {}

// DefaultCohortContext provides the default cohort implementation.
type DefaultCohortContext struct{}

// Research returns the research cohort purpose reference.
func (DefaultCohortContext) Research() CohortPurposeRef {
	return cohortPurposeRef{value: purposeResearch}
}

// Breeding returns the breeding cohort purpose reference.
func (DefaultCohortContext) Breeding() CohortPurposeRef {
	return cohortPurposeRef{value: purposeBreeding}
}

// Teaching returns the teaching cohort purpose reference.
func (DefaultCohortContext) Teaching() CohortPurposeRef {
	return cohortPurposeRef{value: purposeTeaching}
}

// Conservation returns the conservation cohort purpose reference.
func (DefaultCohortContext) Conservation() CohortPurposeRef {
	return cohortPurposeRef{value: purposeConservation}
}

// Production returns the production cohort purpose reference.
func (DefaultCohortContext) Production() CohortPurposeRef {
	return cohortPurposeRef{value: purposeProduction}
}

// NewCohortContext creates a new cohort context instance.
func NewCohortContext() CohortContext {
	return DefaultCohortContext{}
}
