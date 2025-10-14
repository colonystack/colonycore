package pluginapi

const (
	treatmentStatusPlanned    = "planned"
	treatmentStatusInProgress = "in_progress"
	treatmentStatusCompleted  = "completed"
	treatmentStatusFlagged    = "flagged"
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

// TreatmentStatusRef represents an opaque treatment workflow status.
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
	return treatmentStatusRef{value: treatmentStatusPlanned}
}

func (treatmentStatusProvider) InProgress() TreatmentStatusRef {
	return treatmentStatusRef{value: treatmentStatusInProgress}
}

func (treatmentStatusProvider) Completed() TreatmentStatusRef {
	return treatmentStatusRef{value: treatmentStatusCompleted}
}

func (treatmentStatusProvider) Flagged() TreatmentStatusRef {
	return treatmentStatusRef{value: treatmentStatusFlagged}
}

type treatmentStatusRef struct {
	value string
}

func (t treatmentStatusRef) String() string {
	return t.value
}

func (t treatmentStatusRef) IsActive() bool {
	return t.value == treatmentStatusInProgress
}

func (t treatmentStatusRef) IsCompleted() bool {
	return t.value == treatmentStatusCompleted
}

func (t treatmentStatusRef) IsFlagged() bool {
	return t.value == treatmentStatusFlagged
}

func (t treatmentStatusRef) Equals(other TreatmentStatusRef) bool {
	if otherRef, ok := other.(treatmentStatusRef); ok {
		return t.value == otherRef.value
	}
	return false
}

func (t treatmentStatusRef) isTreatmentStatusRef() {}
