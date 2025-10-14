package pluginapi

const (
	permitStatusPending = "pending"
	permitStatusActive  = "active"
	permitStatusExpired = "expired"
)

// PermitContext provides contextual access to permit status references.
type PermitContext interface {
	Statuses() PermitStatusProvider
}

// PermitStatusProvider exposes canonical permit validity references.
type PermitStatusProvider interface {
	Pending() PermitStatusRef
	Active() PermitStatusRef
	Expired() PermitStatusRef
}

// PermitStatusRef represents an opaque permit status value.
type PermitStatusRef interface {
	String() string
	IsActive() bool
	IsExpired() bool
	Equals(other PermitStatusRef) bool
	isPermitStatusRef()
}

type permitContext struct{}

// NewPermitContext constructs the default permit context provider.
func NewPermitContext() PermitContext {
	return permitContext{}
}

func (permitContext) Statuses() PermitStatusProvider {
	return permitStatusProvider{}
}

type permitStatusProvider struct{}

func (permitStatusProvider) Pending() PermitStatusRef {
	return permitStatusRef{value: permitStatusPending}
}

func (permitStatusProvider) Active() PermitStatusRef {
	return permitStatusRef{value: permitStatusActive}
}

func (permitStatusProvider) Expired() PermitStatusRef {
	return permitStatusRef{value: permitStatusExpired}
}

type permitStatusRef struct {
	value string
}

func (p permitStatusRef) String() string {
	return p.value
}

func (p permitStatusRef) IsActive() bool {
	return p.value == permitStatusActive
}

func (p permitStatusRef) IsExpired() bool {
	return p.value == permitStatusExpired
}

func (p permitStatusRef) Equals(other PermitStatusRef) bool {
	if otherRef, ok := other.(permitStatusRef); ok {
		return p.value == otherRef.value
	}
	return false
}

func (p permitStatusRef) isPermitStatusRef() {}
