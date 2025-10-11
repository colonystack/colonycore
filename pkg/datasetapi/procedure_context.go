package datasetapi

// ProcedureContext provides contextual access to procedure-related constants
// without exposing raw constants. This promotes hexagonal architecture
// by decoupling plugins from internal constant definitions.
type ProcedureContext interface {
	// StatusTypes returns contextual references to procedure status types
	Scheduled() ProcedureStatusRef
	InProgress() ProcedureStatusRef
	Completed() ProcedureStatusRef
	Cancelled() ProcedureStatusRef
	Failed() ProcedureStatusRef
}

// ProcedureStatusRef represents an opaque reference to a procedure status.
// This interface prevents direct constant access while providing contextual methods.
type ProcedureStatusRef interface {
	// String returns the string representation of the procedure status.
	String() string

	// IsActive returns true if this status indicates an active procedure.
	IsActive() bool

	// IsTerminal returns true if this status indicates a terminal state.
	IsTerminal() bool

	// IsSuccessful returns true if this status indicates successful completion.
	IsSuccessful() bool

	// Equals compares two ProcedureStatusRef instances for equality.
	Equals(other ProcedureStatusRef) bool

	// Internal marker method to prevent external implementations
	isProcedureStatusRef()
}

// procedureStatusRef is the internal implementation of ProcedureStatusRef.
type procedureStatusRef struct {
	value string
}

func (p procedureStatusRef) String() string {
	return p.value
}

func (p procedureStatusRef) IsActive() bool {
	return p.value == "in_progress"
}

func (p procedureStatusRef) IsTerminal() bool {
	return p.value == statusCompleted || p.value == statusCancelled || p.value == "failed"
}

func (p procedureStatusRef) IsSuccessful() bool {
	return p.value == statusCompleted
}

func (p procedureStatusRef) Equals(other ProcedureStatusRef) bool {
	if otherRef, ok := other.(procedureStatusRef); ok {
		return p.value == otherRef.value
	}
	return false
}

func (p procedureStatusRef) isProcedureStatusRef() {}

// DefaultProcedureContext provides the default procedure implementation.
type DefaultProcedureContext struct{}

// Scheduled returns the scheduled procedure status reference.
func (DefaultProcedureContext) Scheduled() ProcedureStatusRef {
	return procedureStatusRef{value: "scheduled"}
}

// InProgress returns the in progress procedure status reference.
func (DefaultProcedureContext) InProgress() ProcedureStatusRef {
	return procedureStatusRef{value: "in_progress"}
}

// Completed returns the completed procedure status reference.
func (DefaultProcedureContext) Completed() ProcedureStatusRef {
	return procedureStatusRef{value: "completed"}
}

// Cancelled returns the cancelled procedure status reference.
func (DefaultProcedureContext) Cancelled() ProcedureStatusRef {
	return procedureStatusRef{value: "cancelled"}
}

// Failed returns the failed procedure status reference.
func (DefaultProcedureContext) Failed() ProcedureStatusRef {
	return procedureStatusRef{value: "failed"}
}

// NewProcedureContext creates a new procedure context instance.
func NewProcedureContext() ProcedureContext {
	return DefaultProcedureContext{}
}
