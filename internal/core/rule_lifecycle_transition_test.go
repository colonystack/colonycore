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
		res, err := rule.Evaluate(ctx, v, []domain.Change{{Entity: domain.EntityOrganism, Before: before, After: after}})
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
		res, err := rule.Evaluate(ctx, v, []domain.Change{{Entity: domain.EntityProcedure, After: invalid}})
		if err != nil {
			t.Fatalf("evaluate lifecycle rule: %v", err)
		}
		if len(res.Violations) == 0 {
			t.Fatalf("expected violation for invalid procedure state")
		}
		return nil
	})
}

func TestLifecycleEntityIDHelperCoversTypes(t *testing.T) {
	organismID := "org"
	housingID := "house"
	procedureID := "proc"
	treatmentID := "treat"
	protocolID := "proto"
	permitID := "permit"
	sampleID := "sample"
	cases := []struct {
		name  string
		model any
		want  string
	}{
		{"organism", domain.Organism{Organism: entitymodel.Organism{ID: organismID}}, organismID},
		{"housing", domain.HousingUnit{HousingUnit: entitymodel.HousingUnit{ID: housingID}}, housingID},
		{"procedure", domain.Procedure{Procedure: entitymodel.Procedure{ID: procedureID}}, procedureID},
		{"treatment", domain.Treatment{Treatment: entitymodel.Treatment{ID: treatmentID}}, treatmentID},
		{"protocol", domain.Protocol{Protocol: entitymodel.Protocol{ID: protocolID}}, protocolID},
		{"permit", domain.Permit{Permit: entitymodel.Permit{ID: permitID}}, permitID},
		{"sample", domain.Sample{Sample: entitymodel.Sample{ID: sampleID}}, sampleID},
		{"unknown", struct{ ref string }{ref: "ignored"}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := entityID(tc.model); got != tc.want {
				t.Fatalf("entityID(%s) = %s, want %s", tc.name, got, tc.want)
			}
		})
	}
}

func TestLifecycleTransitionRuleName(t *testing.T) {
	if got := LifecycleTransitionRule().Name(); got != "lifecycle_transition" {
		t.Fatalf("unexpected rule name: %s", got)
	}
}
