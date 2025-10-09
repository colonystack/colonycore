package pluginapi

// ProtocolContext provides contextual access to protocol-related constants
// without exposing raw constants. This promotes hexagonal architecture
// by decoupling plugins from internal constant definitions.
type ProtocolContext interface {
	// StatusTypes returns contextual references to protocol status types
	Draft() ProtocolStatusRef
	Active() ProtocolStatusRef
	Suspended() ProtocolStatusRef
	Completed() ProtocolStatusRef
	Cancelled() ProtocolStatusRef
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
	return p.value == "active"
}

func (p protocolStatusRef) IsTerminal() bool {
	return p.value == "completed" || p.value == "cancelled"
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
	return protocolStatusRef{value: "draft"}
}

// Active returns the active protocol status reference.
func (DefaultProtocolContext) Active() ProtocolStatusRef {
	return protocolStatusRef{value: "active"}
}

// Suspended returns the suspended protocol status reference.
func (DefaultProtocolContext) Suspended() ProtocolStatusRef {
	return protocolStatusRef{value: "suspended"}
}

// Completed returns the completed protocol status reference.
func (DefaultProtocolContext) Completed() ProtocolStatusRef {
	return protocolStatusRef{value: "completed"}
}

// Cancelled returns the cancelled protocol status reference.
func (DefaultProtocolContext) Cancelled() ProtocolStatusRef {
	return protocolStatusRef{value: "cancelled"}
}

// NewProtocolContext creates a new protocol context instance.
func NewProtocolContext() ProtocolContext {
	return DefaultProtocolContext{}
}
