package datasetapi

import (
	"testing"
	"time"
)

func TestOrganismNormalizesLifecycleStage(t *testing.T) {
	stages := NewLifecycleStageContext()

	juvenile := NewOrganism(OrganismData{
		Base:  BaseData{ID: "org-juvenile"},
		Name:  "Juvenile",
		Stage: "JuVeNiLe",
	})
	if juvenile.Stage() != stageJuvenile {
		t.Fatalf("expected lifecycle stage %q, got %q", stageJuvenile, juvenile.Stage())
	}
	if !juvenile.GetCurrentStage().Equals(stages.Juvenile()) {
		t.Fatalf("expected juvenile contextual stage")
	}

	defaulted := NewOrganism(OrganismData{
		Base:  BaseData{ID: "org-default"},
		Name:  "Defaulted",
		Stage: "mystery",
	})
	if defaulted.Stage() != stagePlanned {
		t.Fatalf("expected fallback lifecycle stage %q, got %q", stagePlanned, defaulted.Stage())
	}
	if !defaulted.GetCurrentStage().Equals(stages.Planned()) {
		t.Fatalf("expected planned contextual stage for unknown input")
	}
}

func TestHousingUnitNormalizesEnvironmentAndState(t *testing.T) {
	envCtx := NewHousingContext()
	stateCtx := NewHousingStateContext()

	housing := NewHousingUnit(HousingUnitData{
		Base:        BaseData{ID: "housing-1"},
		Name:        "Housing A",
		FacilityID:  "facility-1",
		Capacity:    2,
		Environment: "HUMID",
		State:       "Cleaning",
	})
	if housing.Environment() != environmentTypeHumid {
		t.Fatalf("expected environment %q, got %q", environmentTypeHumid, housing.Environment())
	}
	if !housing.GetEnvironmentType().Equals(envCtx.Humid()) {
		t.Fatalf("expected humid environment reference")
	}
	if housing.State() != string(housingStateCleaning) {
		t.Fatalf("expected housing state %q, got %q", housingStateCleaning, housing.State())
	}
	if !housing.GetState().Equals(stateCtx.Cleaning()) {
		t.Fatalf("expected cleaning housing state reference")
	}

	fallback := NewHousingUnit(HousingUnitData{
		Base:        BaseData{ID: "housing-2"},
		Name:        "Housing B",
		FacilityID:  "facility-2",
		Capacity:    1,
		Environment: "unknown",
		State:       "unknown",
	})
	if fallback.Environment() != environmentTypeTerrestrial {
		t.Fatalf("expected fallback environment %q, got %q", environmentTypeTerrestrial, fallback.Environment())
	}
	if !fallback.GetEnvironmentType().Equals(envCtx.Terrestrial()) {
		t.Fatalf("expected terrestrial environment reference for fallback")
	}
	if fallback.State() != string(housingStateActive) {
		t.Fatalf("expected fallback housing state %q, got %q", housingStateActive, fallback.State())
	}
	if !fallback.GetState().Equals(stateCtx.Active()) {
		t.Fatalf("expected active housing state reference for fallback")
	}

	quarantine := NewHousingUnit(HousingUnitData{
		Base:        BaseData{ID: "housing-3"},
		Name:        "Housing C",
		FacilityID:  "facility-3",
		Capacity:    3,
		Environment: "aquatic",
		State:       "QUARANTINE",
	})
	if quarantine.State() != string(housingStateQuarantine) {
		t.Fatalf("expected quarantine housing state, got %q", quarantine.State())
	}
	if !quarantine.GetState().Equals(stateCtx.Quarantine()) {
		t.Fatalf("expected quarantine housing state reference")
	}
}

func TestProcedureNormalizesStatus(t *testing.T) {
	ctx := NewProcedureContext()
	now := time.Now()

	proc := NewProcedure(ProcedureData{
		Base:        BaseData{ID: "proc-1"},
		Name:        "Procedure",
		Status:      "In_Progress",
		ScheduledAt: now,
		ProtocolID:  "proto",
	})
	if proc.Status() != procedureStatusInProgress {
		t.Fatalf("expected procedure status %q, got %q", procedureStatusInProgress, proc.Status())
	}
	if !proc.GetCurrentStatus().Equals(ctx.InProgress()) {
		t.Fatalf("expected in-progress procedure status reference")
	}

	fallback := NewProcedure(ProcedureData{
		Base:        BaseData{ID: "proc-2"},
		Name:        "Fallback",
		Status:      "unknown",
		ScheduledAt: now,
		ProtocolID:  "proto",
	})
	if fallback.Status() != procedureStatusScheduled {
		t.Fatalf("expected fallback procedure status %q, got %q", procedureStatusScheduled, fallback.Status())
	}
	if !fallback.GetCurrentStatus().Equals(ctx.Scheduled()) {
		t.Fatalf("expected scheduled procedure status reference for fallback")
	}

	failed := NewProcedure(ProcedureData{
		Base:        BaseData{ID: "proc-3"},
		Name:        "Failed",
		Status:      "FAILED",
		ScheduledAt: now,
		ProtocolID:  "proto",
	})
	if failed.Status() != procedureStatusFailed {
		t.Fatalf("expected failed procedure status %q, got %q", procedureStatusFailed, failed.Status())
	}
	if !failed.GetCurrentStatus().Equals(ctx.Failed()) {
		t.Fatalf("expected failed procedure status reference")
	}
}

func TestSampleNormalizesStatus(t *testing.T) {
	statuses := NewSampleContext().Statuses()
	now := time.Now()

	sample := NewSample(SampleData{
		Base:        BaseData{ID: "sample-1"},
		Identifier:  "S-1",
		FacilityID:  "facility",
		CollectedAt: now,
		Status:      "IN-TRANSIT",
	})
	if sample.Status() != datasetSampleStatusInTransit {
		t.Fatalf("expected sample status %q, got %q", datasetSampleStatusInTransit, sample.Status())
	}
	if !sample.GetStatus().Equals(statuses.InTransit()) {
		t.Fatalf("expected in-transit sample status reference")
	}

	fallback := NewSample(SampleData{
		Base:        BaseData{ID: "sample-2"},
		Identifier:  "S-2",
		FacilityID:  "facility",
		CollectedAt: now,
		Status:      "unknown",
	})
	if fallback.Status() != datasetSampleStatusStored {
		t.Fatalf("expected fallback sample status %q, got %q", datasetSampleStatusStored, fallback.Status())
	}
	if !fallback.GetStatus().Equals(statuses.Stored()) {
		t.Fatalf("expected stored sample status reference for fallback")
	}

	consumed := NewSample(SampleData{
		Base:        BaseData{ID: "sample-3"},
		Identifier:  "S-3",
		FacilityID:  "facility",
		CollectedAt: now,
		Status:      "consumed",
	})
	if consumed.Status() != datasetSampleStatusConsumed {
		t.Fatalf("expected consumed sample status %q, got %q", datasetSampleStatusConsumed, consumed.Status())
	}
	if !consumed.GetStatus().Equals(statuses.Consumed()) {
		t.Fatalf("expected consumed sample status reference")
	}
}

func TestProtocolAndPermitNormalizeStatus(t *testing.T) {
	protocolCtx := NewProtocolContext()
	permitCtx := NewPermitContext().Statuses()
	now := time.Now()

	protocol := NewProtocol(ProtocolData{
		Base:        BaseData{ID: "protocol-1"},
		Code:        "PR-1",
		Title:       "Protocol 1",
		MaxSubjects: 1,
		Status:      "APPROVED",
	})
	if !protocol.GetCurrentStatus().Equals(protocolCtx.Approved()) {
		t.Fatalf("expected approved protocol status reference")
	}

	paused := NewProtocol(ProtocolData{
		Base:        BaseData{ID: "protocol-2"},
		Code:        "PR-2",
		Title:       "Protocol 2",
		MaxSubjects: 1,
		Status:      "on_hold",
	})
	if !paused.GetCurrentStatus().Equals(protocolCtx.OnHold()) {
		t.Fatalf("expected on-hold protocol status reference for paused state")
	}

	permit := NewPermit(PermitData{
		Base:              BaseData{ID: "permit-1"},
		PermitNumber:      "PER-1",
		Authority:         "Authority",
		Status:            "SUBMITTED",
		ValidFrom:         now,
		ValidUntil:        now.Add(24 * time.Hour),
		AllowedActivities: []string{"observe"},
		FacilityIDs:       []string{"facility-1"},
		ProtocolIDs:       []string{"protocol-1"},
	})
	if !permit.GetStatus(now).Equals(permitCtx.Submitted()) {
		t.Fatalf("expected submitted permit status reference")
	}

	fallback := NewPermit(PermitData{
		Base:              BaseData{ID: "permit-2"},
		PermitNumber:      "PER-2",
		Authority:         "Authority",
		Status:            "unknown",
		ValidFrom:         now,
		ValidUntil:        now.Add(24 * time.Hour),
		AllowedActivities: []string{"collect"},
		FacilityIDs:       []string{"facility-2"},
		ProtocolIDs:       []string{"protocol-2"},
	})
	if !fallback.GetStatus(now).Equals(permitCtx.Draft()) {
		t.Fatalf("expected draft permit status reference for fallback")
	}

	archived := NewPermit(PermitData{
		Base:              BaseData{ID: "permit-3"},
		PermitNumber:      "PER-3",
		Authority:         "Authority",
		Status:            "ARCHIVED",
		ValidFrom:         now,
		ValidUntil:        now.Add(12 * time.Hour),
		AllowedActivities: []string{"store"},
		FacilityIDs:       []string{"facility-3"},
		ProtocolIDs:       []string{"protocol-3"},
	})
	if !archived.GetStatus(now).Equals(permitCtx.Archived()) {
		t.Fatalf("expected archived permit status reference")
	}
}
