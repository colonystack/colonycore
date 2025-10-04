package core

import (
	"context"
	"time"

	"colonycore/pkg/domain"
	"colonycore/pkg/pluginapi"
)

func adaptPluginRule(rule pluginapi.Rule) Rule {
	if rule == nil {
		return nil
	}
	return pluginRuleAdapter{impl: rule}
}

type pluginRuleAdapter struct {
	impl pluginapi.Rule
}

func (a pluginRuleAdapter) Name() string { return a.impl.Name() }

func (a pluginRuleAdapter) Evaluate(ctx context.Context, view RuleView, changes []domain.Change) (domain.Result, error) {
	pluginView := adaptRuleView(view)
	pluginChanges := toPluginChanges(changes)
	res, err := a.impl.Evaluate(ctx, pluginView, pluginChanges)
	if err != nil {
		return domain.Result{}, err
	}
	return toDomainResult(res), nil
}

func adaptRuleView(view RuleView) pluginapi.RuleView {
	if view == nil {
		return nil
	}
	return ruleViewAdapter{view: view}
}

type ruleViewAdapter struct {
	view RuleView
}

func (a ruleViewAdapter) ListOrganisms() []pluginapi.OrganismView {
	return newOrganismViews(a.view.ListOrganisms())
}

func (a ruleViewAdapter) ListHousingUnits() []pluginapi.HousingUnitView {
	return newHousingUnitViews(a.view.ListHousingUnits())
}

func (a ruleViewAdapter) ListProtocols() []pluginapi.ProtocolView {
	return newProtocolViews(a.view.ListProtocols())
}

func (a ruleViewAdapter) FindOrganism(id string) (pluginapi.OrganismView, bool) {
	org, ok := a.view.FindOrganism(id)
	if !ok {
		return nil, false
	}
	return newOrganismView(org), true
}

func (a ruleViewAdapter) FindHousingUnit(id string) (pluginapi.HousingUnitView, bool) {
	unit, ok := a.view.FindHousingUnit(id)
	if !ok {
		return nil, false
	}
	return newHousingUnitView(unit), true
}

type baseView struct {
	id        string
	createdAt time.Time
	updatedAt time.Time
}

func newBaseView(base domain.Base) baseView {
	return baseView{
		id:        base.ID,
		createdAt: base.CreatedAt,
		updatedAt: base.UpdatedAt,
	}
}

func (b baseView) ID() string           { return b.id }
func (b baseView) CreatedAt() time.Time { return b.createdAt }
func (b baseView) UpdatedAt() time.Time { return b.updatedAt }

type organismView struct {
	baseView
	name       string
	species    string
	line       string
	stage      pluginapi.LifecycleStage
	cohortID   *string
	housingID  *string
	protocolID *string
	projectID  *string
	attributes map[string]any
}

func newOrganismView(org domain.Organism) organismView {
	return organismView{
		baseView:   newBaseView(org.Base),
		name:       org.Name,
		species:    org.Species,
		line:       org.Line,
		stage:      pluginapi.LifecycleStage(org.Stage),
		cohortID:   cloneOptionalString(org.CohortID),
		housingID:  cloneOptionalString(org.HousingID),
		protocolID: cloneOptionalString(org.ProtocolID),
		projectID:  cloneOptionalString(org.ProjectID),
		attributes: cloneAttributes(org.Attributes),
	}
}

func (o organismView) Name() string    { return o.name }
func (o organismView) Species() string { return o.species }
func (o organismView) Line() string    { return o.line }
func (o organismView) Stage() pluginapi.LifecycleStage {
	return o.stage
}
func (o organismView) CohortID() (string, bool) {
	return derefString(o.cohortID)
}
func (o organismView) HousingID() (string, bool) {
	return derefString(o.housingID)
}
func (o organismView) ProtocolID() (string, bool) {
	return derefString(o.protocolID)
}
func (o organismView) ProjectID() (string, bool) {
	return derefString(o.projectID)
}
func (o organismView) Attributes() map[string]any {
	return cloneAttributes(o.attributes)
}

type housingUnitView struct {
	baseView
	name        string
	facility    string
	capacity    int
	environment string
}

func newHousingUnitView(unit domain.HousingUnit) housingUnitView {
	return housingUnitView{
		baseView:    newBaseView(unit.Base),
		name:        unit.Name,
		facility:    unit.Facility,
		capacity:    unit.Capacity,
		environment: unit.Environment,
	}
}

func (h housingUnitView) Name() string        { return h.name }
func (h housingUnitView) Facility() string    { return h.facility }
func (h housingUnitView) Capacity() int       { return h.capacity }
func (h housingUnitView) Environment() string { return h.environment }

type protocolView struct {
	baseView
	code        string
	title       string
	description string
	maxSubjects int
}

func newProtocolView(protocol domain.Protocol) protocolView {
	return protocolView{
		baseView:    newBaseView(protocol.Base),
		code:        protocol.Code,
		title:       protocol.Title,
		description: protocol.Description,
		maxSubjects: protocol.MaxSubjects,
	}
}

func (p protocolView) Code() string        { return p.code }
func (p protocolView) Title() string       { return p.title }
func (p protocolView) Description() string { return p.description }
func (p protocolView) MaxSubjects() int    { return p.maxSubjects }

func newOrganismViews(orgs []domain.Organism) []pluginapi.OrganismView {
	if len(orgs) == 0 {
		return nil
	}
	views := make([]pluginapi.OrganismView, len(orgs))
	for i, org := range orgs {
		ov := newOrganismView(org)
		views[i] = ov
	}
	return views
}

func newHousingUnitViews(units []domain.HousingUnit) []pluginapi.HousingUnitView {
	if len(units) == 0 {
		return nil
	}
	views := make([]pluginapi.HousingUnitView, len(units))
	for i, unit := range units {
		hv := newHousingUnitView(unit)
		views[i] = hv
	}
	return views
}

func newProtocolViews(protocols []domain.Protocol) []pluginapi.ProtocolView {
	if len(protocols) == 0 {
		return nil
	}
	views := make([]pluginapi.ProtocolView, len(protocols))
	for i, protocol := range protocols {
		pv := newProtocolView(protocol)
		views[i] = pv
	}
	return views
}

func toPluginChanges(changes []domain.Change) []pluginapi.Change {
	if len(changes) == 0 {
		return nil
	}
	converted := make([]pluginapi.Change, len(changes))
	for i, change := range changes {
		converted[i] = pluginapi.NewChange(
			pluginapi.EntityType(change.Entity),
			pluginapi.Action(change.Action),
			change.Before,
			change.After,
		)
	}
	return converted
}

func toDomainResult(res pluginapi.Result) domain.Result {
	pvs := res.Violations()
	if len(pvs) == 0 {
		return domain.Result{}
	}
	violations := make([]domain.Violation, len(pvs))
	for i, violation := range pvs {
		violations[i] = domain.Violation{
			Rule:     violation.Rule(),
			Severity: domain.Severity(violation.Severity()),
			Message:  violation.Message(),
			Entity:   domain.EntityType(violation.Entity()),
			EntityID: violation.EntityID(),
		}
	}
	return domain.Result{Violations: violations}
}

func cloneOptionalString(ptr *string) *string {
	if ptr == nil {
		return nil
	}
	value := *ptr
	return &value
}

func derefString(ptr *string) (string, bool) {
	if ptr == nil {
		return "", false
	}
	return *ptr, true
}

func cloneAttributes(attrs map[string]any) map[string]any {
	if len(attrs) == 0 {
		return nil
	}
	out := make(map[string]any, len(attrs))
	for k, v := range attrs {
		out[k] = deepCloneAttribute(v)
	}
	return out
}

// deepCloneAttribute performs a best-effort recursive clone of common container
// shapes used in organism Attributes to harden immutability of projections:
//   - map[string]any
//   - []any
//   - []string
//   - []map[string]any
// Primitive values and unrecognized types are returned as-is. Cycles are not
// supported (the domain model is expected to be acyclic for attributes).
func deepCloneAttribute(v any) any {
	switch tv := v.(type) {
	case map[string]any:
		if len(tv) == 0 {
			return map[string]any{}
		}
		m := make(map[string]any, len(tv))
		for k, vv := range tv {
			m[k] = deepCloneAttribute(vv)
		}
		return m
	case []any:
		if len(tv) == 0 {
			return []any{}
		}
		s := make([]any, len(tv))
		for i, vv := range tv {
			s[i] = deepCloneAttribute(vv)
		}
		return s
	case []string:
		if len(tv) == 0 {
			return []string{}
		}
		s := make([]string, len(tv))
		copy(s, tv)
		return s
	case []map[string]any:
		if len(tv) == 0 {
			return []map[string]any{}
		}
		s := make([]map[string]any, len(tv))
		for i, mv := range tv {
			if mv == nil {
				continue
			}
			s[i] = cloneAttributes(mv)
		}
		return s
	default:
		return v
	}
}
