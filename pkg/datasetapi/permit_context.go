package datasetapi

const (
	datasetPermitStatusPending = "pending"
	datasetPermitStatusActive  = "active"
	datasetPermitStatusExpired = "expired"
)

// PermitContext provides contextual access to permit validity statuses.
type PermitContext interface {
	Statuses() PermitStatusProvider
}

// PermitStatusProvider exposes canonical permit validity references.
type PermitStatusProvider interface {
	Pending() PermitStatusRef
	Active() PermitStatusRef
	Expired() PermitStatusRef
}

// PermitStatusRef represents an opaque permit status reference.
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
	return permitStatusRef{value: datasetPermitStatusPending}
}

func (permitStatusProvider) Active() PermitStatusRef {
	return permitStatusRef{value: datasetPermitStatusActive}
}

func (permitStatusProvider) Expired() PermitStatusRef {
	return permitStatusRef{value: datasetPermitStatusExpired}
}

type permitStatusRef struct {
	value string
}

func (p permitStatusRef) String() string {
	return p.value
}

func (p permitStatusRef) IsActive() bool {
	return p.value == datasetPermitStatusActive
}

func (p permitStatusRef) IsExpired() bool {
	return p.value == datasetPermitStatusExpired
}

func (p permitStatusRef) Equals(other PermitStatusRef) bool {
	if otherRef, ok := other.(permitStatusRef); ok {
		return p.value == otherRef.value
	}
	return false
}

func (p permitStatusRef) isPermitStatusRef() {}
