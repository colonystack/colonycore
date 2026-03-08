package conformance

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"colonycore/pkg/datasetapi"
	"colonycore/pkg/pluginapi"
	"colonycore/plugins/frog"
)

type pluginCase struct {
	name                           string
	newPlugin                      func() pluginapi.Plugin
	expectedSchemas                []string
	expectedRules                  []string
	minDatasetTemplates            int
	expectsDatasetRegistrationFail bool
	ruleScenarios                  []ruleScenario
}

type ruleScenario struct {
	name               string
	ruleName           string
	view               pluginapi.RuleView
	changes            []pluginapi.Change
	wantErr            bool
	wantViolationCount int
	assertViolation    func(*testing.T, pluginapi.Violation)
}

func TestPluginConformance(t *testing.T) {
	cases := []pluginCase{
		frogPluginCase(t),
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			registered := registerPlugin(t, tc)
			assertInitialization(t, tc, registered)
			assertRegistrationSurface(t, tc, registered)
			assertRuleHooks(t, tc, registered)
			assertRegistrationErrors(t, tc)
		})
	}
}

func registerPlugin(t *testing.T, tc pluginCase) *capturingRegistry {
	t.Helper()
	plugin := tc.newPlugin()
	registry := newCapturingRegistry()
	if err := plugin.Register(registry); err != nil {
		t.Fatalf("register plugin %q: %v", tc.name, err)
	}
	if registry.HasErrors() {
		t.Fatalf("plugin %q made invalid registration calls: %s", tc.name, strings.Join(registry.Errors(), "; "))
	}
	return registry
}

func assertInitialization(t *testing.T, tc pluginCase, registry *capturingRegistry) {
	t.Helper()
	plugin := tc.newPlugin()
	if plugin.Name() == "" {
		t.Fatalf("plugin name must not be empty")
	}
	if plugin.Version() == "" {
		t.Fatalf("plugin version must not be empty")
	}
	if len(registry.rules) == 0 {
		t.Fatalf("plugin %q must register at least one rule", tc.name)
	}
	provider := pluginapi.GetVersionProvider()
	if provider == nil {
		t.Fatalf("plugin API version provider must be available")
	}
	apiVersion := provider.APIVersion()
	if extractMajorVersion(apiVersion) != "v1" {
		t.Fatalf("plugin API major version must be v1, got %q", apiVersion)
	}
}

func assertRegistrationSurface(t *testing.T, tc pluginCase, registry *capturingRegistry) {
	t.Helper()

	for _, entity := range tc.expectedSchemas {
		schema, ok := registry.schemas[entity]
		if !ok {
			t.Fatalf("expected schema registration for entity %q", entity)
		}
		if len(schema) == 0 {
			t.Fatalf("schema for entity %q must not be empty", entity)
		}
	}

	ruleNames := make([]string, 0, len(registry.rules))
	seenRuleNames := make(map[string]struct{}, len(registry.rules))
	for _, rule := range registry.rules {
		ruleName := rule.Name()
		if ruleName == "" {
			t.Fatalf("registered rule has empty name: %q", ruleName)
		}
		if _, exists := seenRuleNames[ruleName]; exists {
			t.Fatalf("registered rule has duplicate name: %q", ruleName)
		}
		seenRuleNames[ruleName] = struct{}{}
		ruleNames = append(ruleNames, ruleName)
	}
	for _, name := range tc.expectedRules {
		if !containsString(ruleNames, name) {
			t.Fatalf("expected rule %q, got %v", name, ruleNames)
		}
	}

	if len(registry.templates) < tc.minDatasetTemplates {
		t.Fatalf("expected at least %d dataset templates, got %d", tc.minDatasetTemplates, len(registry.templates))
	}
}

func assertRuleHooks(t *testing.T, tc pluginCase, registry *capturingRegistry) {
	t.Helper()

	rulesByName := make(map[string]pluginapi.Rule, len(registry.rules))
	for _, rule := range registry.rules {
		rulesByName[rule.Name()] = rule
	}

	for _, scenario := range tc.ruleScenarios {
		t.Run(fmt.Sprintf("rule-hook/%s", scenario.name), func(t *testing.T) {
			rule, ok := rulesByName[scenario.ruleName]
			if !ok {
				t.Fatalf("rule %q not registered", scenario.ruleName)
			}

			result, err := rule.Evaluate(context.Background(), scenario.view, scenario.changes)
			if scenario.wantErr {
				if err == nil {
					t.Fatalf("expected error from rule %q", scenario.ruleName)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error from rule %q: %v", scenario.ruleName, err)
			}

			violations := result.Violations()
			if len(violations) != scenario.wantViolationCount {
				t.Fatalf("expected %d violations, got %d", scenario.wantViolationCount, len(violations))
			}
			if scenario.assertViolation != nil && len(violations) > 0 {
				scenario.assertViolation(t, violations[0])
			}
		})
	}
}

func assertRegistrationErrors(t *testing.T, tc pluginCase) {
	t.Helper()
	if !tc.expectsDatasetRegistrationFail {
		return
	}

	plugin := tc.newPlugin()
	errInjected := errors.New("forced dataset registration failure")
	registry := newCapturingRegistry()
	registry.templateErr = errInjected

	err := plugin.Register(registry)
	if !errors.Is(err, errInjected) {
		t.Fatalf("expected dataset registration error %q, got %v", errInjected, err)
	}
	if registry.HasErrors() {
		t.Fatalf("plugin %q made invalid registration calls: %s", tc.name, strings.Join(registry.Errors(), "; "))
	}
}

func extractMajorVersion(version string) string {
	version = strings.TrimSpace(version)
	if version == "" {
		return ""
	}
	for _, separator := range []string{".", "-", "+"} {
		if idx := strings.Index(version, separator); idx >= 0 {
			version = version[:idx]
		}
	}
	return version
}

func frogPluginCase(t *testing.T) pluginCase {
	t.Helper()

	changes := []pluginapi.Change{mustChange(t)}

	dryHousingID := "housing-dry"
	humidHousingID := "housing-humid"
	stages := pluginapi.NewLifecycleStageContext()
	createdAt := time.Unix(0, 0).UTC()

	warnView := staticRuleView{
		organisms: []pluginapi.OrganismView{
			organismFixture{
				id:        "frog-risk",
				name:      "Frog Risk",
				species:   "Tree Frog",
				stage:     stages.Adult(),
				housingID: stringPointer(dryHousingID),
				createdAt: createdAt,
				updatedAt: createdAt.Add(time.Second),
			},
		},
		housingUnits: []pluginapi.HousingUnitView{
			housingFixture{
				id:          dryHousingID,
				name:        "Dry Habitat",
				environment: "dry",
				createdAt:   createdAt,
				updatedAt:   createdAt,
			},
		},
	}

	safeView := staticRuleView{
		organisms: []pluginapi.OrganismView{
			organismFixture{
				id:        "frog-safe",
				name:      "Frog Safe",
				species:   "Tree Frog",
				stage:     stages.Adult(),
				housingID: stringPointer(humidHousingID),
				createdAt: createdAt,
				updatedAt: createdAt.Add(2 * time.Second),
			},
			organismFixture{
				id:        "gecko",
				name:      "Gecko",
				species:   "Gecko",
				stage:     stages.Adult(),
				housingID: stringPointer(humidHousingID),
				createdAt: createdAt,
				updatedAt: createdAt.Add(3 * time.Second),
			},
		},
		housingUnits: []pluginapi.HousingUnitView{
			housingFixture{
				id:          humidHousingID,
				name:        "Humid Habitat",
				environment: "humid",
				createdAt:   createdAt,
				updatedAt:   createdAt,
			},
		},
	}

	return pluginCase{
		name:                           "frog",
		newPlugin:                      func() pluginapi.Plugin { return frog.New() },
		expectedSchemas:                []string{"organism"},
		expectedRules:                  []string{"frog_habitat_warning"},
		minDatasetTemplates:            1,
		expectsDatasetRegistrationFail: true,
		ruleScenarios: []ruleScenario{
			{
				name:               "warns-on-non-aquatic-or-non-humid-housing",
				ruleName:           "frog_habitat_warning",
				view:               warnView,
				changes:            changes,
				wantViolationCount: 1,
				assertViolation: func(t *testing.T, violation pluginapi.Violation) {
					t.Helper()
					if violation.Rule() != "frog_habitat_warning" {
						t.Fatalf("expected violation rule frog_habitat_warning, got %q", violation.Rule())
					}
					if violation.EntityID() != "frog-risk" {
						t.Fatalf("expected violation for frog-risk, got %q", violation.EntityID())
					}
					warn := pluginapi.NewSeverityContext().Warn()
					if violation.Severity() != pluginapi.Severity(warn.String()) {
						t.Fatalf("expected warning severity %q, got %q", warn.String(), violation.Severity())
					}
				},
			},
			{
				name:               "ignores-safe-or-non-frog-organisms",
				ruleName:           "frog_habitat_warning",
				view:               safeView,
				changes:            changes,
				wantViolationCount: 0,
			},
		},
	}
}

func mustChange(t *testing.T) pluginapi.Change {
	t.Helper()
	entities := pluginapi.NewEntityContext()
	actions := pluginapi.NewActionContext()
	change, err := pluginapi.NewChangeBuilder().
		WithEntity(entities.Organism()).
		WithAction(actions.Update()).
		Build()
	if err != nil {
		t.Fatalf("build change fixture: %v", err)
	}
	return change
}

func stringPointer(v string) *string {
	return &v
}

type capturingRegistry struct {
	schemas      map[string]map[string]any
	rules        []pluginapi.Rule
	templates    []datasetapi.Template
	templateErr  error
	invalidCalls []string
}

func newCapturingRegistry() *capturingRegistry {
	return &capturingRegistry{schemas: make(map[string]map[string]any)}
}

func (r *capturingRegistry) recordInvalidCall(message string) {
	r.invalidCalls = append(r.invalidCalls, message)
}

func (r *capturingRegistry) HasErrors() bool {
	return len(r.invalidCalls) > 0
}

func (r *capturingRegistry) Errors() []string {
	if len(r.invalidCalls) == 0 {
		return nil
	}
	out := make([]string, len(r.invalidCalls))
	copy(out, r.invalidCalls)
	return out
}

func (r *capturingRegistry) RegisterSchema(entity string, schema map[string]any) {
	if entity == "" || schema == nil {
		r.recordInvalidCall(fmt.Sprintf("RegisterSchema(entity=%q): entity must be non-empty and schema must be non-nil", entity))
		return
	}
	cloned := make(map[string]any, len(schema))
	for key, value := range schema {
		cloned[key] = value
	}
	r.schemas[entity] = cloned
}

func (r *capturingRegistry) RegisterRule(rule pluginapi.Rule) {
	if rule == nil {
		r.recordInvalidCall("RegisterRule(rule=nil): rule must be non-nil")
		return
	}
	r.rules = append(r.rules, rule)
}

func (r *capturingRegistry) RegisterDatasetTemplate(template datasetapi.Template) error {
	if r.templateErr != nil {
		return r.templateErr
	}
	r.templates = append(r.templates, template)
	return nil
}

type organismFixture struct {
	id         string
	name       string
	species    string
	line       string
	lineID     *string
	strainID   *string
	parentIDs  []string
	cohortID   *string
	housingID  *string
	protocolID *string
	projectID  *string
	attributes map[string]any
	stage      pluginapi.LifecycleStageRef
	createdAt  time.Time
	updatedAt  time.Time
}

func (o organismFixture) ID() string           { return o.id }
func (o organismFixture) CreatedAt() time.Time { return o.createdAt }
func (o organismFixture) UpdatedAt() time.Time { return o.updatedAt }
func (o organismFixture) Name() string         { return o.name }
func (o organismFixture) Species() string      { return o.species }
func (o organismFixture) Line() string         { return o.line }

func (o organismFixture) LineID() (string, bool) {
	return optionalString(o.lineID)
}

func (o organismFixture) StrainID() (string, bool) {
	return optionalString(o.strainID)
}

func (o organismFixture) ParentIDs() []string {
	return append([]string(nil), o.parentIDs...)
}

func (o organismFixture) Stage() pluginapi.LifecycleStage {
	return pluginapi.LifecycleStage(o.GetCurrentStage().String())
}

func (o organismFixture) CohortID() (string, bool) {
	return optionalString(o.cohortID)
}

func (o organismFixture) HousingID() (string, bool) {
	return optionalString(o.housingID)
}

func (o organismFixture) ProtocolID() (string, bool) {
	return optionalString(o.protocolID)
}

func (o organismFixture) ProjectID() (string, bool) {
	return optionalString(o.projectID)
}

func (o organismFixture) Attributes() map[string]any {
	return deepCloneAnyMap(o.attributes)
}

func (organismFixture) Extensions() pluginapi.ExtensionSet {
	return pluginapi.NewExtensionSet(nil)
}

func (o organismFixture) CoreAttributes() map[string]any {
	return deepCloneAnyMap(o.attributes)
}

func (o organismFixture) CoreAttributesPayload() pluginapi.ObjectPayload {
	return pluginapi.NewObjectPayload(o.CoreAttributes())
}

func (o organismFixture) GetCurrentStage() pluginapi.LifecycleStageRef {
	if o.stage != nil {
		return o.stage
	}
	return pluginapi.NewLifecycleStageContext().Planned()
}

func (o organismFixture) IsActive() bool {
	return o.GetCurrentStage().IsActive()
}

func (o organismFixture) IsRetired() bool {
	return o.GetCurrentStage().Equals(pluginapi.NewLifecycleStageContext().Retired())
}

func (o organismFixture) IsDeceased() bool {
	return o.GetCurrentStage().Equals(pluginapi.NewLifecycleStageContext().Deceased())
}

type housingFixture struct {
	id          string
	name        string
	facilityID  string
	capacity    int
	environment string
	state       string
	createdAt   time.Time
	updatedAt   time.Time
}

func (h housingFixture) ID() string           { return h.id }
func (h housingFixture) CreatedAt() time.Time { return h.createdAt }
func (h housingFixture) UpdatedAt() time.Time { return h.updatedAt }
func (h housingFixture) Name() string         { return h.name }
func (h housingFixture) FacilityID() string   { return h.facilityID }
func (h housingFixture) Capacity() int        { return h.capacity }

func (h housingFixture) Environment() string {
	return h.GetEnvironmentType().String()
}

func (h housingFixture) State() string {
	return h.GetCurrentState().String()
}

func (h housingFixture) GetEnvironmentType() pluginapi.EnvironmentTypeRef {
	ctx := pluginapi.NewHousingContext()
	switch strings.ToLower(strings.TrimSpace(h.environment)) {
	case ctx.Aquatic().String():
		return ctx.Aquatic()
	case ctx.Humid().String():
		return ctx.Humid()
	case ctx.Arboreal().String():
		return ctx.Arboreal()
	default:
		return ctx.Terrestrial()
	}
}

func (h housingFixture) IsAquaticEnvironment() bool {
	return h.GetEnvironmentType().IsAquatic()
}

func (h housingFixture) IsHumidEnvironment() bool {
	return h.GetEnvironmentType().IsHumid()
}

func (h housingFixture) SupportsSpecies(species string) bool {
	if strings.Contains(strings.ToLower(species), "frog") {
		return h.IsAquaticEnvironment() || h.IsHumidEnvironment()
	}
	return true
}

func (h housingFixture) GetCurrentState() pluginapi.HousingStateRef {
	ctx := pluginapi.NewHousingStateContext()
	switch strings.ToLower(strings.TrimSpace(h.state)) {
	case ctx.Quarantine().String():
		return ctx.Quarantine()
	case ctx.Cleaning().String():
		return ctx.Cleaning()
	case ctx.Decommissioned().String():
		return ctx.Decommissioned()
	default:
		return ctx.Active()
	}
}

func (h housingFixture) IsActiveState() bool {
	return h.GetCurrentState().IsActive()
}

func (h housingFixture) IsDecommissioned() bool {
	return h.GetCurrentState().IsDecommissioned()
}

type staticRuleView struct {
	organisms    []pluginapi.OrganismView
	housingUnits []pluginapi.HousingUnitView
}

func (v staticRuleView) ListOrganisms() []pluginapi.OrganismView {
	if len(v.organisms) == 0 {
		return nil
	}
	out := make([]pluginapi.OrganismView, len(v.organisms))
	copy(out, v.organisms)
	return out
}

func (v staticRuleView) ListHousingUnits() []pluginapi.HousingUnitView {
	if len(v.housingUnits) == 0 {
		return nil
	}
	out := make([]pluginapi.HousingUnitView, len(v.housingUnits))
	copy(out, v.housingUnits)
	return out
}

func (staticRuleView) ListFacilities() []pluginapi.FacilityView      { return nil }
func (staticRuleView) ListTreatments() []pluginapi.TreatmentView     { return nil }
func (staticRuleView) ListObservations() []pluginapi.ObservationView { return nil }
func (staticRuleView) ListSamples() []pluginapi.SampleView           { return nil }
func (staticRuleView) ListProtocols() []pluginapi.ProtocolView       { return nil }
func (staticRuleView) ListPermits() []pluginapi.PermitView           { return nil }
func (staticRuleView) ListProjects() []pluginapi.ProjectView         { return nil }
func (staticRuleView) ListSupplyItems() []pluginapi.SupplyItemView   { return nil }

func (v staticRuleView) FindOrganism(id string) (pluginapi.OrganismView, bool) {
	for _, organism := range v.organisms {
		if organism.ID() == id {
			return organism, true
		}
	}
	return nil, false
}

func (v staticRuleView) FindHousingUnit(id string) (pluginapi.HousingUnitView, bool) {
	for _, housing := range v.housingUnits {
		if housing.ID() == id {
			return housing, true
		}
	}
	return nil, false
}

func (staticRuleView) FindFacility(string) (pluginapi.FacilityView, bool)       { return nil, false }
func (staticRuleView) FindTreatment(string) (pluginapi.TreatmentView, bool)     { return nil, false }
func (staticRuleView) FindObservation(string) (pluginapi.ObservationView, bool) { return nil, false }
func (staticRuleView) FindSample(string) (pluginapi.SampleView, bool)           { return nil, false }
func (staticRuleView) FindPermit(string) (pluginapi.PermitView, bool)           { return nil, false }
func (staticRuleView) FindSupplyItem(string) (pluginapi.SupplyItemView, bool)   { return nil, false }

func optionalString(value *string) (string, bool) {
	if value == nil {
		return "", false
	}
	return *value, true
}

func deepCloneAnyMap(input map[string]any) map[string]any {
	if len(input) == 0 {
		return nil
	}
	cloned := make(map[string]any, len(input))
	for key, value := range input {
		cloned[key] = deepCloneAnyValue(value)
	}
	return cloned
}

func deepCloneAnyValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return deepCloneAnyMap(typed)
	case []any:
		if len(typed) == 0 {
			return []any{}
		}
		cloned := make([]any, len(typed))
		for index, item := range typed {
			cloned[index] = deepCloneAnyValue(item)
		}
		return cloned
	case []string:
		if len(typed) == 0 {
			return []string{}
		}
		cloned := make([]string, len(typed))
		copy(cloned, typed)
		return cloned
	case []map[string]any:
		if len(typed) == 0 {
			return []map[string]any{}
		}
		cloned := make([]map[string]any, len(typed))
		for index, item := range typed {
			cloned[index] = deepCloneAnyMap(item)
		}
		return cloned
	default:
		return value
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
