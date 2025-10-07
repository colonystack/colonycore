package pluginapi

// ActionContext provides contextual access to action type identifiers
// without exposing raw constants. This promotes hexagonal architecture
// by keeping business logic independent of specific action representations.
type ActionContext interface {
	// Create returns an opaque reference to a create action.
	Create() ActionRef
	// Update returns an opaque reference to an update action.
	Update() ActionRef
	// Delete returns an opaque reference to a delete action.
	Delete() ActionRef
}

// ActionRef represents an opaque reference to an action type.
// Plugin rules should not inspect or manipulate the underlying value directly.
type ActionRef interface {
	// String returns the string representation for debugging/logging purposes only.
	// Do not use this value for business logic comparisons.
	String() string
	// IsMutation returns true if this action modifies state (create, update, delete all return true).
	IsMutation() bool
	// IsDestructive returns true if this action removes data (delete returns true).
	IsDestructive() bool
	// Equals compares two ActionRef instances for equality.
	Equals(other ActionRef) bool
	// internal marker to prevent external implementations
	isActionRef()
}

// actionRef is the internal implementation of ActionRef.
type actionRef struct {
	value Action
}

func (a actionRef) String() string {
	return string(a.value)
}

func (a actionRef) IsMutation() bool {
	// All currently defined actions are mutations
	return a.value == ActionCreate || a.value == ActionUpdate || a.value == ActionDelete
}

func (a actionRef) IsDestructive() bool {
	return a.value == ActionDelete
}

func (a actionRef) Equals(other ActionRef) bool {
	if otherRef, ok := other.(actionRef); ok {
		return a.value == otherRef.value
	}
	return false
}

func (a actionRef) isActionRef() {}

// newActionRef creates a new action reference from the internal Action.
func newActionRef(action Action) ActionRef {
	return actionRef{value: action}
}

// actionContext is the default implementation of ActionContext.
type actionContext struct{}

func (actionContext) Create() ActionRef { return newActionRef(ActionCreate) }
func (actionContext) Update() ActionRef { return newActionRef(ActionUpdate) }
func (actionContext) Delete() ActionRef { return newActionRef(ActionDelete) }

// NewActionContext creates a new action context for accessing action references.
func NewActionContext() ActionContext {
	return actionContext{}
}
