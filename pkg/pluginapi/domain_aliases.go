// Package pluginapi provides a stable surface for plugin authors by defining
// rule evaluation primitives and canonical identifiers independent of the core
// domain package.
package pluginapi

import "errors"

// Severity captures rule outcomes exposed to plugins.
type Severity string

// Rule evaluation severities determine commit behavior and logging.
// These are now internal - use SeverityContext for plugin access.
const (
	severityBlock Severity = "block"
	severityWarn  Severity = "warn"
	severityLog   Severity = "log"
)

// LifecycleStage represents canonical organism lifecycle identifiers available to plugins.
type LifecycleStage string

// EntityType identifies the type of record referenced by rule changes.
type EntityType string

// Supported entity type identifiers used in Change records and persistence buckets.
// These are now internal - use EntityContext for plugin access.
const (
	entityOrganism    EntityType = "organism"
	entityCohort      EntityType = "cohort"
	entityHousingUnit EntityType = "housing_unit"
	entityFacility    EntityType = "facility"
	entityBreeding    EntityType = "breeding_unit"
	entityProcedure   EntityType = "procedure"
	entityTreatment   EntityType = "treatment"
	entityObservation EntityType = "observation"
	entitySample      EntityType = "sample"
	entityProtocol    EntityType = "protocol"
	entityProject     EntityType = "project"
	entityPermit      EntityType = "permit"
	entitySupplyItem  EntityType = "supply_item"
)

// Action indicates the type of modification performed.
type Action string

// Change actions enumerate supported CRUD operations captured in audit trail.
// These are now internal - use ActionContext for plugin access.
const (
	actionCreate Action = "create"
	actionUpdate Action = "update"
	actionDelete Action = "delete"
)

// Change describes a mutation applied to an entity during a transaction. It is
// immutable to plugin authors (fields unexported) and accessed via getter
// methods to prevent accidental mutation of underlying state. Before/After are
// JSON snapshots of the prior and new values.
type Change struct {
	entity EntityType
	action Action
	before ChangePayload
	after  ChangePayload
}

// NewChange constructs an immutable Change for use by the host when adapting
// domain transactions for plugin rule evaluation.
func NewChange(entity EntityTypeRef, action ActionRef, before, after ChangePayload) Change {
	return Change{
		entity: entity.Value(),
		action: action.Value(),
		before: cloneChangePayload(before),
		after:  cloneChangePayload(after),
	}
}

// Entity returns the entity type affected.
func (c Change) Entity() EntityType { return c.entity }

// Action returns the CRUD action performed.
func (c Change) Action() Action { return c.action }

// Before returns the previous value (if any) of the entity. Treat as read-only.
func (c Change) Before() ChangePayload { return cloneChangePayload(c.before) }

// After returns the new value (if any) of the entity. Treat as read-only.
func (c Change) After() ChangePayload { return cloneChangePayload(c.after) }

// ChangeBuilder provides a fluent interface for building changes.
type ChangeBuilder struct {
	entity EntityTypeRef
	action ActionRef
	before ChangePayload
	after  ChangePayload
	built  bool
}

// NewChangeBuilder creates a new change builder.
func NewChangeBuilder() *ChangeBuilder {
	return &ChangeBuilder{}
}

// WithEntity sets the entity using contextual interface.
func (cb *ChangeBuilder) WithEntity(entity EntityTypeRef) *ChangeBuilder {
	if cb.built {
		panic("cannot modify builder after Build() is called")
	}
	cb.entity = entity
	return cb
}

// WithAction sets the action using contextual interface.
func (cb *ChangeBuilder) WithAction(action ActionRef) *ChangeBuilder {
	if cb.built {
		panic("cannot modify builder after Build() is called")
	}
	cb.action = action
	return cb
}

// WithBefore sets the before state.
func (cb *ChangeBuilder) WithBefore(before ChangePayload) *ChangeBuilder {
	if cb.built {
		panic("cannot modify builder after Build() is called")
	}
	cb.before = before
	return cb
}

// WithAfter sets the after state.
func (cb *ChangeBuilder) WithAfter(after ChangePayload) *ChangeBuilder {
	if cb.built {
		panic("cannot modify builder after Build() is called")
	}
	cb.after = after
	return cb
}

// Build creates the immutable Change.
func (cb *ChangeBuilder) Build() (Change, error) {
	if cb.built {
		return Change{}, errors.New("builder already used")
	}

	// Validation
	if cb.entity == nil {
		return Change{}, errors.New("entity is required")
	}
	if cb.action == nil {
		return Change{}, errors.New("action is required")
	}

	cb.built = true

	return Change{
		entity: cb.entity.Value(),
		action: cb.action.Value(),
		before: cloneChangePayload(cb.before),
		after:  cloneChangePayload(cb.after),
	}, nil
}

// Violation reports a failed rule evaluation.
type Violation struct {
	rule     string
	severity Severity
	message  string
	entity   EntityType
	entityID string
}

// NewViolation constructs an immutable violation using contextual interface references.
// This promotes hexagonal architecture by allowing violation creation without direct constant access.
// DEPRECATED: Use NewViolationBuilder() instead for better fluent interface and validation.
func NewViolation(rule string, severity SeverityRef, message string, entity EntityTypeRef, entityID string) Violation {
	// Extract underlying values from contextual interfaces
	rawSeverity := extractSeverity(severity)
	rawEntity := extractEntityType(entity)
	return Violation{rule: rule, severity: rawSeverity, message: message, entity: rawEntity, entityID: entityID}
}

// ConvertEntityType converts a raw EntityType to EntityTypeRef for core->plugin API conversion
func ConvertEntityType(et EntityType) EntityTypeRef {
	return newEntityTypeRef(et)
}

// ConvertAction converts a raw Action to ActionRef for core->plugin API conversion
func ConvertAction(a Action) ActionRef {
	return newActionRef(a)
}

// NewViolationWithEntityRef is an alias for NewViolation for backwards compatibility.
// DEPRECATED: Use NewViolationBuilder() instead for better fluent interface and validation.
func NewViolationWithEntityRef(rule string, severity SeverityRef, message string, entity EntityTypeRef, entityID string) Violation {
	return NewViolation(rule, severity, message, entity, entityID)
}

// ViolationBuilder provides a fluent interface for building violations.
type ViolationBuilder struct {
	rule     string
	severity SeverityRef
	message  string
	entity   EntityTypeRef
	entityID string
	built    bool
}

// NewViolationBuilder creates a new violation builder.
func NewViolationBuilder() *ViolationBuilder {
	return &ViolationBuilder{}
}

// WithRule sets the rule identifier.
func (vb *ViolationBuilder) WithRule(rule string) *ViolationBuilder {
	if vb.built {
		panic("cannot modify builder after Build() is called")
	}
	vb.rule = rule
	return vb
}

// WithSeverity sets the severity using contextual interface.
func (vb *ViolationBuilder) WithSeverity(severity SeverityRef) *ViolationBuilder {
	if vb.built {
		panic("cannot modify builder after Build() is called")
	}
	vb.severity = severity
	return vb
}

// WithMessage sets the violation message.
func (vb *ViolationBuilder) WithMessage(message string) *ViolationBuilder {
	if vb.built {
		panic("cannot modify builder after Build() is called")
	}
	vb.message = message
	return vb
}

// WithEntity sets the entity using contextual interface.
func (vb *ViolationBuilder) WithEntity(entity EntityTypeRef) *ViolationBuilder {
	if vb.built {
		panic("cannot modify builder after Build() is called")
	}
	vb.entity = entity
	return vb
}

// WithEntityID sets the entity identifier.
func (vb *ViolationBuilder) WithEntityID(entityID string) *ViolationBuilder {
	if vb.built {
		panic("cannot modify builder after Build() is called")
	}
	vb.entityID = entityID
	return vb
}

// BuildWarning creates a warning violation with the current builder state.
func (vb *ViolationBuilder) BuildWarning() (Violation, error) {
	severityCtx := NewSeverityContext()
	return vb.WithSeverity(severityCtx.Warn()).Build()
}

// BuildBlocking creates a blocking violation with the current builder state.
func (vb *ViolationBuilder) BuildBlocking() (Violation, error) {
	severityCtx := NewSeverityContext()
	return vb.WithSeverity(severityCtx.Block()).Build()
}

// BuildLog creates a log violation with the current builder state.
func (vb *ViolationBuilder) BuildLog() (Violation, error) {
	severityCtx := NewSeverityContext()
	return vb.WithSeverity(severityCtx.Log()).Build()
}

// Build creates the immutable Violation.
func (vb *ViolationBuilder) Build() (Violation, error) {
	if vb.built {
		return Violation{}, errors.New("builder already used")
	}

	// Validation
	if vb.rule == "" {
		return Violation{}, errors.New("rule is required")
	}
	if vb.severity == nil {
		return Violation{}, errors.New("severity is required")
	}
	if vb.entity == nil {
		return Violation{}, errors.New("entity is required")
	}

	vb.built = true

	// Extract underlying values from contextual interfaces
	rawSeverity := extractSeverity(vb.severity)
	rawEntity := extractEntityType(vb.entity)

	return Violation{
		rule:     vb.rule,
		severity: rawSeverity,
		message:  vb.message,
		entity:   rawEntity,
		entityID: vb.entityID,
	}, nil
}

// extractSeverity safely extracts the underlying Severity from a SeverityRef.
func extractSeverity(ref SeverityRef) Severity {
	if severityRef, ok := ref.(severityRef); ok {
		return severityRef.value
	}
	// Fallback for unexpected implementations - should not happen in normal usage
	return Severity(ref.String())
}

// extractEntityType safely extracts the underlying EntityType from an EntityTypeRef.
func extractEntityType(ref EntityTypeRef) EntityType {
	if entityRef, ok := ref.(entityTypeRef); ok {
		return entityRef.value
	}
	// Fallback for unexpected implementations - should not happen in normal usage
	return EntityType(ref.String())
}

// Rule returns the identifier of the rule that produced the violation.
func (v Violation) Rule() string { return v.rule }

// Severity returns the severity classification of the violation.
func (v Violation) Severity() Severity { return v.severity }

// Message returns the human-readable description of the violation.
func (v Violation) Message() string { return v.message }

// Entity returns the entity type associated with the violation.
func (v Violation) Entity() EntityType { return v.entity }

// EntityID returns the identifier of the entity instance associated with the violation.
func (v Violation) EntityID() string { return v.entityID }

// Result aggregates violations from the rules engine in an immutable slice.
type Result struct {
	violations []Violation
}

// NewResult constructs a Result from provided violations (defensively copied).
func NewResult(violations ...Violation) Result {
	if len(violations) == 0 {
		return Result{}
	}
	cp := make([]Violation, len(violations))
	copy(cp, violations)
	return Result{violations: cp}
}

// Violations returns a defensive copy of contained violations.
func (r Result) Violations() []Violation {
	if len(r.violations) == 0 {
		return nil
	}
	cp := make([]Violation, len(r.violations))
	copy(cp, r.violations)
	return cp
}

// AddViolation returns a new Result with the additional violation appended.
func (r Result) AddViolation(v Violation) Result {
	if r.violations == nil {
		return Result{violations: []Violation{v}}
	}
	out := make([]Violation, len(r.violations)+1)
	copy(out, r.violations)
	out[len(out)-1] = v
	return Result{violations: out}
}

// Merge returns a new Result containing violations from both results.
func (r Result) Merge(other Result) Result {
	ov := other.violations
	if len(ov) == 0 {
		return r
	}
	if len(r.violations) == 0 {
		return other
	}
	out := make([]Violation, 0, len(r.violations)+len(ov))
	out = append(out, r.violations...)
	out = append(out, ov...)
	return Result{violations: out}
}

// HasBlocking returns true if the result contains blocking violations.
func (r Result) HasBlocking() bool {
	for _, v := range r.violations {
		if v.severity == severityBlock {
			return true
		}
	}
	return false
}

// RuleViolationError is returned when blocking violations are present.
type RuleViolationError struct {
	Result Result
}

func (e RuleViolationError) Error() string {
	return "transaction blocked by rules"
}

// ResultBuilder provides a fluent interface for building results.
type ResultBuilder struct {
	violations []Violation
	built      bool
}

// NewResultBuilder creates a new result builder.
func NewResultBuilder() *ResultBuilder {
	return &ResultBuilder{
		violations: make([]Violation, 0),
	}
}

// AddViolation appends a violation to the result.
func (rb *ResultBuilder) AddViolation(violation Violation) *ResultBuilder {
	if rb.built {
		panic("cannot modify builder after Build() is called")
	}
	rb.violations = append(rb.violations, violation)
	return rb
}

// AddViolations appends multiple violations to the result.
func (rb *ResultBuilder) AddViolations(violations ...Violation) *ResultBuilder {
	if rb.built {
		panic("cannot modify builder after Build() is called")
	}
	rb.violations = append(rb.violations, violations...)
	return rb
}

// FromBuilder adds violations from a violation builder.
func (rb *ResultBuilder) FromBuilder(vb *ViolationBuilder) *ResultBuilder {
	if rb.built {
		panic("cannot modify builder after Build() is called")
	}
	violation, err := vb.Build()
	if err != nil {
		panic("violation builder error: " + err.Error())
	}
	rb.violations = append(rb.violations, violation)
	return rb
}

// MergeResult adds violations from an existing result.
func (rb *ResultBuilder) MergeResult(result Result) *ResultBuilder {
	if rb.built {
		panic("cannot modify builder after Build() is called")
	}
	rb.violations = append(rb.violations, result.Violations()...)
	return rb
}

// Build creates the immutable Result.
func (rb *ResultBuilder) Build() Result {
	if rb.built {
		panic("builder already used")
	}

	rb.built = true

	if len(rb.violations) == 0 {
		return Result{}
	}

	// Defensive copy
	cp := make([]Violation, len(rb.violations))
	copy(cp, rb.violations)

	return Result{violations: cp}
}

// Internal test helpers - DO NOT USE IN PRODUCTION CODE
// These are only for internal testing and will be removed in future versions

func newViolationForTest(rule string, severity Severity, message string, entity EntityType, entityID string) Violation {
	return Violation{
		rule:     rule,
		severity: severity,
		message:  message,
		entity:   entity,
		entityID: entityID,
	}
}

func newChangeForTest(entity EntityType, action Action, before, after ChangePayload) Change {
	return Change{
		entity: entity,
		action: action,
		before: cloneChangePayload(before),
		after:  cloneChangePayload(after),
	}
}
