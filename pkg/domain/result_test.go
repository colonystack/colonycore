package domain

import (
	"context"
	"fmt"
	"testing"
)

func TestResultMergeAndBlocking(t *testing.T) {
	var result Result
	result.Merge(Result{Violations: []Violation{{Rule: "warn", Severity: SeverityWarn}}})
	if result.HasBlocking() {
		t.Fatalf("expected no blocking violations")
	}
	result.Merge(Result{Violations: []Violation{{Rule: "block", Severity: SeverityBlock}}})
	if !result.HasBlocking() {
		t.Fatalf("expected blocking violation")
	}
	err := RuleViolationError{Result: result}
	if err.Error() == "" {
		t.Fatalf("expected error string")
	}
}

func TestResultMergeEmptyInput(t *testing.T) {
	original := Result{Violations: []Violation{{Rule: "existing", Severity: SeverityWarn}}}
	original.Merge(Result{})
	if len(original.Violations) != 1 || original.Violations[0].Rule != "existing" {
		t.Fatalf("expected original violations to remain, got %+v", original.Violations)
	}
}

func TestRulesEngineEvaluate(t *testing.T) {
	engine := NewRulesEngine()
	engine.Register(staticRule{"warn"})
	res, err := engine.Evaluate(context.Background(), emptyView{}, nil)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if len(res.Violations) != 1 {
		t.Fatalf("expected violation")
	}
}

type staticRule struct{ name string }

func (r staticRule) Name() string { return r.name }

func (r staticRule) Evaluate(ctx context.Context, view RuleView, changes []Change) (Result, error) {
	return Result{Violations: []Violation{{Rule: r.name, Severity: SeverityWarn}}}, nil
}

type emptyView struct{}

func (emptyView) ListOrganisms() []Organism                  { return nil }
func (emptyView) ListHousingUnits() []HousingUnit            { return nil }
func (emptyView) ListProtocols() []Protocol                  { return nil }
func (emptyView) FindOrganism(string) (Organism, bool)       { return Organism{}, false }
func (emptyView) FindHousingUnit(string) (HousingUnit, bool) { return HousingUnit{}, false }

func TestRulesEngineEvaluateError(t *testing.T) {
	engine := NewRulesEngine()
	engine.Register(errorRule{})
	if _, err := engine.Evaluate(context.Background(), emptyView{}, nil); err == nil {
		t.Fatalf("expected evaluation error")
	}
}

type errorRule struct{}

func (errorRule) Name() string { return "error" }

func (errorRule) Evaluate(ctx context.Context, view RuleView, changes []Change) (Result, error) {
	return Result{}, fmt.Errorf("boom")
}
