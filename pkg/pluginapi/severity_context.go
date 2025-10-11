package pluginapi

// SeverityContext provides contextual access to severity levels
// without exposing raw constants. This promotes hexagonal architecture
// by keeping business logic independent of specific severity representations.
type SeverityContext interface {
	// Log returns an opaque reference to a logging severity level.
	Log() SeverityRef
	// Warn returns an opaque reference to a warning severity level.
	Warn() SeverityRef
	// Block returns an opaque reference to a blocking severity level.
	Block() SeverityRef
}

// SeverityRef represents an opaque reference to a severity level.
// Plugin rules should not inspect or manipulate the underlying value directly.
type SeverityRef interface {
	// String returns the string representation for debugging/logging purposes only.
	// Do not use this value for business logic comparisons.
	String() string
	// IsBlocking returns true if this severity level should block operations.
	IsBlocking() bool
	// Equals compares two SeverityRef instances for equality.
	Equals(other SeverityRef) bool
	// internal marker to prevent external implementations
	isSeverityRef()
}

// severityRef is the internal implementation of SeverityRef.
type severityRef struct {
	value Severity
}

func (s severityRef) String() string {
	return string(s.value)
}

func (s severityRef) IsBlocking() bool {
	return s.value == severityBlock
}

func (s severityRef) Equals(other SeverityRef) bool {
	if otherRef, ok := other.(severityRef); ok {
		return s.value == otherRef.value
	}
	return false
}

func (s severityRef) isSeverityRef() {}

// newSeverityRef creates a new severity reference from the internal Severity.
func newSeverityRef(severity Severity) SeverityRef {
	return severityRef{value: severity}
}

// severityContext is the default implementation of SeverityContext.
type severityContext struct{}

func (severityContext) Log() SeverityRef   { return newSeverityRef(severityLog) }
func (severityContext) Warn() SeverityRef  { return newSeverityRef(severityWarn) }
func (severityContext) Block() SeverityRef { return newSeverityRef(severityBlock) }

// NewSeverityContext creates a new severity context for accessing severity references.
func NewSeverityContext() SeverityContext {
	return severityContext{}
}
