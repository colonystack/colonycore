package sqlite

import (
	"colonycore/pkg/domain"
	entitymodel "colonycore/pkg/domain/entitymodel"
	"context"
	"fmt"
	"testing"
	"time"
)

// Migrated minimal representative tests; original exhaustive tests remain at old path until cleanup.

func TestMustApplyNoPanicSQLite(t *testing.T) {
	defer func() {
		if err := recover(); err != nil {
			t.Fatalf("unexpected panic: %v", err)
		}
	}()
	mustApply("noop", nil)
}

func TestMustApplyPanicsSQLite(t *testing.T) {
	defer func() {
		if err := recover(); err == nil {
			t.Fatalf("expected panic when err is non-nil")
		}
	}()
	mustApply("fail", fmt.Errorf("boom"))
}

func TestMemStoreBasicLifecycle(t *testing.T) {
	store := newMemStore(nil)
	ctx := context.Background()
	if store.NowFunc() == nil {
		t.Fatalf("expected NowFunc to be initialized")
	}
	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		_, err := tx.CreateOrganism(domain.Organism{Organism: entitymodel.Organism{Name: "Specimen", Species: "Test"}})
		return err
	}); err != nil {
		t.Fatalf("create organism: %v", err)
	}
	if len(store.ListOrganisms()) != 1 {
		t.Fatalf("expected 1 organism")
	}
	snapshot := store.ExportState()
	store.ImportState(Snapshot{})
	if len(store.ListOrganisms()) != 0 {
		t.Fatalf("expected cleared state")
	}
	store.ImportState(snapshot)
	if len(store.ListOrganisms()) != 1 {
		t.Fatalf("expected restored organism")
	}
}

func TestMemStoreRuleViolation(t *testing.T) {
	store := newMemStore(domain.NewRulesEngine())
	store.RulesEngine().Register(blockingRule{})
	if _, err := store.RunInTransaction(context.Background(), func(tx domain.Transaction) error {
		_, e := tx.CreateOrganism(domain.Organism{Organism: entitymodel.Organism{Name: "Fail"}})
		return e
	}); err == nil {
		t.Fatalf("expected violation error")
	}
}

type blockingRule struct{}

func (blockingRule) Name() string { return "block" }
func (blockingRule) Evaluate(_ context.Context, _ domain.RuleView, _ []domain.Change) (domain.Result, error) {
	r := domain.Result{}
	r.Merge(domain.Result{Violations: []domain.Violation{{Rule: "block", Severity: domain.SeverityBlock}}})
	return r, nil
}

func TestMemStoreCRUDReduced(t *testing.T) {
	store := newMemStore(nil)
	ctx := context.Background()
	var projectID string
	const updatedDesc = "updated"
	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		facility, err := tx.CreateFacility(domain.Facility{Facility: entitymodel.Facility{Name: "Project Facility"}})
		if err != nil {
			return err
		}
		proj, err := tx.CreateProject(domain.Project{Project: entitymodel.Project{Code: "PRJ", Title: "Project", FacilityIDs: []string{facility.ID}}})
		if err != nil {
			return err
		}
		projectID = proj.ID
		if _, err := tx.CreateOrganism(domain.Organism{Organism: entitymodel.Organism{Name: "Alpha", Species: "Frog"}}); err != nil {
			return err
		}
		return nil
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if got := len(store.ListProjects()); got != 1 {
		t.Fatalf("expected 1 project, got %d", got)
	}
	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		if _, err := tx.UpdateProject(projectID, func(p *domain.Project) error {
			p.Description = strPtr(updatedDesc)
			return nil
		}); err != nil {
			return err
		}
		return tx.DeleteProject(projectID)
	}); err != nil {
		t.Fatalf("mutate: %v", err)
	}
}

func TestMemStoreProcedureLifecycleReduced(t *testing.T) {
	store := newMemStore(nil)
	ctx := context.Background()
	now := time.Now().UTC()
	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		prot, err := tx.CreateProtocol(domain.Protocol{Protocol: entitymodel.Protocol{Code: "P", Title: "Proto", MaxSubjects: 5}})
		if err != nil {
			return err
		}
		_, err = tx.CreateProcedure(domain.Procedure{Procedure: entitymodel.Procedure{Name: "Check", Status: domain.ProcedureStatusScheduled, ScheduledAt: now, ProtocolID: prot.ID}})
		return err
	}); err != nil {
		t.Fatalf("create procedure: %v", err)
	}
	if got := len(store.ListProcedures()); got != 1 {
		t.Fatalf("expected one procedure, got %d", got)
	}
}

func TestMemStoreViewSnapshotReduced(t *testing.T) {
	store := newMemStore(nil)
	ctx := context.Background()
	if err := store.View(ctx, func(v domain.TransactionView) error {
		if len(v.ListOrganisms()) != 0 {
			return fmt.Errorf("expected empty")
		}
		return nil
	}); err != nil {
		t.Fatalf("view: %v", err)
	}
}

func TestMigrateSnapshotRelationships(t *testing.T) {
	store := newMemStore(nil)
	now := time.Now().UTC()

	organisms := map[string]domain.Organism{
		"org-1": {Organism: entitymodel.Organism{ID: "org-1", Name: "Org", Species: "Spec"}},
	}
	cohorts := map[string]domain.Cohort{
		"cohort-1": {Cohort: entitymodel.Cohort{ID: "cohort-1", Name: "Cohort"}},
	}
	protocols := map[string]domain.Protocol{
		"prot-1": {Protocol: entitymodel.Protocol{ID: "prot-1", Code: "PR", Title: "Protocol", MaxSubjects: 10, Status: domain.ProtocolStatusApproved}},
	}
	procedures := map[string]domain.Procedure{
		"proc-1": {Procedure: entitymodel.Procedure{ID: "proc-1", Name: "Proc", Status: domain.ProcedureStatusScheduled, ScheduledAt: now, ProtocolID: "prot-1", OrganismIDs: []string{"org-1"}}},
	}

	snapshot := Snapshot{
		Organisms: organisms,
		Cohorts:   cohorts,
		Housing: map[string]domain.HousingUnit{
			"house-1": {HousingUnit: entitymodel.HousingUnit{ID: "house-1", Name: "Housing", FacilityID: "fac-1", Capacity: 2}},
		},
		Facilities: map[string]domain.Facility{
			"fac-1": {Facility: entitymodel.Facility{ID: "fac-1", Name: "Facility"}},
		},
		Procedures: procedures,
		Treatments: map[string]domain.Treatment{
			"treat-1": {Treatment: entitymodel.Treatment{ID: "treat-1", Name: "Treat", Status: domain.TreatmentStatusPlanned, ProcedureID: "proc-1", OrganismIDs: []string{"org-1", "org-1"}, CohortIDs: []string{"missing"}}},
		},
		Observations: map[string]domain.Observation{
			"obs-1": {Observation: entitymodel.Observation{ID: "obs-1", ProcedureID: ptr("proc-1"), Observer: "Tech", RecordedAt: now}},
		},
		Samples: map[string]domain.Sample{
			"sample-1": {Sample: entitymodel.Sample{ID: "sample-1", Identifier: "S1", SourceType: "blood", FacilityID: "fac-1", OrganismID: ptr("org-1"), CollectedAt: now, Status: domain.SampleStatusStored, StorageLocation: "freezer", ChainOfCustody: []domain.SampleCustodyEvent{{Actor: "tech", Location: "freezer", Timestamp: now}}}},
		},
		Protocols: protocols,
		Permits: map[string]domain.Permit{
			"permit-1": {Permit: entitymodel.Permit{ID: "permit-1", PermitNumber: "P1", Authority: "Gov", Status: domain.PermitStatusApproved, ValidFrom: now, ValidUntil: now.AddDate(1, 0, 0), FacilityIDs: []string{"fac-1", "fac-1"}, ProtocolIDs: []string{"prot-1"}}},
		},
		Projects: map[string]domain.Project{
			"proj-1": {Project: entitymodel.Project{ID: "proj-1", Code: "P1", Title: "Project", FacilityIDs: []string{"fac-1", "fac-1"}}},
		},
		Supplies: map[string]domain.SupplyItem{
			"supply-1": {SupplyItem: entitymodel.SupplyItem{ID: "supply-1", SKU: "SKU", Name: "Gloves", QuantityOnHand: 5, Unit: "box", FacilityIDs: []string{"fac-1"}, ProjectIDs: []string{"proj-1", "proj-1"}}},
		},
	}

	store.ImportState(snapshot)

	facility, ok := store.GetFacility("fac-1")
	if !ok {
		t.Fatalf("expected facility present")
	}
	if len(facility.HousingUnitIDs) != 1 || facility.HousingUnitIDs[0] != "house-1" {
		t.Fatalf("expected facility housing ids migrated, got %+v", facility.HousingUnitIDs)
	}
	if len(facility.ProjectIDs) != 1 || facility.ProjectIDs[0] != "proj-1" {
		t.Fatalf("expected facility project ids migrated, got %+v", facility.ProjectIDs)
	}

	if treatments := store.ListTreatments(); len(treatments) != 1 || len(treatments[0].OrganismIDs) != 1 || treatments[0].OrganismIDs[0] != "org-1" {
		t.Fatalf("expected deduped treatment organism ids, got %+v", treatments)
	}

	if permits := store.ListPermits(); len(permits) != 1 || len(permits[0].FacilityIDs) != 1 || permits[0].FacilityIDs[0] != "fac-1" {
		t.Fatalf("expected permit facility ids filtered, got %+v", permits)
	}

	if supplies := store.ListSupplyItems(); len(supplies) != 1 || len(supplies[0].ProjectIDs) != 1 || supplies[0].ProjectIDs[0] != "proj-1" {
		t.Fatalf("expected supply project ids filtered, got %+v", supplies)
	}
}

func TestMemStoreStateNormalization(t *testing.T) {
	store := newMemStore(nil)
	now := time.Now().UTC()
	if _, err := store.RunInTransaction(context.Background(), func(tx domain.Transaction) error {
		facility, err := tx.CreateFacility(domain.Facility{Facility: entitymodel.Facility{Name: "Lab"}})
		if err != nil {
			return err
		}
		housing, err := tx.CreateHousingUnit(domain.HousingUnit{HousingUnit: entitymodel.HousingUnit{Name: "H", FacilityID: facility.ID, Capacity: 1}})
		if err != nil {
			return err
		}
		if housing.State != domain.HousingStateQuarantine || housing.Environment != domain.HousingEnvironmentTerrestrial {
			return fmt.Errorf("expected housing defaults applied, got state=%q env=%q", housing.State, housing.Environment)
		}
		if _, err := tx.CreateHousingUnit(domain.HousingUnit{HousingUnit: entitymodel.HousingUnit{Name: "InvalidEnv", FacilityID: facility.ID, Capacity: 1, Environment: domain.HousingEnvironment("invalid")}}); err == nil {
			return fmt.Errorf("expected invalid housing environment to error")
		}

		protocol, err := tx.CreateProtocol(domain.Protocol{Protocol: entitymodel.Protocol{Code: "P", Title: "Proto", MaxSubjects: 1}})
		if err != nil {
			return err
		}
		if protocol.Status != domain.ProtocolStatusDraft {
			return fmt.Errorf("expected protocol status defaulted, got %q", protocol.Status)
		}
		if _, err := tx.CreateProtocol(domain.Protocol{Protocol: entitymodel.Protocol{Code: "P2", Title: "Invalid", MaxSubjects: 1, Status: domain.ProtocolStatus("invalid")}}); err == nil {
			return fmt.Errorf("expected invalid protocol status to error")
		}

		permit, err := tx.CreatePermit(domain.Permit{Permit: entitymodel.Permit{PermitNumber: "PER",
			Authority:         "Gov",
			ValidFrom:         now,
			ValidUntil:        now.Add(time.Hour),
			AllowedActivities: []string{"store"},
			FacilityIDs:       []string{facility.ID},
			ProtocolIDs:       []string{protocol.ID}},
		})
		if err != nil {
			return err
		}
		if permit.Status != domain.PermitStatusDraft {
			return fmt.Errorf("expected permit status defaulted, got %q", permit.Status)
		}
		if _, err := tx.CreatePermit(domain.Permit{Permit: entitymodel.Permit{PermitNumber: "PER-2",
			Authority:         "Gov",
			Status:            domain.PermitStatus("invalid"),
			ValidFrom:         now,
			ValidUntil:        now.Add(time.Hour),
			AllowedActivities: []string{"store"},
			FacilityIDs:       []string{facility.ID},
			ProtocolIDs:       []string{protocol.ID}},
		}); err == nil {
			return fmt.Errorf("expected invalid permit status to error")
		}
		if _, err := tx.UpdateHousingUnit(housing.ID, func(h *domain.HousingUnit) error {
			h.Environment = domain.HousingEnvironment("invalid")
			return nil
		}); err == nil {
			return fmt.Errorf("expected invalid housing environment on update to error")
		}
		if _, err := tx.UpdateProtocol(protocol.ID, func(p *domain.Protocol) error {
			p.Status = domain.ProtocolStatus("invalid")
			return nil
		}); err == nil {
			return fmt.Errorf("expected invalid protocol status on update to error")
		}
		if _, err := tx.UpdatePermit(permit.ID, func(p *domain.Permit) error {
			p.Status = domain.PermitStatus("invalid")
			return nil
		}); err == nil {
			return fmt.Errorf("expected invalid permit status on update to error")
		}
		return nil
	}); err != nil {
		t.Fatalf("transaction error: %v", err)
	}
}

func TestMemStoreTransactionViewMissingFinders(t *testing.T) {
	store := newMemStore(nil)
	if err := store.View(context.Background(), func(v domain.TransactionView) error {
		if _, ok := v.FindOrganism("missing"); ok {
			t.Fatalf("expected missing organism")
		}
		if _, ok := v.FindTreatment("missing"); ok {
			t.Fatalf("expected missing treatment")
		}
		if _, ok := v.FindObservation("missing"); ok {
			t.Fatalf("expected missing observation")
		}
		if _, ok := v.FindPermit("missing"); ok {
			t.Fatalf("expected missing permit")
		}
		if _, ok := v.FindLine("missing"); ok {
			t.Fatalf("expected missing line")
		}
		if _, ok := v.FindStrain("missing"); ok {
			t.Fatalf("expected missing strain")
		}
		if _, ok := v.FindGenotypeMarker("missing"); ok {
			t.Fatalf("expected missing genotype marker")
		}
		if _, ok := v.FindSupplyItem("missing"); ok {
			t.Fatalf("expected missing supply item")
		}
		if _, ok := v.FindHousingUnit("missing"); ok {
			t.Fatalf("expected missing housing unit")
		}
		if _, ok := v.FindProcedure("missing"); ok {
			t.Fatalf("expected missing procedure")
		}
		return nil
	}); err != nil {
		t.Fatalf("view error: %v", err)
	}
}

func TestCloneDeepCopiesSQLite(t *testing.T) {
	desc := "desc"
	reason := "reason"
	now := time.Now().UTC()

	line := Line{Line: entitymodel.Line{Description: &desc,
		Origin:            "field",
		GenotypeMarkerIDs: []string{"marker-1"},
		DeprecatedAt:      &now,
		DeprecationReason: &reason},
	}
	if err := line.ApplyDefaultAttributes(map[string]any{"core": map[string]any{"seed": true}}); err != nil {
		t.Fatalf("apply line defaults: %v", err)
	}
	if err := line.ApplyExtensionOverrides(map[string]any{"plugin": map[string]any{"override": 1}}); err != nil {
		t.Fatalf("apply line overrides: %v", err)
	}
	clonedLine := cloneLine(line)
	line.GenotypeMarkerIDs[0] = changedValue
	if clonedLine.GenotypeMarkerIDs[0] != "marker-1" {
		t.Fatalf("expected line marker IDs to be deep copied")
	}
	if clonedLine.Description == line.Description || clonedLine.DeprecatedAt == line.DeprecatedAt {
		t.Fatalf("expected line pointers to be copied, not shared")
	}

	strain := Strain{Strain: entitymodel.Strain{Description: &desc,
		Generation:        &desc,
		RetiredAt:         &now,
		RetirementReason:  &reason,
		LineID:            "line-1",
		GenotypeMarkerIDs: []string{"marker-1"}},
	}
	if err := strain.ApplyStrainAttributes(map[string]any{"core": map[string]any{"note": "strain"}}); err != nil {
		t.Fatalf("apply strain attributes: %v", err)
	}
	clonedStrain := cloneStrain(strain)
	strain.GenotypeMarkerIDs[0] = "mutated"
	if clonedStrain.GenotypeMarkerIDs[0] != "marker-1" {
		t.Fatalf("expected strain markers to be deep copied")
	}
	if clonedStrain.Description == strain.Description || clonedStrain.RetiredAt == strain.RetiredAt {
		t.Fatalf("expected strain pointers to be copied, not shared")
	}

	marker := GenotypeMarker{GenotypeMarker: entitymodel.GenotypeMarker{Name: "Marker",
		Locus:          "loc",
		Alleles:        []string{"A"},
		AssayMethod:    "PCR",
		Interpretation: "control",
		Version:        "v1"},
	}
	if err := marker.ApplyGenotypeMarkerAttributes(map[string]any{"core": map[string]any{"attr": "value"}}); err != nil {
		t.Fatalf("apply marker attributes: %v", err)
	}
	clonedMarker := cloneGenotypeMarker(marker)
	marker.Alleles[0] = "mutated"
	if clonedMarker.Alleles[0] != "A" {
		t.Fatalf("expected marker alleles to be deep copied")
	}
	if clonedMarker.Interpretation != marker.Interpretation {
		t.Fatalf("expected marker interpretation copied")
	}

	breeding := BreedingUnit{BreedingUnit: entitymodel.BreedingUnit{FemaleIDs: []string{"f1"},
		MaleIDs: []string{"m1"}},
	}
	if err := breeding.ApplyPairingAttributes(map[string]any{"core": map[string]any{"note": "pair"}}); err != nil {
		t.Fatalf("apply breeding attributes: %v", err)
	}
	clonedBreeding := cloneBreeding(breeding)
	breeding.FemaleIDs[0] = changedValue
	if clonedBreeding.FemaleIDs[0] != "f1" {
		t.Fatalf("expected breeding IDs deep copied")
	}

	observation := Observation{Observation: entitymodel.Observation{}}
	if err := observation.ApplyObservationData(map[string]any{"score": 5}); err != nil {
		t.Fatalf("apply observation data: %v", err)
	}
	clonedObservation := cloneObservation(observation)
	data := clonedObservation.ObservationData()
	if data["score"].(int) != 5 {
		t.Fatalf("expected observation data copied, got %+v", data)
	}

	facility := Facility{Facility: entitymodel.Facility{HousingUnitIDs: []string{"h1"}, ProjectIDs: []string{"p1"}}}
	if err := facility.ApplyEnvironmentBaselines(map[string]any{"temp": 22}); err != nil {
		t.Fatalf("apply facility baselines: %v", err)
	}
	clonedFacility := cloneFacility(facility)
	facility.HousingUnitIDs[0] = changedValue
	if clonedFacility.HousingUnitIDs[0] != "h1" {
		t.Fatalf("expected facility IDs deep copied")
	}
	if baselines := clonedFacility.EnvironmentBaselines(); baselines["temp"].(int) != 22 {
		t.Fatalf("expected facility baselines copied")
	}
}

func TestSnapshotRoundTripCoverageSQLite(t *testing.T) {
	now := time.Now().UTC()
	lineID := "line-rtt"
	strainID := "strain-rtt"
	markerID := "marker-rtt"
	orgID := "org-rtt"
	procID := "proc-rtt"
	facilityID := "fac-rtt"
	cohortID := "cohort-rtt"
	projectID := "project-rtt"

	org := Organism{Organism: entitymodel.Organism{ID: orgID, Name: "Org", Species: "Spec", ParentIDs: []string{"parent"}}}
	if err := org.SetCoreAttributes(map[string]any{}); err != nil {
		t.Fatalf("set organism attrs: %v", err)
	}

	line := Line{Line: entitymodel.Line{ID: lineID,
		Code:              "L-RT",
		Name:              "Line",
		Origin:            "field",
		GenotypeMarkerIDs: []string{markerID}},
	}
	if err := line.ApplyDefaultAttributes(map[string]any{}); err != nil {
		t.Fatalf("apply line defaults: %v", err)
	}
	if err := line.ApplyExtensionOverrides(map[string]any{}); err != nil {
		t.Fatalf("apply line overrides: %v", err)
	}

	strain := Strain{Strain: entitymodel.Strain{ID: strainID,
		Code:              "S-RT",
		Name:              "Strain",
		LineID:            lineID,
		GenotypeMarkerIDs: []string{markerID}},
	}
	if err := strain.ApplyStrainAttributes(map[string]any{}); err != nil {
		t.Fatalf("apply strain attrs: %v", err)
	}

	marker := GenotypeMarker{GenotypeMarker: entitymodel.GenotypeMarker{ID: markerID,
		Name:        "Marker",
		Locus:       "loc",
		Alleles:     []string{"A"},
		AssayMethod: "PCR",
		Version:     "v1"},
	}
	if err := marker.ApplyGenotypeMarkerAttributes(map[string]any{}); err != nil {
		t.Fatalf("apply marker attrs: %v", err)
	}

	breeding := BreedingUnit{BreedingUnit: entitymodel.BreedingUnit{ID: "breed-rtt", Name: "Breed", Strategy: "pair", LineID: &lineID, StrainID: &strainID, FemaleIDs: []string{"f"}, MaleIDs: []string{"m"}}}
	if err := breeding.ApplyPairingAttributes(map[string]any{}); err != nil {
		t.Fatalf("apply breeding attrs: %v", err)
	}

	protocol := Protocol{Protocol: entitymodel.Protocol{ID: "prot-rtt", Code: "PR", Title: "Protocol", MaxSubjects: 1, Status: domain.ProtocolStatusApproved}}

	housing := HousingUnit{HousingUnit: entitymodel.HousingUnit{ID: "house-rtt", Name: "Housing", FacilityID: facilityID, Capacity: 2, Environment: domain.HousingEnvironmentAquatic}}

	facility := Facility{Facility: entitymodel.Facility{ID: facilityID, Code: "FAC", Name: "Facility", Zone: "zone", AccessPolicy: "policy", HousingUnitIDs: []string{housing.ID}, ProjectIDs: []string{projectID}}}
	if err := facility.ApplyEnvironmentBaselines(map[string]any{"temp": 21}); err != nil {
		t.Fatalf("apply baselines: %v", err)
	}

	cohort := Cohort{Cohort: entitymodel.Cohort{ID: cohortID, Name: "Cohort", Purpose: "Study", ProjectID: &projectID, HousingID: &housing.FacilityID, ProtocolID: &protocol.ID}}

	procedure := Procedure{Procedure: entitymodel.Procedure{ID: procID, Name: "Proc", Status: domain.ProcedureStatusScheduled, ScheduledAt: now, ProtocolID: protocol.ID, CohortID: &cohortID, OrganismIDs: []string{orgID}}}

	treatment := Treatment{Treatment: entitymodel.Treatment{ID: "treat-rtt", Name: "Treat", Status: domain.TreatmentStatusPlanned, ProcedureID: procID, OrganismIDs: []string{orgID}, CohortIDs: []string{cohortID}}}

	obs := Observation{Observation: entitymodel.Observation{ID: "obs-rtt", ProcedureID: &procID, OrganismID: &orgID, RecordedAt: now, Observer: "tech"}}
	if err := obs.ApplyObservationData(map[string]any{"score": 1}); err != nil {
		t.Fatalf("apply observation data: %v", err)
	}

	sample := Sample{Sample: entitymodel.Sample{ID: "samp-rtt", Identifier: "S", SourceType: "blood", FacilityID: facilityID, OrganismID: &orgID, CollectedAt: now, Status: domain.SampleStatusStored, StorageLocation: "cold"}}
	if err := sample.ApplySampleAttributes(map[string]any{}); err != nil {
		t.Fatalf("apply sample attrs: %v", err)
	}

	permit := Permit{Permit: entitymodel.Permit{ID: "permit-rtt",
		PermitNumber:      "PERMIT",
		Authority:         "Gov",
		Status:            domain.PermitStatusApproved,
		ValidFrom:         now,
		ValidUntil:        now.Add(time.Hour),
		AllowedActivities: []string{"store"},
		FacilityIDs:       []string{facilityID},
		ProtocolIDs:       []string{protocol.ID}},
	}

	project := Project{Project: entitymodel.Project{ID: projectID, Code: "PRJ", Title: "Project", FacilityIDs: []string{facilityID}}}

	supply := SupplyItem{SupplyItem: entitymodel.SupplyItem{ID: "supply-rtt",
		SKU:            "SKU",
		Name:           "Supply",
		QuantityOnHand: 1,
		Unit:           "unit",
		FacilityIDs:    []string{facilityID},
		ProjectIDs:     []string{projectID}},
	}
	if err := supply.ApplySupplyAttributes(map[string]any{}); err != nil {
		t.Fatalf("apply supply attrs: %v", err)
	}

	state := memoryState{
		organisms:    map[string]Organism{orgID: org},
		cohorts:      map[string]Cohort{cohortID: cohort},
		housing:      map[string]HousingUnit{housing.ID: housing},
		facilities:   map[string]Facility{facilityID: facility},
		breeding:     map[string]BreedingUnit{breeding.ID: breeding},
		lines:        map[string]Line{lineID: line},
		strains:      map[string]Strain{strainID: strain},
		markers:      map[string]GenotypeMarker{markerID: marker},
		procedures:   map[string]Procedure{procID: procedure},
		treatments:   map[string]Treatment{treatment.ID: treatment},
		observations: map[string]Observation{obs.ID: obs},
		samples:      map[string]Sample{sample.ID: sample},
		protocols:    map[string]Protocol{protocol.ID: protocol},
		permits:      map[string]Permit{permit.ID: permit},
		projects:     map[string]Project{projectID: project},
		supplies:     map[string]SupplyItem{supply.ID: supply},
	}

	snapshot := snapshotFromMemoryState(state)
	restored := memoryStateFromSnapshot(snapshot)
	if len(restored.organisms) != 1 || len(restored.strains) != 1 || len(restored.markers) != 1 {
		t.Fatalf("expected round-trip data to persist")
	}
}
func TestMemStoreProcedureObservationSampleLifecycle(t *testing.T) {
	store := newMemStore(nil)
	now := time.Now().UTC()
	if _, err := store.RunInTransaction(context.Background(), func(tx domain.Transaction) error {
		facility, err := tx.CreateFacility(domain.Facility{Facility: entitymodel.Facility{Name: "Lab-2"}})
		if err != nil {
			return err
		}
		housing, err := tx.CreateHousingUnit(domain.HousingUnit{HousingUnit: entitymodel.HousingUnit{Name: "H2", FacilityID: facility.ID, Capacity: 1, Environment: domain.HousingEnvironmentTerrestrial}})
		if err != nil {
			return err
		}
		cohort, err := tx.CreateCohort(domain.Cohort{Cohort: entitymodel.Cohort{Name: "C2"}})
		if err != nil {
			return err
		}
		housingID := housing.ID
		organism, err := tx.CreateOrganism(domain.Organism{Organism: entitymodel.Organism{Name: "Org", Species: "Spec", HousingID: &housingID}})
		if err != nil {
			return err
		}
		protocol, err := tx.CreateProtocol(domain.Protocol{Protocol: entitymodel.Protocol{Code: "PR-2", Title: "Protocol 2", MaxSubjects: 1, Status: domain.ProtocolStatusApproved}})
		if err != nil {
			return err
		}
		procedure, err := tx.CreateProcedure(domain.Procedure{Procedure: entitymodel.Procedure{Name: "Proc",
			Status:      domain.ProcedureStatusScheduled,
			ScheduledAt: now,
			ProtocolID:  protocol.ID,
			OrganismIDs: []string{organism.ID}},
		})
		if err != nil {
			return err
		}
		treatment, err := tx.CreateTreatment(domain.Treatment{Treatment: entitymodel.Treatment{Name: "Treat",
			Status:            domain.TreatmentStatusPlanned,
			ProcedureID:       procedure.ID,
			OrganismIDs:       []string{organism.ID},
			AdministrationLog: []string{},
			AdverseEvents:     []string{}},
		})
		if err != nil {
			return err
		}
		if _, err := tx.UpdateTreatment(treatment.ID, func(t *domain.Treatment) error {
			t.Status = domain.TreatmentStatusCompleted
			return nil
		}); err != nil {
			return err
		}
		observation, err := tx.CreateObservation(domain.Observation{Observation: entitymodel.Observation{ProcedureID: &procedure.ID,
			OrganismID: &organism.ID,
			CohortID:   &cohort.ID,
			Observer:   "Tech",
			RecordedAt: now},
		})
		if err != nil {
			return err
		}
		if _, err := tx.UpdateObservation(observation.ID, func(o *domain.Observation) error {
			note := "updated"
			o.Notes = &note
			return nil
		}); err != nil {
			return err
		}
		sample, err := tx.CreateSample(domain.Sample{Sample: entitymodel.Sample{Identifier: "S-1",
			SourceType:      "blood",
			FacilityID:      facility.ID,
			CollectedAt:     now,
			Status:          domain.SampleStatusStored,
			StorageLocation: "loc",
			OrganismID:      &organism.ID,
			ChainOfCustody: []domain.SampleCustodyEvent{{
				Actor:     "tech",
				Location:  "loc",
				Timestamp: now,
			}}},
		})
		if err != nil {
			return err
		}
		if _, err := tx.UpdateSample(sample.ID, func(s *domain.Sample) error {
			s.Status = domain.SampleStatusInTransit
			return nil
		}); err != nil {
			return err
		}
		if err := tx.DeleteProcedure(procedure.ID); err == nil {
			return fmt.Errorf("expected delete procedure to fail while referenced")
		}
		if err := tx.DeleteTreatment(treatment.ID); err != nil {
			return err
		}
		if err := tx.DeleteObservation(observation.ID); err != nil {
			return err
		}
		if err := tx.DeleteProcedure(procedure.ID); err != nil {
			return err
		}
		return nil
	}); err != nil {
		t.Fatalf("transaction error: %v", err)
	}
}

func ptr[T any](v T) *T { return &v }
