package core

import (
	"colonycore/pkg/domain"
	entitymodel "colonycore/pkg/domain/entitymodel"
	"context"
	"testing"
	"time"
)

func TestProtocolCoverageOrganismMismatch(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore(NewRulesEngine())
	rule := ProtocolCoverageRule()

	var protocol domain.Protocol
	var organism domain.Organism
	_, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		proto, err := tx.CreateProtocol(domain.Protocol{Protocol: entitymodel.Protocol{
			ID:          "prot-1",
			Code:        "P-1",
			Title:       "Study",
			MaxSubjects: 10,
			Status:      entitymodel.ProtocolStatusApproved,
		}})
		if err != nil {
			return err
		}
		protocol = proto
		org, err := tx.CreateOrganism(domain.Organism{Organism: entitymodel.Organism{
			ID:      "org-1",
			Name:    "Specimen",
			Species: "frog",
			Line:    "L1",
			Stage:   entitymodel.LifecycleStageAdult,
		}})
		if err != nil {
			return err
		}
		organism = org
		return nil
	})
	if err != nil {
		t.Fatalf("prepare state: %v", err)
	}

	procedure := domain.Procedure{Procedure: entitymodel.Procedure{
		ID:          "proc-1",
		Name:        "Dose",
		ProtocolID:  protocol.ID,
		ScheduledAt: time.Now(),
		Status:      entitymodel.ProcedureStatusScheduled,
		OrganismIDs: []string{organism.ID},
	}}

	_ = store.View(ctx, func(v domain.TransactionView) error {
		res, evalErr := rule.Evaluate(ctx, v, []domain.Change{{Entity: domain.EntityProcedure, After: mustChangePayload(t, procedure)}})
		if evalErr != nil {
			t.Fatalf("evaluate protocol coverage: %v", evalErr)
		}
		if len(res.Violations) == 0 {
			t.Fatalf("expected violation when organism lacks protocol assignment")
		}
		return nil
	})
}

func TestProtocolCoverageTreatmentBlocksPendingProtocol(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore(NewRulesEngine())
	rule := ProtocolCoverageRule()

	var procedure domain.Procedure
	_, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		proto, err := tx.CreateProtocol(domain.Protocol{Protocol: entitymodel.Protocol{
			ID:          "prot-2",
			Code:        "P-2",
			Title:       "Study 2",
			MaxSubjects: 10,
			Status:      entitymodel.ProtocolStatusSubmitted,
		}})
		if err != nil {
			return err
		}
		organismProtocolID := proto.ID
		_, err = tx.CreateOrganism(domain.Organism{Organism: entitymodel.Organism{
			ID:         "org-2",
			Name:       "Specimen",
			Species:    "frog",
			Line:       "L1",
			Stage:      entitymodel.LifecycleStageAdult,
			ProtocolID: &organismProtocolID,
		}})
		if err != nil {
			return err
		}
		proc, err := tx.CreateProcedure(domain.Procedure{Procedure: entitymodel.Procedure{
			ID:          "proc-2",
			Name:        "Dose",
			ProtocolID:  proto.ID,
			ScheduledAt: time.Now(),
			Status:      entitymodel.ProcedureStatusScheduled,
		}})
		if err != nil {
			return err
		}
		procedure = proc
		return nil
	})
	if err != nil {
		t.Fatalf("prepare pending protocol: %v", err)
	}

	treatment := domain.Treatment{Treatment: entitymodel.Treatment{
		ID:          "treat-1",
		Name:        "Dose",
		ProcedureID: procedure.ID,
		Status:      entitymodel.TreatmentStatusPlanned,
		DosagePlan:  "standard",
	}}

	_ = store.View(ctx, func(v domain.TransactionView) error {
		res, evalErr := rule.Evaluate(ctx, v, []domain.Change{{Entity: domain.EntityTreatment, After: mustChangePayload(t, treatment)}})
		if evalErr != nil {
			t.Fatalf("evaluate protocol coverage: %v", evalErr)
		}
		if len(res.Violations) == 0 {
			t.Fatalf("expected violation when protocol is not approved")
		}
		return nil
	})
}

func TestProtocolCoverageProcedureMissingProtocol(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore(NewRulesEngine())
	rule := ProtocolCoverageRule()

	procedure := domain.Procedure{Procedure: entitymodel.Procedure{
		ID:          "proc-missing-protocol",
		Name:        "Dose",
		ScheduledAt: time.Now(),
		Status:      entitymodel.ProcedureStatusScheduled,
	}}

	_ = store.View(ctx, func(v domain.TransactionView) error {
		res, err := rule.Evaluate(ctx, v, []domain.Change{{Entity: domain.EntityProcedure, After: mustChangePayload(t, procedure)}})
		if err != nil {
			t.Fatalf("evaluate protocol coverage: %v", err)
		}
		if len(res.Violations) == 0 {
			t.Fatalf("expected violation when procedure lacks protocol")
		}
		return nil
	})
}

func TestProtocolCoverageProcedureUnknownProtocol(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore(NewRulesEngine())
	rule := ProtocolCoverageRule()

	procedure := domain.Procedure{Procedure: entitymodel.Procedure{
		ID:          "proc-unknown-protocol",
		Name:        "Dose",
		ProtocolID:  "missing",
		ScheduledAt: time.Now(),
		Status:      entitymodel.ProcedureStatusScheduled,
	}}

	_ = store.View(ctx, func(v domain.TransactionView) error {
		res, err := rule.Evaluate(ctx, v, []domain.Change{{Entity: domain.EntityProcedure, After: mustChangePayload(t, procedure)}})
		if err != nil {
			t.Fatalf("evaluate protocol coverage: %v", err)
		}
		if len(res.Violations) == 0 {
			t.Fatalf("expected violation when procedure references unknown protocol")
		}
		return nil
	})
}

func TestProtocolCoverageProcedureUnknownOrganism(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore(NewRulesEngine())
	rule := ProtocolCoverageRule()

	var protocol domain.Protocol
	_, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		proto, err := tx.CreateProtocol(domain.Protocol{Protocol: entitymodel.Protocol{
			ID:          "prot-organism",
			Code:        "PO",
			Title:       "Study",
			MaxSubjects: 5,
			Status:      entitymodel.ProtocolStatusApproved,
		}})
		if err != nil {
			return err
		}
		protocol = proto
		return nil
	})
	if err != nil {
		t.Fatalf("prepare protocol: %v", err)
	}

	procedure := domain.Procedure{Procedure: entitymodel.Procedure{
		ID:          "proc-unknown-organism",
		Name:        "Dose",
		ProtocolID:  protocol.ID,
		ScheduledAt: time.Now(),
		Status:      entitymodel.ProcedureStatusScheduled,
		OrganismIDs: []string{"missing"},
	}}

	_ = store.View(ctx, func(v domain.TransactionView) error {
		res, evalErr := rule.Evaluate(ctx, v, []domain.Change{{Entity: domain.EntityProcedure, After: mustChangePayload(t, procedure)}})
		if evalErr != nil {
			t.Fatalf("evaluate protocol coverage: %v", evalErr)
		}
		if len(res.Violations) == 0 {
			t.Fatalf("expected violation when organism is unknown")
		}
		return nil
	})
}

func TestProtocolCoverageTreatmentMissingProcedure(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore(NewRulesEngine())
	rule := ProtocolCoverageRule()

	treatment := domain.Treatment{Treatment: entitymodel.Treatment{
		ID:         "treat-missing-procedure",
		Name:       "Dose",
		DosagePlan: "plan",
	}}

	_ = store.View(ctx, func(v domain.TransactionView) error {
		res, err := rule.Evaluate(ctx, v, []domain.Change{{Entity: domain.EntityTreatment, After: mustChangePayload(t, treatment)}})
		if err != nil {
			t.Fatalf("evaluate protocol coverage: %v", err)
		}
		if len(res.Violations) == 0 {
			t.Fatalf("expected violation when treatment lacks procedure reference")
		}
		return nil
	})
}

func TestProtocolCoverageTreatmentUnknownProcedure(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore(NewRulesEngine())
	rule := ProtocolCoverageRule()

	treatment := domain.Treatment{Treatment: entitymodel.Treatment{
		ID:          "treat-unknown-procedure",
		Name:        "Dose",
		ProcedureID: "missing",
		DosagePlan:  "plan",
	}}

	_ = store.View(ctx, func(v domain.TransactionView) error {
		res, err := rule.Evaluate(ctx, v, []domain.Change{{Entity: domain.EntityTreatment, After: mustChangePayload(t, treatment)}})
		if err != nil {
			t.Fatalf("evaluate protocol coverage: %v", err)
		}
		if len(res.Violations) == 0 {
			t.Fatalf("expected violation when treatment references unknown procedure")
		}
		return nil
	})
}

func TestProtocolCoverageTreatmentProcedureMissingProtocol(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore(NewRulesEngine())
	rule := ProtocolCoverageRule()

	_, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		_, err := tx.CreateProcedure(domain.Procedure{Procedure: entitymodel.Procedure{
			ID:          "proc-no-protocol",
			Name:        "Dose",
			ScheduledAt: time.Now(),
			Status:      entitymodel.ProcedureStatusScheduled,
		}})
		return err
	})
	if err != nil {
		t.Fatalf("prepare procedure: %v", err)
	}

	treatment := domain.Treatment{Treatment: entitymodel.Treatment{
		ID:          "treat-no-protocol",
		Name:        "Dose",
		ProcedureID: "proc-no-protocol",
		DosagePlan:  "plan",
	}}

	_ = store.View(ctx, func(v domain.TransactionView) error {
		res, err := rule.Evaluate(ctx, v, []domain.Change{{Entity: domain.EntityTreatment, After: mustChangePayload(t, treatment)}})
		if err != nil {
			t.Fatalf("evaluate protocol coverage: %v", err)
		}
		if len(res.Violations) == 0 {
			t.Fatalf("expected violation when procedure lacks protocol")
		}
		return nil
	})
}

func TestProtocolCoverageTreatmentUnknownOrganism(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore(NewRulesEngine())
	rule := ProtocolCoverageRule()

	var procedure domain.Procedure
	_, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		proto, err := tx.CreateProtocol(domain.Protocol{Protocol: entitymodel.Protocol{
			ID:          "prot-treatment",
			Code:        "PT",
			Title:       "Study",
			MaxSubjects: 5,
			Status:      entitymodel.ProtocolStatusApproved,
		}})
		if err != nil {
			return err
		}
		proc, err := tx.CreateProcedure(domain.Procedure{Procedure: entitymodel.Procedure{
			ID:          "proc-treatment",
			Name:        "Dose",
			ProtocolID:  proto.ID,
			ScheduledAt: time.Now(),
			Status:      entitymodel.ProcedureStatusScheduled,
		}})
		if err != nil {
			return err
		}
		procedure = proc
		return nil
	})
	if err != nil {
		t.Fatalf("prepare treatment procedure: %v", err)
	}

	treatment := domain.Treatment{Treatment: entitymodel.Treatment{
		ID:          "treat-missing-organism",
		Name:        "Dose",
		ProcedureID: procedure.ID,
		Status:      entitymodel.TreatmentStatusPlanned,
		OrganismIDs: []string{"missing"},
	}}

	_ = store.View(ctx, func(v domain.TransactionView) error {
		res, evalErr := rule.Evaluate(ctx, v, []domain.Change{{Entity: domain.EntityTreatment, After: mustChangePayload(t, treatment)}})
		if evalErr != nil {
			t.Fatalf("evaluate protocol coverage: %v", evalErr)
		}
		if len(res.Violations) == 0 {
			t.Fatalf("expected violation when treatment references unknown organism")
		}
		return nil
	})
}

func TestProtocolCoverageTreatmentOrganismProtocolMismatch(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore(NewRulesEngine())
	rule := ProtocolCoverageRule()

	var procedure domain.Procedure
	var organism domain.Organism
	_, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		approved, err := tx.CreateProtocol(domain.Protocol{Protocol: entitymodel.Protocol{
			ID:          "prot-approved",
			Code:        "PA",
			Title:       "Approved",
			MaxSubjects: 5,
			Status:      entitymodel.ProtocolStatusApproved,
		}})
		if err != nil {
			return err
		}
		other, err := tx.CreateProtocol(domain.Protocol{Protocol: entitymodel.Protocol{
			ID:          "prot-other",
			Code:        "PO",
			Title:       "Other",
			MaxSubjects: 5,
			Status:      entitymodel.ProtocolStatusApproved,
		}})
		if err != nil {
			return err
		}
		orgProtoID := other.ID
		org, err := tx.CreateOrganism(domain.Organism{Organism: entitymodel.Organism{
			ID:         "org-mismatch",
			Name:       "Specimen",
			Species:    "frog",
			Line:       "line",
			Stage:      entitymodel.LifecycleStageAdult,
			ProtocolID: &orgProtoID,
		}})
		if err != nil {
			return err
		}
		organism = org
		proc, err := tx.CreateProcedure(domain.Procedure{Procedure: entitymodel.Procedure{
			ID:          "proc-approved",
			Name:        "Dose",
			ProtocolID:  approved.ID,
			ScheduledAt: time.Now(),
			Status:      entitymodel.ProcedureStatusScheduled,
		}})
		if err != nil {
			return err
		}
		procedure = proc
		return nil
	})
	if err != nil {
		t.Fatalf("prepare mismatch treatment: %v", err)
	}

	treatment := domain.Treatment{Treatment: entitymodel.Treatment{
		ID:          "treat-mismatch",
		Name:        "Dose",
		ProcedureID: procedure.ID,
		Status:      entitymodel.TreatmentStatusInProgress,
		OrganismIDs: []string{organism.ID},
	}}

	_ = store.View(ctx, func(v domain.TransactionView) error {
		res, evalErr := rule.Evaluate(ctx, v, []domain.Change{{Entity: domain.EntityTreatment, After: mustChangePayload(t, treatment)}})
		if evalErr != nil {
			t.Fatalf("evaluate protocol coverage: %v", evalErr)
		}
		if len(res.Violations) == 0 {
			t.Fatalf("expected violation when treatment organism is not covered by procedure protocol")
		}
		return nil
	})
}

func TestProtocolCoverageRuleName(t *testing.T) {
	if got := ProtocolCoverageRule().Name(); got != "protocol_coverage" {
		t.Fatalf("unexpected rule name: %s", got)
	}
}

func TestProtocolCoverageIgnoresNilChanges(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore(NewRulesEngine())
	rule := ProtocolCoverageRule()

	_ = store.View(ctx, func(v domain.TransactionView) error {
		changes := []domain.Change{{Entity: domain.EntityProcedure}, {Entity: domain.EntityTreatment}}
		res, err := rule.Evaluate(ctx, v, changes)
		if err != nil {
			t.Fatalf("evaluate protocol coverage: %v", err)
		}
		if len(res.Violations) != 0 {
			t.Fatalf("expected no violations for nil changes, got %d", len(res.Violations))
		}
		return nil
	})
}

func TestProtocolCoverageSkipsInvalidPayload(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore(NewRulesEngine())
	rule := ProtocolCoverageRule()

	changeSet := []domain.Change{
		{Entity: domain.EntityProcedure, After: domain.NewChangePayload([]byte("{"))},
		{Entity: domain.EntityTreatment, After: domain.NewChangePayload([]byte("{"))},
	}

	_ = store.View(ctx, func(v domain.TransactionView) error {
		res, err := rule.Evaluate(ctx, v, changeSet)
		if err != nil {
			t.Fatalf("evaluate protocol coverage: %v", err)
		}
		if len(res.Violations) != 0 {
			t.Fatalf("expected no violations when payloads are wrong type")
		}
		return nil
	})
}
