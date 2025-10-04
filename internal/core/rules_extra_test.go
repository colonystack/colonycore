package core

import (
	"context"
	"testing"

	"colonycore/pkg/domain"
)

func TestProtocolSubjectCapRuleViolation(t *testing.T) {
	rule := NewProtocolSubjectCapRule()
	mem := NewMemoryStore(NewRulesEngine())
	// create protocol with cap 1 and two organisms referencing it
	_, _ = mem.RunInTransaction(context.Background(), func(tx domain.Transaction) error {
		p, _ := tx.CreateProtocol(domain.Protocol{Code: "P", Title: "Prot", MaxSubjects: 1})
		_, _ = tx.CreateOrganism(domain.Organism{Name: "A", Species: "frog", ProtocolID: &p.ID})
		_, _ = tx.CreateOrganism(domain.Organism{Name: "B", Species: "frog", ProtocolID: &p.ID})
		return nil
	})
	// obtain a read-only snapshot view and evaluate rule
	_ = mem.View(context.Background(), func(v domain.TransactionView) error {
		vr, err := rule.Evaluate(context.Background(), v, nil)
		if err != nil || !vr.HasBlocking() {
			t.Fatalf("expected blocking violation: %+v %v", vr, err)
		}
		return nil
	})
}

func TestHousingCapacityRuleViolation(t *testing.T) {
	rule := NewHousingCapacityRule()
	mem := NewMemoryStore(NewRulesEngine())
	_, _ = mem.RunInTransaction(context.Background(), func(tx domain.Transaction) error {
		h, _ := tx.CreateHousingUnit(domain.HousingUnit{Name: "H", Capacity: 1})
		_, _ = tx.CreateOrganism(domain.Organism{Name: "A", Species: "frog", HousingID: &h.ID})
		_, _ = tx.CreateOrganism(domain.Organism{Name: "B", Species: "frog", HousingID: &h.ID})
		return nil
	})
	_ = mem.View(context.Background(), func(v domain.TransactionView) error {
		vr, err := rule.Evaluate(context.Background(), v, nil)
		if err != nil || !vr.HasBlocking() {
			t.Fatalf("expected housing capacity violation")
		}
		return nil
	})
}
