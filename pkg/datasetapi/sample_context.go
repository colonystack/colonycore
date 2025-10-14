package datasetapi

import "strings"

const (
	datasetSampleSourceOrganism      = "organism"
	datasetSampleSourceCohort        = "cohort"
	datasetSampleSourceEnvironmental = "environmental"
	datasetSampleSourceUnknown       = "unknown"

	datasetSampleStatusStored    = "stored"
	datasetSampleStatusInTransit = "in_transit"
	datasetSampleStatusConsumed  = "consumed"
	datasetSampleStatusDisposed  = "disposed"
)

// SampleContext provides contextual access to sample source and status references.
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
	return sampleSourceRef{value: datasetSampleSourceOrganism}
}

func (sampleSourceProvider) Cohort() SampleSourceRef {
	return sampleSourceRef{value: datasetSampleSourceCohort}
}

func (sampleSourceProvider) Environmental() SampleSourceRef {
	return sampleSourceRef{value: datasetSampleSourceEnvironmental}
}

func (sampleSourceProvider) Unknown() SampleSourceRef {
	return sampleSourceRef{value: datasetSampleSourceUnknown}
}

type sampleSourceRef struct {
	value string
}

func (s sampleSourceRef) String() string {
	return s.value
}

func (s sampleSourceRef) IsOrganismDerived() bool {
	return strings.EqualFold(s.value, datasetSampleSourceOrganism)
}

func (s sampleSourceRef) IsCohortDerived() bool {
	return strings.EqualFold(s.value, datasetSampleSourceCohort)
}

func (s sampleSourceRef) IsEnvironmental() bool {
	return strings.EqualFold(s.value, datasetSampleSourceEnvironmental)
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
	return sampleStatusRef{value: datasetSampleStatusStored}
}

func (sampleStatusProvider) InTransit() SampleStatusRef {
	return sampleStatusRef{value: datasetSampleStatusInTransit}
}

func (sampleStatusProvider) Consumed() SampleStatusRef {
	return sampleStatusRef{value: datasetSampleStatusConsumed}
}

func (sampleStatusProvider) Disposed() SampleStatusRef {
	return sampleStatusRef{value: datasetSampleStatusDisposed}
}

type sampleStatusRef struct {
	value string
}

func (s sampleStatusRef) String() string {
	return s.value
}

func (s sampleStatusRef) IsAvailable() bool {
	return s.value == datasetSampleStatusStored || s.value == datasetSampleStatusInTransit
}

func (s sampleStatusRef) IsTerminal() bool {
	return s.value == datasetSampleStatusConsumed || s.value == datasetSampleStatusDisposed
}

func (s sampleStatusRef) Equals(other SampleStatusRef) bool {
	if otherRef, ok := other.(sampleStatusRef); ok {
		return strings.EqualFold(s.value, otherRef.value)
	}
	return false
}

func (s sampleStatusRef) isSampleStatusRef() {}
