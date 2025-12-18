package pluginapi

import "strings"

const (
	sampleSourceOrganism      = "organism"
	sampleSourceCohort        = "cohort"
	sampleSourceEnvironmental = "environmental"
	sampleSourceUnknown       = "unknown"
)

// SampleContext provides contextual access to sample source and status values.
type SampleContext interface {
	Sources() SampleSourceProvider
	Statuses() SampleStatusProvider
}

// SampleSourceProvider exposes canonical sample source references.
type SampleSourceProvider interface {
	Organism() SampleSourceRef
	Cohort() SampleSourceRef
	Environmental() SampleSourceRef
	Unknown() SampleSourceRef
}

// SampleSourceRef represents an opaque sample source reference.
type SampleSourceRef interface {
	String() string
	IsOrganismDerived() bool
	IsCohortDerived() bool
	IsEnvironmental() bool
	Equals(other SampleSourceRef) bool
	isSampleSourceRef()
}

// SampleStatusProvider exposes canonical sample status references.
type SampleStatusProvider interface {
	Stored() SampleStatusRef
	InTransit() SampleStatusRef
	Consumed() SampleStatusRef
	Disposed() SampleStatusRef
}

// SampleStatusRef represents an opaque sample status reference.
type SampleStatusRef interface {
	String() string
	IsAvailable() bool
	IsTerminal() bool
	Equals(other SampleStatusRef) bool
	isSampleStatusRef()
}

type sampleContext struct{}

// NewSampleContext constructs the default sample context provider.
func NewSampleContext() SampleContext {
	return sampleContext{}
}

func (sampleContext) Sources() SampleSourceProvider {
	return sampleSourceProvider{}
}

func (sampleContext) Statuses() SampleStatusProvider {
	return sampleStatusProvider{}
}

type sampleSourceProvider struct{}

func (sampleSourceProvider) Organism() SampleSourceRef {
	return sampleSourceRef{value: sampleSourceOrganism}
}

func (sampleSourceProvider) Cohort() SampleSourceRef {
	return sampleSourceRef{value: sampleSourceCohort}
}

func (sampleSourceProvider) Environmental() SampleSourceRef {
	return sampleSourceRef{value: sampleSourceEnvironmental}
}

func (sampleSourceProvider) Unknown() SampleSourceRef {
	return sampleSourceRef{value: sampleSourceUnknown}
}

type sampleSourceRef struct {
	value string
}

func (s sampleSourceRef) String() string {
	return s.value
}

func (s sampleSourceRef) IsOrganismDerived() bool {
	return strings.EqualFold(s.value, sampleSourceOrganism)
}

func (s sampleSourceRef) IsCohortDerived() bool {
	return strings.EqualFold(s.value, sampleSourceCohort)
}

func (s sampleSourceRef) IsEnvironmental() bool {
	return strings.EqualFold(s.value, sampleSourceEnvironmental)
}

func (s sampleSourceRef) Equals(other SampleSourceRef) bool {
	if otherRef, ok := other.(sampleSourceRef); ok {
		return strings.EqualFold(s.value, otherRef.value)
	}
	return false
}

func (s sampleSourceRef) isSampleSourceRef() {}

type sampleStatusProvider struct{}

func (sampleStatusProvider) Stored() SampleStatusRef {
	return sampleStatusRef{value: sampleStatusStored}
}

func (sampleStatusProvider) InTransit() SampleStatusRef {
	return sampleStatusRef{value: sampleStatusInTransit}
}

func (sampleStatusProvider) Consumed() SampleStatusRef {
	return sampleStatusRef{value: sampleStatusConsumed}
}

func (sampleStatusProvider) Disposed() SampleStatusRef {
	return sampleStatusRef{value: sampleStatusDisposed}
}

type sampleStatusRef struct {
	value string
}

func (s sampleStatusRef) String() string {
	return s.value
}

func (s sampleStatusRef) IsAvailable() bool {
	return s.value == sampleStatusStored || s.value == sampleStatusInTransit
}

func (s sampleStatusRef) IsTerminal() bool {
	return s.value == sampleStatusConsumed || s.value == sampleStatusDisposed
}

func (s sampleStatusRef) Equals(other SampleStatusRef) bool {
	if otherRef, ok := other.(sampleStatusRef); ok {
		return strings.EqualFold(s.value, otherRef.value)
	}
	return false
}

func (s sampleStatusRef) isSampleStatusRef() {}
