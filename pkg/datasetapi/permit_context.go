package datasetapi

// PermitContext provides contextual access to permit validity statuses.
type PermitContext interface {
	Statuses() PermitStatusProvider
}

// PermitStatusProvider exposes canonical permit validity references.
type PermitStatusProvider interface {
	Draft() PermitStatusRef
	Submitted() PermitStatusRef
	Approved() PermitStatusRef
	OnHold() PermitStatusRef
	Expired() PermitStatusRef
	Archived() PermitStatusRef
}

// PermitStatusRef represents an opaque permit status reference.
type PermitStatusRef interface {
	String() string
	IsActive() bool
	IsExpired() bool
	IsArchived() bool
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

func (permitStatusProvider) Draft() PermitStatusRef {
	return permitStatusRef{value: datasetPermitStatusDraft}
}

func (permitStatusProvider) Submitted() PermitStatusRef {
	return permitStatusRef{value: datasetPermitStatusSubmitted}
}

func (permitStatusProvider) Approved() PermitStatusRef {
	return permitStatusRef{value: datasetPermitStatusApproved}
}

func (permitStatusProvider) OnHold() PermitStatusRef {
	return permitStatusRef{value: datasetPermitStatusOnHold}
}

func (permitStatusProvider) Expired() PermitStatusRef {
	return permitStatusRef{value: datasetPermitStatusExpired}
}

func (permitStatusProvider) Archived() PermitStatusRef {
	return permitStatusRef{value: datasetPermitStatusArchived}
}

type permitStatusRef struct {
	value string
}

func (p permitStatusRef) String() string {
	return p.value
}

func (p permitStatusRef) IsActive() bool {
	return p.value == datasetPermitStatusApproved
}

func (p permitStatusRef) IsExpired() bool {
	return p.value == datasetPermitStatusExpired
}

func (p permitStatusRef) IsArchived() bool {
	return p.value == datasetPermitStatusArchived
}

func (p permitStatusRef) Equals(other PermitStatusRef) bool {
	if otherRef, ok := other.(permitStatusRef); ok {
		return p.value == otherRef.value
	}
	return false
}

func (p permitStatusRef) isPermitStatusRef() {}
