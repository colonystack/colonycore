package pluginapi

const (
	protocolStatusDraft     = "draft"
	protocolStatusSubmitted = "submitted"
	protocolStatusApproved  = "approved"
	protocolStatusOnHold    = "on_hold"
	protocolStatusExpired   = "expired"
	protocolStatusArchived  = "archived"
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
	return p.value == protocolStatusApproved
}

func (p protocolStatusRef) IsTerminal() bool {
	return p.value == protocolStatusExpired || p.value == protocolStatusArchived
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
	return protocolStatusRef{value: protocolStatusDraft}
}

// Submitted returns the submitted protocol status reference.
func (DefaultProtocolContext) Submitted() ProtocolStatusRef {
	return protocolStatusRef{value: protocolStatusSubmitted}
}

// Approved returns the approved protocol status reference.
func (DefaultProtocolContext) Approved() ProtocolStatusRef {
	return protocolStatusRef{value: protocolStatusApproved}
}

// OnHold returns the on_hold protocol status reference.
func (DefaultProtocolContext) OnHold() ProtocolStatusRef {
	return protocolStatusRef{value: protocolStatusOnHold}
}

// Expired returns the expired protocol status reference.
func (DefaultProtocolContext) Expired() ProtocolStatusRef {
	return protocolStatusRef{value: protocolStatusExpired}
}

// Archived returns the archived protocol status reference.
func (DefaultProtocolContext) Archived() ProtocolStatusRef {
	return protocolStatusRef{value: protocolStatusArchived}
}

// NewProtocolContext creates a new protocol context instance.
func NewProtocolContext() ProtocolContext {
	return DefaultProtocolContext{}
}
