package memory

import (
	"context"
	"fmt"
	"testing"
	"time"

	"colonycore/pkg/domain"
)

const changedValue = "changed"

func TestMustApplyNoPanic(t *testing.T) {
	defer func() {
		if err := recover(); err != nil {
			t.Fatalf("unexpected panic: %v", err)
		}
	}()
	mustApply("noop", nil)
}

func TestMustApplyPanicsOnError(t *testing.T) {
	defer func() {
		if err := recover(); err == nil {
			t.Fatalf("expected panic when err is non-nil")
		}
	}()
	mustApply("fail", fmt.Errorf("boom"))
}

func TestStoreGettersMissing(t *testing.T) {
	store := NewStore(nil)
	if _, ok := store.GetOrganism("missing"); ok {
		t.Fatalf("expected missing organism to return false")
	}
	if _, ok := store.GetHousingUnit("missing"); ok {
		t.Fatalf("expected missing housing unit to return false")
	}
	if _, ok := store.GetFacility("missing"); ok {
		t.Fatalf("expected missing facility to return false")
	}
	if _, ok := store.GetLine("missing"); ok {
		t.Fatalf("expected missing line to return false")
	}
	if _, ok := store.GetStrain("missing"); ok {
		t.Fatalf("expected missing strain to return false")
	}
	if _, ok := store.GetGenotypeMarker("missing"); ok {
		t.Fatalf("expected missing genotype marker to return false")
	}
	if _, ok := store.GetPermit("missing"); ok {
		t.Fatalf("expected missing permit to return false")
	}
	if _, err := store.RunInTransaction(context.Background(), func(tx domain.Transaction) error {
		if err := tx.DeleteProtocol("missing"); err == nil {
			t.Fatalf("expected delete protocol to error on missing id")
		}
		if err := tx.DeleteProject("missing"); err == nil {
			t.Fatalf("expected delete project to error on missing id")
		}
		if err := tx.DeleteSupplyItem("missing"); err == nil {
			t.Fatalf("expected delete supply to error on missing id")
		}
		return nil
	}); err != nil {
		t.Fatalf("unexpected transaction error: %v", err)
	}
}

func TestStoreDeleteGuards(t *testing.T) {
	store := NewStore(nil)
	ctx := context.Background()
	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		facility, err := tx.CreateFacility(domain.Facility{Name: "Test Facility"})
		if err != nil {
			return err
		}
		protocol, err := tx.CreateProtocol(domain.Protocol{Code: "PR", Title: "Protocol", MaxSubjects: 1})
		if err != nil {
			return err
		}
		validFrom := time.Now().UTC()
		validUntil := validFrom.Add(time.Hour)
		_, err = tx.CreatePermit(domain.Permit{
			PermitNumber:      "PERM-1",
			Authority:         "Gov",
			Status:            domain.PermitStatusApproved,
			ValidFrom:         validFrom,
			ValidUntil:        validUntil,
			AllowedActivities: []string{"store"},
			FacilityIDs:       []string{facility.ID},
			ProtocolIDs:       []string{protocol.ID},
		})
		if err != nil {
			return err
		}
		if err := tx.DeleteProtocol(protocol.ID); err == nil {
			t.Fatalf("expected protocol delete to fail when referenced")
		}

		project, err := tx.CreateProject(domain.Project{Code: "PRJ", Title: "Project", FacilityIDs: []string{facility.ID}})
		if err != nil {
			return err
		}
		_, err = tx.CreateSupplyItem(domain.SupplyItem{
			SKU:            "SKU-1",
			Name:           "Supply",
			QuantityOnHand: 1,
			Unit:           "unit",
			FacilityIDs:    []string{facility.ID},
			ProjectIDs:     []string{project.ID},
			ReorderLevel:   1,
		})
		if err != nil {
			return err
		}
		if err := tx.DeleteProject(project.ID); err == nil {
			t.Fatalf("expected project delete to fail when referenced")
		}
		return nil
	}); err != nil {
		t.Fatalf("transaction error: %v", err)
	}
}

func TestTransactionViewMissingFinders(t *testing.T) {
	store := NewStore(nil)
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
		return nil
	}); err != nil {
		t.Fatalf("view error: %v", err)
	}
}

func TestCloneDeepCopies(t *testing.T) {
	desc := "desc"
	reason := "reason"
	now := time.Now().UTC()

	line := Line{
		Description:       &desc,
		Origin:            "field",
		GenotypeMarkerIDs: []string{"marker-1"},
		DeprecatedAt:      &now,
		DeprecationReason: &reason,
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

	strain := Strain{
		Description:       &desc,
		Generation:        &desc,
		RetiredAt:         &now,
		RetirementReason:  &reason,
		LineID:            "line-1",
		GenotypeMarkerIDs: []string{"marker-1"},
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

	marker := GenotypeMarker{
		Name:           "Marker",
		Locus:          "loc",
		Alleles:        []string{"A"},
		AssayMethod:    "PCR",
		Interpretation: "control",
		Version:        "v1",
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

	breeding := BreedingUnit{
		FemaleIDs: []string{"f1"},
		MaleIDs:   []string{"m1"},
	}
	if err := breeding.ApplyPairingAttributes(map[string]any{"core": map[string]any{"note": "pair"}}); err != nil {
		t.Fatalf("apply breeding attributes: %v", err)
	}
	clonedBreeding := cloneBreeding(breeding)
	breeding.FemaleIDs[0] = changedValue
	if clonedBreeding.FemaleIDs[0] != "f1" {
		t.Fatalf("expected breeding IDs deep copied")
	}

	observation := Observation{}
	if err := observation.ApplyObservationData(map[string]any{"score": 5}); err != nil {
		t.Fatalf("apply observation data: %v", err)
	}
	clonedObservation := cloneObservation(observation)
	data := clonedObservation.ObservationData()
	if data["score"].(int) != 5 {
		t.Fatalf("expected observation data copied, got %+v", data)
	}

	facility := Facility{HousingUnitIDs: []string{"h1"}, ProjectIDs: []string{"p1"}}
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

func TestSnapshotRoundTripCoverage(t *testing.T) {
	now := time.Now().UTC()
	lineID := "line-rtt"
	strainID := "strain-rtt"
	markerID := "marker-rtt"
	orgID := "org-rtt"
	procID := "proc-rtt"
	facilityID := "fac-rtt"
	cohortID := "cohort-rtt"
	projectID := "project-rtt"

	org := Organism{Base: domain.Base{ID: orgID}, Name: "Org", Species: "Spec", ParentIDs: []string{"parent"}}
	if err := org.SetCoreAttributes(map[string]any{}); err != nil {
		t.Fatalf("set organism attrs: %v", err)
	}

	line := Line{
		Base:              domain.Base{ID: lineID},
		Code:              "L-RT",
		Name:              "Line",
		Origin:            "field",
		GenotypeMarkerIDs: []string{markerID},
	}
	if err := line.ApplyDefaultAttributes(map[string]any{}); err != nil {
		t.Fatalf("apply line defaults: %v", err)
	}
	if err := line.ApplyExtensionOverrides(map[string]any{}); err != nil {
		t.Fatalf("apply line overrides: %v", err)
	}

	strain := Strain{
		Base:              domain.Base{ID: strainID},
		Code:              "S-RT",
		Name:              "Strain",
		LineID:            lineID,
		GenotypeMarkerIDs: []string{markerID},
	}
	if err := strain.ApplyStrainAttributes(map[string]any{}); err != nil {
		t.Fatalf("apply strain attrs: %v", err)
	}

	marker := GenotypeMarker{
		Base:        domain.Base{ID: markerID},
		Name:        "Marker",
		Locus:       "loc",
		Alleles:     []string{"A"},
		AssayMethod: "PCR",
		Version:     "v1",
	}
	if err := marker.ApplyGenotypeMarkerAttributes(map[string]any{}); err != nil {
		t.Fatalf("apply marker attrs: %v", err)
	}

	breeding := BreedingUnit{Base: domain.Base{ID: "breed-rtt"}, Name: "Breed", Strategy: "pair", LineID: &lineID, StrainID: &strainID, FemaleIDs: []string{"f"}, MaleIDs: []string{"m"}}
	if err := breeding.ApplyPairingAttributes(map[string]any{}); err != nil {
		t.Fatalf("apply breeding attrs: %v", err)
	}

	protocol := Protocol{Base: domain.Base{ID: "prot-rtt"}, Code: "PR", Title: "Protocol", MaxSubjects: 1, Status: domain.ProtocolStatusApproved}

	housing := HousingUnit{Base: domain.Base{ID: "house-rtt"}, Name: "Housing", FacilityID: facilityID, Capacity: 2, Environment: domain.HousingEnvironmentAquatic}

	facility := Facility{Base: domain.Base{ID: facilityID}, Code: "FAC", Name: "Facility", Zone: "zone", AccessPolicy: "policy", HousingUnitIDs: []string{housing.ID}, ProjectIDs: []string{projectID}}
	if err := facility.ApplyEnvironmentBaselines(map[string]any{"temp": 21}); err != nil {
		t.Fatalf("apply baselines: %v", err)
	}

	cohort := Cohort{Base: domain.Base{ID: cohortID}, Name: "Cohort", Purpose: "Study", ProjectID: &projectID, HousingID: &housing.FacilityID, ProtocolID: &protocol.ID}

	procedure := Procedure{Base: domain.Base{ID: procID}, Name: "Proc", Status: domain.ProcedureStatusScheduled, ScheduledAt: now, ProtocolID: protocol.ID, CohortID: &cohortID, OrganismIDs: []string{orgID}}

	treatment := Treatment{Base: domain.Base{ID: "treat-rtt"}, Name: "Treat", Status: domain.TreatmentStatusPlanned, ProcedureID: procID, OrganismIDs: []string{orgID}, CohortIDs: []string{cohortID}}

	if err := breeding.ApplyPairingAttributes(map[string]any{}); err != nil {
		t.Fatalf("apply breeding attrs: %v", err)
	}

	obs := Observation{Base: domain.Base{ID: "obs-rtt"}, ProcedureID: &procID, OrganismID: &orgID, RecordedAt: now, Observer: "tech"}
	if err := obs.ApplyObservationData(map[string]any{"score": 1}); err != nil {
		t.Fatalf("apply observation data: %v", err)
	}

	sample := Sample{Base: domain.Base{ID: "samp-rtt"}, Identifier: "S", SourceType: "blood", FacilityID: facilityID, OrganismID: &orgID, CollectedAt: now, Status: domain.SampleStatusStored, StorageLocation: "cold"}
	if err := sample.ApplySampleAttributes(map[string]any{}); err != nil {
		t.Fatalf("apply sample attrs: %v", err)
	}

	permit := Permit{
		Base:              domain.Base{ID: "permit-rtt"},
		PermitNumber:      "PERMIT",
		Authority:         "Gov",
		Status:            domain.PermitStatusApproved,
		ValidFrom:         now,
		ValidUntil:        now.Add(time.Hour),
		AllowedActivities: []string{"store"},
		FacilityIDs:       []string{facilityID},
		ProtocolIDs:       []string{protocol.ID},
	}

	project := Project{Base: domain.Base{ID: projectID}, Code: "PRJ", Title: "Project", FacilityIDs: []string{facilityID}}

	supply := SupplyItem{
		Base:           domain.Base{ID: "supply-rtt"},
		SKU:            "SKU",
		Name:           "Supply",
		QuantityOnHand: 1,
		Unit:           "unit",
		FacilityIDs:    []string{facilityID},
		ProjectIDs:     []string{projectID},
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

func TestMigrateSnapshotPrunesInvalidReferences(t *testing.T) {
	now := time.Now().UTC()
	lineID := "line-keep"
	missingLineID := "line-missing"
	markerID := "marker-keep"
	strainDropID := "strain-drop"
	strainKeepID := "strain-keep"
	facilityID := "facility-keep"
	procID := "proc-keep"

	org := Organism{Base: domain.Base{ID: "org-1"}, Name: "Org", Species: "Spec", LineID: &missingLineID, StrainID: &strainDropID}

	snapshot := Snapshot{
		Organisms: map[string]Organism{org.ID: org},
		Lines: map[string]Line{
			lineID: {Base: domain.Base{ID: lineID}, Code: "L", Name: "Line", GenotypeMarkerIDs: []string{markerID, "missing-marker"}},
		},
		Strains: map[string]Strain{
			strainDropID: {Base: domain.Base{ID: strainDropID}, Code: "SD", Name: "Drop", LineID: missingLineID, GenotypeMarkerIDs: []string{markerID, "missing-marker"}},
			strainKeepID: {Base: domain.Base{ID: strainKeepID}, Code: "SK", Name: "Keep", LineID: lineID, GenotypeMarkerIDs: []string{markerID, "missing-marker"}},
		},
		Markers: map[string]GenotypeMarker{
			markerID: {Base: domain.Base{ID: markerID}, Name: "Marker", Locus: "loc", Alleles: []string{"A", "A"}},
		},
		Housing: map[string]HousingUnit{
			"housing-keep": {Base: domain.Base{ID: "housing-keep"}, Name: "Keep", FacilityID: facilityID, Capacity: 0},
			"housing-drop": {Base: domain.Base{ID: "housing-drop"}, Name: "Drop", FacilityID: "missing", Capacity: 0},
		},
		Facilities: map[string]Facility{
			facilityID: {Base: domain.Base{ID: facilityID}, Code: "FAC", Name: "Facility"},
		},
		Treatments: map[string]Treatment{
			"treatment-drop": {Base: domain.Base{ID: "treatment-drop"}, Name: "Treat", ProcedureID: "missing-proc", OrganismIDs: []string{org.ID}},
		},
		Observations: map[string]Observation{
			"observation-drop": {Base: domain.Base{ID: "observation-drop"}, RecordedAt: now, Observer: "tech"},
			"observation-keep": {Base: domain.Base{ID: "observation-keep"}, ProcedureID: &procID, RecordedAt: now, Observer: "tech"},
		},
		Samples: map[string]Sample{
			"sample-keep": {Base: domain.Base{ID: "sample-keep"}, Identifier: "S", SourceType: "organism", FacilityID: facilityID, OrganismID: &org.ID, CollectedAt: now, Status: domain.SampleStatusStored, StorageLocation: "cold"},
			"sample-drop": {Base: domain.Base{ID: "sample-drop"}, Identifier: "SD", SourceType: "organism", OrganismID: &org.ID, CollectedAt: now, Status: domain.SampleStatusStored},
		},
		Protocols: map[string]Protocol{
			"protocol-keep": {Base: domain.Base{ID: "protocol-keep"}, Code: "PR", Title: "Protocol"},
		},
		Procedures: map[string]Procedure{
			procID: {Base: domain.Base{ID: procID}, Name: "Proc", ProtocolID: "protocol-keep", Status: domain.ProcedureStatusScheduled},
		},
		Permits: map[string]Permit{
			"permit-keep": {Base: domain.Base{ID: "permit-keep"}, PermitNumber: "PN", Authority: "Auth", ValidFrom: now, ValidUntil: now.Add(time.Hour), AllowedActivities: []string{"store"}, FacilityIDs: []string{facilityID, "missing"}, ProtocolIDs: []string{"protocol-keep", "missing"}},
		},
		Projects: map[string]Project{
			"project-keep": {Base: domain.Base{ID: "project-keep"}, Code: "PRJ", Title: "Project", FacilityIDs: []string{facilityID, "missing"}},
		},
	}

	migrated := migrateSnapshot(snapshot)

	if migrated.Cohorts == nil || migrated.Treatments == nil || migrated.Supplies == nil {
		t.Fatalf("expected nil maps to be initialized")
	}
	if _, ok := migrated.Strains[strainDropID]; ok {
		t.Fatalf("expected strain with missing line removed")
	}
	keptStrain := migrated.Strains[strainKeepID]
	if len(keptStrain.GenotypeMarkerIDs) != 1 || keptStrain.GenotypeMarkerIDs[0] != markerID {
		t.Fatalf("expected strain markers filtered, got %+v", keptStrain.GenotypeMarkerIDs)
	}
	keptLine := migrated.Lines[lineID]
	if len(keptLine.GenotypeMarkerIDs) != 1 || keptLine.GenotypeMarkerIDs[0] != markerID {
		t.Fatalf("expected line markers filtered, got %+v", keptLine.GenotypeMarkerIDs)
	}
	if alleles := migrated.Markers[markerID].Alleles; len(alleles) != 1 || alleles[0] != "A" {
		t.Fatalf("expected marker alleles deduped, got %+v", alleles)
	}
	migratedOrg := migrated.Organisms[org.ID]
	if migratedOrg.LineID != nil || migratedOrg.StrainID != nil {
		t.Fatalf("expected organism references cleared, got line=%v strain=%v", migratedOrg.LineID, migratedOrg.StrainID)
	}
	if _, ok := migrated.Housing["housing-drop"]; ok {
		t.Fatalf("expected housing with missing facility removed")
	}
	normalizedHousing := migrated.Housing["housing-keep"]
	if normalizedHousing.Capacity != 1 || normalizedHousing.Environment != defaultHousingEnvironment || normalizedHousing.State != defaultHousingState {
		t.Fatalf("expected housing normalized, got %+v", normalizedHousing)
	}
	if _, ok := migrated.Samples["sample-drop"]; ok {
		t.Fatalf("expected sample without facility removed")
	}
	if _, ok := migrated.Samples["sample-keep"]; !ok {
		t.Fatalf("expected sample with facility retained")
	}
	if _, ok := migrated.Observations["observation-drop"]; ok {
		t.Fatalf("expected observation without references removed")
	}
	if _, ok := migrated.Treatments["treatment-drop"]; ok {
		t.Fatalf("expected treatments without procedure removed")
	}
	migratedPermit := migrated.Permits["permit-keep"]
	if migratedPermit.Status != defaultPermitStatus || len(migratedPermit.FacilityIDs) != 1 || len(migratedPermit.ProtocolIDs) != 1 {
		t.Fatalf("expected permit normalized and filtered, got %+v", migratedPermit)
	}
	if migrated.Protocols["protocol-keep"].Status != defaultProtocolStatus {
		t.Fatalf("expected protocol status defaulted")
	}
	if facilities := migrated.Projects["project-keep"].FacilityIDs; len(facilities) != 1 || facilities[0] != facilityID {
		t.Fatalf("expected project facilities filtered, got %+v", facilities)
	}
	proc := migrated.Procedures[procID]
	if len(proc.ObservationIDs) != 1 || proc.ObservationIDs[0] != "observation-keep" {
		t.Fatalf("expected observation IDs populated, got %+v", proc.ObservationIDs)
	}
}
