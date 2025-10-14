package pluginapi

import "context"

// Rule defines a plugin-contributed validation routine executed within a transaction.
type Rule interface {
	Name() string
	Evaluate(ctx context.Context, view RuleView, changes []Change) (Result, error)
}

// RuleView exposes read-only access to core entities during rule evaluation.
type RuleView interface {
	ListOrganisms() []OrganismView
	ListHousingUnits() []HousingUnitView
	ListFacilities() []FacilityView
	ListTreatments() []TreatmentView
	ListObservations() []ObservationView
	ListSamples() []SampleView
	ListProtocols() []ProtocolView
	ListPermits() []PermitView
	ListProjects() []ProjectView
	ListSupplyItems() []SupplyItemView
	FindOrganism(id string) (OrganismView, bool)
	FindHousingUnit(id string) (HousingUnitView, bool)
	FindFacility(id string) (FacilityView, bool)
	FindTreatment(id string) (TreatmentView, bool)
	FindObservation(id string) (ObservationView, bool)
	FindSample(id string) (SampleView, bool)
	FindPermit(id string) (PermitView, bool)
	FindSupplyItem(id string) (SupplyItemView, bool)
}
