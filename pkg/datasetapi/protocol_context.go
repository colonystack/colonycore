package datasetapi

const (
	datasetProtocolStatusDraft     = "draft"
	datasetProtocolStatusSubmitted = "submitted"
	datasetProtocolStatusApproved  = "approved"
	datasetProtocolStatusOnHold    = "on_hold"
	datasetProtocolStatusExpired   = "expired"
	datasetProtocolStatusArchived  = "archived"
)

// ProtocolContext provides contextual access to protocol-related constants
// without exposing raw constants. This promotes hexagonal architecture
// by decoupling plugins from internal constant definitions.
type ProtocolContext interface {
	// StatusTypes returns contextual references to protocol status types
	Draft() ProtocolStatusRef
	Submitted() ProtocolStatusRef
	Approved() ProtocolStatusRef
	OnHold() ProtocolStatusRef
	Expired() ProtocolStatusRef
	Archived() ProtocolStatusRef
}

// ProtocolStatusRef represents an opaque reference to a protocol status.
// This interface prevents direct constant access while providing contextual methods.
type ProtocolStatusRef interface {
	// String returns the string representation of the protocol status.
	String() string

	// IsActive returns true if this status indicates an active protocol.
	IsActive() bool

	// IsTerminal returns true if this status indicates a terminal state.
	IsTerminal() bool

	// Equals compares two ProtocolStatusRef instances for equality.
	Equals(other ProtocolStatusRef) bool

	// Internal marker method to prevent external implementations
	isProtocolStatusRef()
}

// protocolStatusRef is the internal implementation of ProtocolStatusRef.
type protocolStatusRef struct {
	value string
}

func (p protocolStatusRef) String() string {
	return p.value
}

func (p protocolStatusRef) IsActive() bool {
	return p.value == datasetProtocolStatusApproved
}

func (p protocolStatusRef) IsTerminal() bool {
	return p.value == datasetProtocolStatusExpired || p.value == datasetProtocolStatusArchived
}

func (p protocolStatusRef) Equals(other ProtocolStatusRef) bool {
	if otherRef, ok := other.(protocolStatusRef); ok {
		return p.value == otherRef.value
	}
	return false
}

func (p protocolStatusRef) isProtocolStatusRef() {}

// DefaultProtocolContext provides the default protocol implementation.
type DefaultProtocolContext struct{}

// Draft returns the draft protocol status reference.
func (DefaultProtocolContext) Draft() ProtocolStatusRef {
	return protocolStatusRef{value: datasetProtocolStatusDraft}
}

// Submitted returns the submitted protocol status reference.
func (DefaultProtocolContext) Submitted() ProtocolStatusRef {
	return protocolStatusRef{value: datasetProtocolStatusSubmitted}
}

// Approved returns the approved protocol status reference.
func (DefaultProtocolContext) Approved() ProtocolStatusRef {
	return protocolStatusRef{value: datasetProtocolStatusApproved}
}

// OnHold returns the on_hold protocol status reference.
func (DefaultProtocolContext) OnHold() ProtocolStatusRef {
	return protocolStatusRef{value: datasetProtocolStatusOnHold}
}

// Expired returns the expired protocol status reference.
func (DefaultProtocolContext) Expired() ProtocolStatusRef {
	return protocolStatusRef{value: datasetProtocolStatusExpired}
}

// Archived returns the archived protocol status reference.
func (DefaultProtocolContext) Archived() ProtocolStatusRef {
	return protocolStatusRef{value: datasetProtocolStatusArchived}
}

// NewProtocolContext creates a new protocol context instance.
func NewProtocolContext() ProtocolContext {
	return DefaultProtocolContext{}
}
