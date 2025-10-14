package core

import (
	"context"
	"strings"
	"time"

	"colonycore/pkg/domain"
	"colonycore/pkg/pluginapi"
)

const (
	stageAdult = "adult"
)

func adaptPluginRule(rule pluginapi.Rule) domain.Rule {
	if rule == nil {
		return nil
	}
	return pluginRuleAdapter{impl: rule}
}

type pluginRuleAdapter struct {
	impl pluginapi.Rule
}

func (a pluginRuleAdapter) Name() string { return a.impl.Name() }

func (a pluginRuleAdapter) Evaluate(ctx context.Context, view domain.RuleView, changes []domain.Change) (domain.Result, error) {
	pluginView := adaptRuleView(view)
	pluginChanges := toPluginChanges(changes)
	res, err := a.impl.Evaluate(ctx, pluginView, pluginChanges)
	if err != nil {
		return domain.Result{}, err
	}
	return toDomainResult(res), nil
}

func adaptRuleView(view domain.RuleView) pluginapi.RuleView {
	if view == nil {
		return nil
	}
	return ruleViewAdapter{view: view}
}

type ruleViewAdapter struct {
	view domain.RuleView
}

func (a ruleViewAdapter) ListOrganisms() []pluginapi.OrganismView {
	return newOrganismViews(a.view.ListOrganisms())
}

func (a ruleViewAdapter) ListHousingUnits() []pluginapi.HousingUnitView {
	return newHousingUnitViews(a.view.ListHousingUnits())
}

func (a ruleViewAdapter) ListFacilities() []pluginapi.FacilityView {
	return newFacilityViews(a.view.ListFacilities())
}

func (a ruleViewAdapter) ListTreatments() []pluginapi.TreatmentView {
	return newTreatmentViews(a.view.ListTreatments())
}

func (a ruleViewAdapter) ListObservations() []pluginapi.ObservationView {
	return newObservationViews(a.view.ListObservations())
}

func (a ruleViewAdapter) ListSamples() []pluginapi.SampleView {
	return newSampleViews(a.view.ListSamples())
}

func (a ruleViewAdapter) ListProtocols() []pluginapi.ProtocolView {
	return newProtocolViews(a.view.ListProtocols())
}

func (a ruleViewAdapter) ListPermits() []pluginapi.PermitView {
	return newPermitViews(a.view.ListPermits())
}

func (a ruleViewAdapter) ListProjects() []pluginapi.ProjectView {
	return newProjectViews(a.view.ListProjects())
}

func (a ruleViewAdapter) ListSupplyItems() []pluginapi.SupplyItemView {
	return newSupplyItemViews(a.view.ListSupplyItems())
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

func (a ruleViewAdapter) FindFacility(id string) (pluginapi.FacilityView, bool) {
	facility, ok := a.view.FindFacility(id)
	if !ok {
		return nil, false
	}
	return newFacilityView(facility), true
}

func (a ruleViewAdapter) FindTreatment(id string) (pluginapi.TreatmentView, bool) {
	treatment, ok := a.view.FindTreatment(id)
	if !ok {
		return nil, false
	}
	return newTreatmentView(treatment), true
}

func (a ruleViewAdapter) FindObservation(id string) (pluginapi.ObservationView, bool) {
	observation, ok := a.view.FindObservation(id)
	if !ok {
		return nil, false
	}
	return newObservationView(observation), true
}

func (a ruleViewAdapter) FindSample(id string) (pluginapi.SampleView, bool) {
	sample, ok := a.view.FindSample(id)
	if !ok {
		return nil, false
	}
	return newSampleView(sample), true
}

func (a ruleViewAdapter) FindPermit(id string) (pluginapi.PermitView, bool) {
	permit, ok := a.view.FindPermit(id)
	if !ok {
		return nil, false
	}
	return newPermitView(permit), true
}

func (a ruleViewAdapter) FindSupplyItem(id string) (pluginapi.SupplyItemView, bool) {
	item, ok := a.view.FindSupplyItem(id)
	if !ok {
		return nil, false
	}
	return newSupplyItemView(item), true
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

// Contextual lifecycle stage accessors
func (o organismView) GetCurrentStage() pluginapi.LifecycleStageRef {
	stages := pluginapi.NewLifecycleStageContext()
	switch o.stage {
	case "planned":
		return stages.Planned()
	case "embryo_larva":
		return stages.Larva()
	case "juvenile":
		return stages.Juvenile()
	case stageAdult:
		return stages.Adult()
	case "retired":
		return stages.Retired()
	case "deceased":
		return stages.Deceased()
	default:
		// Fallback for unknown stages - should not happen in normal operation
		return stages.Adult()
	}
}

func (o organismView) IsActive() bool {
	return o.GetCurrentStage().IsActive()
}

func (o organismView) IsRetired() bool {
	return string(o.stage) == "retired"
}

func (o organismView) IsDeceased() bool {
	return string(o.stage) == "deceased"
}

type housingUnitView struct {
	baseView
	name        string
	facilityID  string
	capacity    int
	environment string
}

func newHousingUnitView(unit domain.HousingUnit) housingUnitView {
	return housingUnitView{
		baseView:    newBaseView(unit.Base),
		name:        unit.Name,
		facilityID:  unit.FacilityID,
		capacity:    unit.Capacity,
		environment: unit.Environment,
	}
}

func (h housingUnitView) Name() string        { return h.name }
func (h housingUnitView) FacilityID() string  { return h.facilityID }
func (h housingUnitView) Capacity() int       { return h.capacity }
func (h housingUnitView) Environment() string { return h.environment }

// Contextual environment accessors
func (h housingUnitView) GetEnvironmentType() pluginapi.EnvironmentTypeRef {
	ctx := pluginapi.NewHousingContext()
	switch h.environment {
	case "aquatic":
		return ctx.Aquatic()
	case "terrestrial":
		return ctx.Terrestrial()
	case "arboreal":
		return ctx.Arboreal()
	case "humid":
		return ctx.Humid()
	default:
		// Default to terrestrial for unknown environments
		return ctx.Terrestrial()
	}
}

func (h housingUnitView) IsAquaticEnvironment() bool {
	return h.GetEnvironmentType().IsAquatic()
}

func (h housingUnitView) IsHumidEnvironment() bool {
	return h.GetEnvironmentType().IsHumid()
}

func (h housingUnitView) SupportsSpecies(species string) bool {
	envType := h.GetEnvironmentType()

	// Basic species-environment compatibility logic
	if species == "frog" || species == "amphibian" {
		return envType.IsAquatic() || envType.IsHumid()
	}

	if species == "fish" {
		return envType.IsAquatic()
	}

	// Default: terrestrial animals can live in terrestrial environments
	return !envType.IsAquatic() || envType.String() == "terrestrial"
}

type facilityView struct {
	baseView
	name                 string
	zone                 string
	accessPolicy         string
	environmentBaselines map[string]any
	housingUnitIDs       []string
	projectIDs           []string
}

func newFacilityView(facility domain.Facility) facilityView {
	return facilityView{
		baseView:             newBaseView(facility.Base),
		name:                 facility.Name,
		zone:                 facility.Zone,
		accessPolicy:         facility.AccessPolicy,
		environmentBaselines: cloneAttributes(facility.EnvironmentBaselines),
		housingUnitIDs:       cloneStringSlice(facility.HousingUnitIDs),
		projectIDs:           cloneStringSlice(facility.ProjectIDs),
	}
}

func (f facilityView) Name() string         { return f.name }
func (f facilityView) Zone() string         { return f.zone }
func (f facilityView) AccessPolicy() string { return f.accessPolicy }
func (f facilityView) EnvironmentBaselines() map[string]any {
	return cloneAttributes(f.environmentBaselines)
}
func (f facilityView) HousingUnitIDs() []string { return cloneStringSlice(f.housingUnitIDs) }
func (f facilityView) ProjectIDs() []string     { return cloneStringSlice(f.projectIDs) }

// Contextual zone & access policy accessors
func (f facilityView) GetZone() pluginapi.FacilityZoneRef {
	ctx := pluginapi.NewFacilityContext().Zones()
	zone := strings.ToLower(strings.TrimSpace(f.zone))
	switch {
	case zone == "":
		return ctx.General()
	case strings.Contains(zone, "bio") || strings.Contains(zone, "bsl"):
		return ctx.Biosecure()
	case strings.Contains(zone, "quarantine") || strings.Contains(zone, "isolation"):
		return ctx.Quarantine()
	default:
		return ctx.General()
	}
}

func (f facilityView) GetAccessPolicy() pluginapi.FacilityAccessPolicyRef {
	ctx := pluginapi.NewFacilityContext().AccessPolicies()
	policy := strings.ToLower(strings.TrimSpace(f.accessPolicy))
	switch {
	case policy == "":
		return ctx.Open()
	case strings.Contains(policy, "restricted") || strings.Contains(policy, "secure"):
		return ctx.Restricted()
	case strings.Contains(policy, "staff"):
		return ctx.StaffOnly()
	default:
		return ctx.Open()
	}
}

func (f facilityView) SupportsHousingUnit(id string) bool {
	for _, housingID := range f.housingUnitIDs {
		if housingID == id {
			return true
		}
	}
	return false
}

type treatmentView struct {
	baseView
	name              string
	procedureID       string
	organismIDs       []string
	cohortIDs         []string
	dosagePlan        string
	administrationLog []string
	adverseEvents     []string
}

func newTreatmentView(treatment domain.Treatment) treatmentView {
	return treatmentView{
		baseView:          newBaseView(treatment.Base),
		name:              treatment.Name,
		procedureID:       treatment.ProcedureID,
		organismIDs:       cloneStringSlice(treatment.OrganismIDs),
		cohortIDs:         cloneStringSlice(treatment.CohortIDs),
		dosagePlan:        treatment.DosagePlan,
		administrationLog: cloneStringSlice(treatment.AdministrationLog),
		adverseEvents:     cloneStringSlice(treatment.AdverseEvents),
	}
}

func (t treatmentView) Name() string          { return t.name }
func (t treatmentView) ProcedureID() string   { return t.procedureID }
func (t treatmentView) OrganismIDs() []string { return cloneStringSlice(t.organismIDs) }
func (t treatmentView) CohortIDs() []string   { return cloneStringSlice(t.cohortIDs) }
func (t treatmentView) DosagePlan() string    { return t.dosagePlan }
func (t treatmentView) AdministrationLog() []string {
	return cloneStringSlice(t.administrationLog)
}
func (t treatmentView) AdverseEvents() []string {
	return cloneStringSlice(t.adverseEvents)
}

// Contextual workflow accessors
func (t treatmentView) GetCurrentStatus() pluginapi.TreatmentStatusRef {
	return treatmentStatusFromLogs(t.administrationLog, t.adverseEvents)
}

func (t treatmentView) IsCompleted() bool {
	status := t.GetCurrentStatus()
	return status.IsCompleted() || status.IsFlagged()
}

func (t treatmentView) HasAdverseEvents() bool {
	return len(t.adverseEvents) > 0
}

type observationView struct {
	baseView
	procedureID *string
	organismID  *string
	cohortID    *string
	recordedAt  time.Time
	observer    string
	data        map[string]any
	notes       string
}

func newObservationView(observation domain.Observation) observationView {
	return observationView{
		baseView:    newBaseView(observation.Base),
		procedureID: cloneOptionalString(observation.ProcedureID),
		organismID:  cloneOptionalString(observation.OrganismID),
		cohortID:    cloneOptionalString(observation.CohortID),
		recordedAt:  observation.RecordedAt,
		observer:    observation.Observer,
		data:        cloneAttributes(observation.Data),
		notes:       observation.Notes,
	}
}

func (o observationView) ProcedureID() (string, bool) {
	return derefString(o.procedureID)
}

func (o observationView) OrganismID() (string, bool) {
	return derefString(o.organismID)
}

func (o observationView) CohortID() (string, bool) {
	return derefString(o.cohortID)
}

func (o observationView) RecordedAt() time.Time { return o.recordedAt }
func (o observationView) Observer() string      { return o.observer }
func (o observationView) Data() map[string]any  { return cloneAttributes(o.data) }
func (o observationView) Notes() string         { return o.notes }

// Contextual data shape accessors
func (o observationView) GetDataShape() pluginapi.ObservationShapeRef {
	return observationShapeFromData(len(o.data) > 0, o.notes)
}

func (o observationView) HasStructuredPayload() bool {
	return o.GetDataShape().HasStructuredPayload()
}

func (o observationView) HasNarrativeNotes() bool {
	return o.GetDataShape().HasNarrativeNotes()
}

type sampleView struct {
	baseView
	identifier      string
	sourceType      string
	organismID      *string
	cohortID        *string
	facilityID      string
	collectedAt     time.Time
	status          string
	storageLocation string
	assayType       string
	chainOfCustody  []domain.SampleCustodyEvent
	attributes      map[string]any
}

func newSampleView(sample domain.Sample) sampleView {
	return sampleView{
		baseView:        newBaseView(sample.Base),
		identifier:      sample.Identifier,
		sourceType:      sample.SourceType,
		organismID:      cloneOptionalString(sample.OrganismID),
		cohortID:        cloneOptionalString(sample.CohortID),
		facilityID:      sample.FacilityID,
		collectedAt:     sample.CollectedAt,
		status:          sample.Status,
		storageLocation: sample.StorageLocation,
		assayType:       sample.AssayType,
		chainOfCustody:  cloneCustodyEvents(sample.ChainOfCustody),
		attributes:      cloneAttributes(sample.Attributes),
	}
}

func (s sampleView) Identifier() string      { return s.identifier }
func (s sampleView) SourceType() string      { return s.sourceType }
func (s sampleView) FacilityID() string      { return s.facilityID }
func (s sampleView) CollectedAt() time.Time  { return s.collectedAt }
func (s sampleView) Status() string          { return s.status }
func (s sampleView) StorageLocation() string { return s.storageLocation }
func (s sampleView) AssayType() string       { return s.assayType }
func (s sampleView) ChainOfCustody() []map[string]any {
	return cloneCustodyEventMaps(s.chainOfCustody)
}
func (s sampleView) Attributes() map[string]any {
	return cloneAttributes(s.attributes)
}

func (s sampleView) OrganismID() (string, bool) {
	return derefString(s.organismID)
}

func (s sampleView) CohortID() (string, bool) {
	return derefString(s.cohortID)
}

// Contextual sample accessors
func (s sampleView) GetSource() pluginapi.SampleSourceRef {
	ctx := pluginapi.NewSampleContext().Sources()
	source := strings.ToLower(strings.TrimSpace(s.sourceType))
	switch source {
	case "organism":
		return ctx.Organism()
	case "cohort":
		return ctx.Cohort()
	case "environment", "environmental":
		return ctx.Environmental()
	default:
		return ctx.Unknown()
	}
}

func (s sampleView) GetStatus() pluginapi.SampleStatusRef {
	ctx := pluginapi.NewSampleContext().Statuses()
	status := strings.ToLower(strings.TrimSpace(s.status))
	switch status {
	case "stored":
		return ctx.Stored()
	case "in_transit", "in-transit", "transit":
		return ctx.InTransit()
	case "consumed":
		return ctx.Consumed()
	case "disposed":
		return ctx.Disposed()
	default:
		return ctx.Stored()
	}
}

func (s sampleView) IsAvailable() bool {
	return s.GetStatus().IsAvailable()
}

type protocolView struct {
	baseView
	code        string
	title       string
	description string
	maxSubjects int
	status      string
}

func newProtocolView(protocol domain.Protocol) protocolView {
	return protocolView{
		baseView:    newBaseView(protocol.Base),
		code:        protocol.Code,
		title:       protocol.Title,
		description: protocol.Description,
		maxSubjects: protocol.MaxSubjects,
		status:      protocol.Status,
	}
}

func (p protocolView) Code() string        { return p.code }
func (p protocolView) Title() string       { return p.title }
func (p protocolView) Description() string { return p.description }
func (p protocolView) MaxSubjects() int    { return p.maxSubjects }

// Contextual status accessors
func (p protocolView) GetCurrentStatus() pluginapi.ProtocolStatusRef {
	ctx := pluginapi.NewProtocolContext()
	switch p.status {
	case "draft":
		return ctx.Draft()
	case "active":
		return ctx.Active()
	case "suspended":
		return ctx.Suspended()
	case "completed":
		return ctx.Completed()
	case "cancelled":
		return ctx.Cancelled()
	default:
		// Default to draft for unknown statuses
		return ctx.Draft()
	}
}

func (p protocolView) IsActiveProtocol() bool {
	return p.GetCurrentStatus().IsActive()
}

func (p protocolView) IsTerminalStatus() bool {
	return p.GetCurrentStatus().IsTerminal()
}

func (p protocolView) CanAcceptNewSubjects() bool {
	return p.GetCurrentStatus().IsActive() && p.maxSubjects > 0
}

type permitView struct {
	baseView
	permitNumber      string
	authority         string
	validFrom         time.Time
	validUntil        time.Time
	allowedActivities []string
	facilityIDs       []string
	protocolIDs       []string
	notes             string
}

func newPermitView(permit domain.Permit) permitView {
	return permitView{
		baseView:          newBaseView(permit.Base),
		permitNumber:      permit.PermitNumber,
		authority:         permit.Authority,
		validFrom:         permit.ValidFrom,
		validUntil:        permit.ValidUntil,
		allowedActivities: cloneStringSlice(permit.AllowedActivities),
		facilityIDs:       cloneStringSlice(permit.FacilityIDs),
		protocolIDs:       cloneStringSlice(permit.ProtocolIDs),
		notes:             permit.Notes,
	}
}

func (p permitView) PermitNumber() string  { return p.permitNumber }
func (p permitView) Authority() string     { return p.authority }
func (p permitView) ValidFrom() time.Time  { return p.validFrom }
func (p permitView) ValidUntil() time.Time { return p.validUntil }
func (p permitView) AllowedActivities() []string {
	return cloneStringSlice(p.allowedActivities)
}
func (p permitView) FacilityIDs() []string { return cloneStringSlice(p.facilityIDs) }
func (p permitView) ProtocolIDs() []string { return cloneStringSlice(p.protocolIDs) }
func (p permitView) Notes() string         { return p.notes }

// Contextual validity accessors
func (p permitView) GetStatus(reference time.Time) pluginapi.PermitStatusRef {
	statuses := pluginapi.NewPermitContext().Statuses()
	switch {
	case reference.Before(p.validFrom):
		return statuses.Pending()
	case !p.validUntil.IsZero() && reference.After(p.validUntil):
		return statuses.Expired()
	default:
		return statuses.Active()
	}
}

func (p permitView) IsActive(reference time.Time) bool {
	return p.GetStatus(reference).IsActive()
}

func (p permitView) IsExpired(reference time.Time) bool {
	return p.GetStatus(reference).IsExpired()
}

type projectView struct {
	baseView
	code        string
	title       string
	description string
	facilityIDs []string
}

func newProjectView(project domain.Project) projectView {
	return projectView{
		baseView:    newBaseView(project.Base),
		code:        project.Code,
		title:       project.Title,
		description: project.Description,
		facilityIDs: cloneStringSlice(project.FacilityIDs),
	}
}

func (p projectView) Code() string        { return p.code }
func (p projectView) Title() string       { return p.title }
func (p projectView) Description() string { return p.description }
func (p projectView) FacilityIDs() []string {
	return cloneStringSlice(p.facilityIDs)
}

type supplyItemView struct {
	baseView
	sku            string
	name           string
	description    string
	quantityOnHand int
	unit           string
	lotNumber      string
	expiresAt      *time.Time
	facilityIDs    []string
	projectIDs     []string
	reorderLevel   int
	attributes     map[string]any
}

func newSupplyItemView(item domain.SupplyItem) supplyItemView {
	return supplyItemView{
		baseView:       newBaseView(item.Base),
		sku:            item.SKU,
		name:           item.Name,
		description:    item.Description,
		quantityOnHand: item.QuantityOnHand,
		unit:           item.Unit,
		lotNumber:      item.LotNumber,
		expiresAt:      cloneTimePtr(item.ExpiresAt),
		facilityIDs:    cloneStringSlice(item.FacilityIDs),
		projectIDs:     cloneStringSlice(item.ProjectIDs),
		reorderLevel:   item.ReorderLevel,
		attributes:     cloneAttributes(item.Attributes),
	}
}

func (s supplyItemView) SKU() string         { return s.sku }
func (s supplyItemView) Name() string        { return s.name }
func (s supplyItemView) Description() string { return s.description }
func (s supplyItemView) QuantityOnHand() int { return s.quantityOnHand }
func (s supplyItemView) Unit() string        { return s.unit }
func (s supplyItemView) LotNumber() string   { return s.lotNumber }
func (s supplyItemView) FacilityIDs() []string {
	return cloneStringSlice(s.facilityIDs)
}
func (s supplyItemView) ProjectIDs() []string {
	return cloneStringSlice(s.projectIDs)
}
func (s supplyItemView) ReorderLevel() int { return s.reorderLevel }
func (s supplyItemView) Attributes() map[string]any {
	return cloneAttributes(s.attributes)
}

func (s supplyItemView) ExpiresAt() (*time.Time, bool) {
	return derefTime(s.expiresAt)
}

// Contextual inventory accessors
func (s supplyItemView) GetInventoryStatus(reference time.Time) pluginapi.SupplyStatusRef {
	return supplyStatusFromInventory(s.quantityOnHand, s.reorderLevel, s.expiresAt, reference)
}

func (s supplyItemView) RequiresReorder(reference time.Time) bool {
	return s.GetInventoryStatus(reference).RequiresReorder()
}

func (s supplyItemView) IsExpired(reference time.Time) bool {
	return s.GetInventoryStatus(reference).IsExpired()
}

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

func newFacilityViews(facilities []domain.Facility) []pluginapi.FacilityView {
	if len(facilities) == 0 {
		return nil
	}
	views := make([]pluginapi.FacilityView, len(facilities))
	for i, facility := range facilities {
		views[i] = newFacilityView(facility)
	}
	return views
}

func newTreatmentViews(treatments []domain.Treatment) []pluginapi.TreatmentView {
	if len(treatments) == 0 {
		return nil
	}
	views := make([]pluginapi.TreatmentView, len(treatments))
	for i, treatment := range treatments {
		views[i] = newTreatmentView(treatment)
	}
	return views
}

func newObservationViews(observations []domain.Observation) []pluginapi.ObservationView {
	if len(observations) == 0 {
		return nil
	}
	views := make([]pluginapi.ObservationView, len(observations))
	for i, observation := range observations {
		views[i] = newObservationView(observation)
	}
	return views
}

func newSampleViews(samples []domain.Sample) []pluginapi.SampleView {
	if len(samples) == 0 {
		return nil
	}
	views := make([]pluginapi.SampleView, len(samples))
	for i, sample := range samples {
		views[i] = newSampleView(sample)
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

func newPermitViews(permits []domain.Permit) []pluginapi.PermitView {
	if len(permits) == 0 {
		return nil
	}
	views := make([]pluginapi.PermitView, len(permits))
	for i, permit := range permits {
		views[i] = newPermitView(permit)
	}
	return views
}

func newProjectViews(projects []domain.Project) []pluginapi.ProjectView {
	if len(projects) == 0 {
		return nil
	}
	views := make([]pluginapi.ProjectView, len(projects))
	for i, project := range projects {
		views[i] = newProjectView(project)
	}
	return views
}

func newSupplyItemViews(items []domain.SupplyItem) []pluginapi.SupplyItemView {
	if len(items) == 0 {
		return nil
	}
	views := make([]pluginapi.SupplyItemView, len(items))
	for i, item := range items {
		views[i] = newSupplyItemView(item)
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
			pluginapi.ConvertEntityType(pluginapi.EntityType(change.Entity)),
			pluginapi.ConvertAction(pluginapi.Action(change.Action)),
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

func cloneStringSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, len(values))
	copy(out, values)
	return out
}

func cloneCustodyEvents(events []domain.SampleCustodyEvent) []domain.SampleCustodyEvent {
	if len(events) == 0 {
		return nil
	}
	out := make([]domain.SampleCustodyEvent, len(events))
	for i, event := range events {
		out[i] = domain.SampleCustodyEvent{
			Actor:     event.Actor,
			Location:  event.Location,
			Timestamp: event.Timestamp,
			Notes:     event.Notes,
		}
	}
	return out
}

func cloneCustodyEventMaps(events []domain.SampleCustodyEvent) []map[string]any {
	if len(events) == 0 {
		return nil
	}
	out := make([]map[string]any, len(events))
	for i, event := range events {
		out[i] = map[string]any{
			"actor":     event.Actor,
			"location":  event.Location,
			"timestamp": event.Timestamp,
			"notes":     event.Notes,
		}
	}
	return out
}

func cloneTimePtr(src *time.Time) *time.Time {
	if src == nil {
		return nil
	}
	value := *src
	return &value
}

func derefTime(src *time.Time) (*time.Time, bool) {
	if src == nil {
		return nil, false
	}
	value := *src
	return &value, true
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

func treatmentStatusFromLogs(administrationLog, adverseEvents []string) pluginapi.TreatmentStatusRef {
	statuses := pluginapi.NewTreatmentContext().Statuses()
	switch {
	case len(administrationLog) == 0:
		return statuses.Planned()
	case len(adverseEvents) == 0:
		return statuses.Completed()
	default:
		return statuses.Flagged()
	}
}

func observationShapeFromData(hasStructured bool, notes string) pluginapi.ObservationShapeRef {
	shapes := pluginapi.NewObservationContext().Shapes()
	trimmedNotes := strings.TrimSpace(notes)
	switch {
	case hasStructured && trimmedNotes != "":
		return shapes.Mixed()
	case hasStructured:
		return shapes.Structured()
	default:
		return shapes.Narrative()
	}
}

func supplyStatusFromInventory(quantity, reorderLevel int, expiresAt *time.Time, now time.Time) pluginapi.SupplyStatusRef {
	statuses := pluginapi.NewSupplyContext().Statuses()
	if expiresAt != nil && !expiresAt.IsZero() && expiresAt.Before(now) {
		return statuses.Expired()
	}
	switch {
	case quantity <= 0:
		return statuses.Critical()
	case reorderLevel > 0 && quantity <= reorderLevel:
		return statuses.Reorder()
	default:
		return statuses.Healthy()
	}
}

// deepCloneAttribute performs a best-effort recursive clone of common container
// shapes used in organism Attributes to harden immutability of projections:
//   - map[string]any
//   - []any
//   - []string
//   - []map[string]any
//
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
