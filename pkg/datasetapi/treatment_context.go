package datasetapi

const (
	datasetTreatmentStatusPlanned    = "planned"
	datasetTreatmentStatusInProgress = "in_progress"
	datasetTreatmentStatusCompleted  = "completed"
	datasetTreatmentStatusFlagged    = "flagged"
)

// TreatmentContext provides contextual access to treatment workflow statuses.
type TreatmentContext interface {
	Statuses() TreatmentStatusProvider
}

// TreatmentStatusProvider exposes canonical treatment workflow statuses.
type TreatmentStatusProvider interface {
	Planned() TreatmentStatusRef
	InProgress() TreatmentStatusRef
	Completed() TreatmentStatusRef
	Flagged() TreatmentStatusRef
}

// TreatmentStatusRef represents an opaque treatment workflow status reference.
type TreatmentStatusRef interface {
	String() string
	IsActive() bool
	IsCompleted() bool
	IsFlagged() bool
	Equals(other TreatmentStatusRef) bool
	isTreatmentStatusRef()
}

type treatmentContext struct{}

// NewTreatmentContext constructs the default treatment context provider.
func NewTreatmentContext() TreatmentContext {
	return treatmentContext{}
}

func (treatmentContext) Statuses() TreatmentStatusProvider {
	return treatmentStatusProvider{}
}

type treatmentStatusProvider struct{}

func (treatmentStatusProvider) Planned() TreatmentStatusRef {
	return treatmentStatusRef{value: datasetTreatmentStatusPlanned}
}

func (treatmentStatusProvider) InProgress() TreatmentStatusRef {
	return treatmentStatusRef{value: datasetTreatmentStatusInProgress}
}

func (treatmentStatusProvider) Completed() TreatmentStatusRef {
	return treatmentStatusRef{value: datasetTreatmentStatusCompleted}
}

func (treatmentStatusProvider) Flagged() TreatmentStatusRef {
	return treatmentStatusRef{value: datasetTreatmentStatusFlagged}
}

type treatmentStatusRef struct {
	value string
}

func (t treatmentStatusRef) String() string {
	return t.value
}

func (t treatmentStatusRef) IsActive() bool {
	return t.value == datasetTreatmentStatusInProgress
}

func (t treatmentStatusRef) IsCompleted() bool {
	return t.value == datasetTreatmentStatusCompleted
}

func (t treatmentStatusRef) IsFlagged() bool {
	return t.value == datasetTreatmentStatusFlagged
}

func (t treatmentStatusRef) Equals(other TreatmentStatusRef) bool {
	if otherRef, ok := other.(treatmentStatusRef); ok {
		return t.value == otherRef.value
	}
	return false
}

func (t treatmentStatusRef) isTreatmentStatusRef() {}

// deriveTreatmentStatus infers a treatment workflow status based on log state.
func deriveTreatmentStatus(administrationLog, adverseEvents []string) TreatmentStatusRef {
	statuses := treatmentStatusProvider{}
	switch {
	case len(administrationLog) == 0:
		return statuses.Planned()
	case len(administrationLog) > 0 && len(adverseEvents) == 0:
		return statuses.Completed()
	default:
		return statuses.Flagged()
	}
}
