package core

import (
	"colonycore/pkg/domain"
	entitymodel "colonycore/pkg/domain/entitymodel"
	"context"
	"testing"
	"time"
)

func TestLifecycleTransitionBlocksTerminalExit(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore(NewRulesEngine())
	rule := LifecycleTransitionRule()

	before := domain.Organism{Organism: entitymodel.Organism{ID: "o1", Name: "Retired", Species: "frog", Line: "L1", Stage: entitymodel.LifecycleStageRetired}}
	after := domain.Organism{Organism: entitymodel.Organism{ID: "o1", Name: "Retired", Species: "frog", Line: "L1", Stage: entitymodel.LifecycleStageAdult}}

	_ = store.View(ctx, func(v domain.TransactionView) error {
		res, err := rule.Evaluate(ctx, v, []domain.Change{{
			Entity: domain.EntityOrganism,
			Before: mustChangePayload(t, before),
			After:  mustChangePayload(t, after),
		}})
		if err != nil {
			t.Fatalf("evaluate lifecycle rule: %v", err)
		}
		if len(res.Violations) == 0 {
			t.Fatalf("expected lifecycle transition violation when leaving terminal state")
		}
		return nil
	})
}

func TestLifecycleTransitionInvalidState(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore(NewRulesEngine())
	rule := LifecycleTransitionRule()

	invalid := domain.Procedure{Procedure: entitymodel.Procedure{
		ID:          "p1",
		Name:        "Proc",
		ProtocolID:  "proto",
		Status:      entitymodel.ProcedureStatus("warp"),
		ScheduledAt: time.Now(),
	}}

	_ = store.View(ctx, func(v domain.TransactionView) error {
		res, err := rule.Evaluate(ctx, v, []domain.Change{{
			Entity: domain.EntityProcedure,
			After:  mustChangePayload(t, invalid),
		}})
		if err != nil {
			t.Fatalf("evaluate lifecycle rule: %v", err)
		}
		if len(res.Violations) == 0 {
			t.Fatalf("expected violation for invalid procedure state")
		}
		return nil
	})
}

func TestLifecycleTransitionSkipsInvalidPayload(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore(NewRulesEngine())
	rule := LifecycleTransitionRule()

	invalid := domain.NewChangePayload([]byte("{"))
	_ = store.View(ctx, func(v domain.TransactionView) error {
		res, err := rule.Evaluate(ctx, v, []domain.Change{{
			Entity: domain.EntityProcedure,
			After:  invalid,
		}})
		if err != nil {
			t.Fatalf("evaluate lifecycle rule: %v", err)
		}
		if len(res.Violations) != 0 {
			t.Fatalf("expected invalid payload to be skipped, got %v", res.Violations)
		}
		return nil
	})
}

func TestLifecycleTransitionRuleName(t *testing.T) {
	if got := LifecycleTransitionRule().Name(); got != "lifecycle_transition" {
		t.Fatalf("unexpected rule name: %s", got)
	}
}
