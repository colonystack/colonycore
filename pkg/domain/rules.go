package domain

import (
	"context"
	"sync"
	"time"
)

// RuleView provides read-only access to domain entities for rule evaluation.
type RuleView interface {
	ListOrganisms() []Organism
	ListHousingUnits() []HousingUnit
	ListFacilities() []Facility
	ListTreatments() []Treatment
	ListObservations() []Observation
	ListSamples() []Sample
	ListProtocols() []Protocol
	ListPermits() []Permit
	ListProjects() []Project
	ListSupplyItems() []SupplyItem
	FindOrganism(id string) (Organism, bool)
	FindHousingUnit(id string) (HousingUnit, bool)
	FindFacility(id string) (Facility, bool)
	FindTreatment(id string) (Treatment, bool)
	FindObservation(id string) (Observation, bool)
	FindSample(id string) (Sample, bool)
	FindPermit(id string) (Permit, bool)
	FindSupplyItem(id string) (SupplyItem, bool)
	FindProcedure(id string) (Procedure, bool)
}

// Rule defines an evaluation executed within a transaction boundary.
type Rule interface {
	Name() string
	Evaluate(ctx context.Context, view RuleView, changes []Change) (Result, error)
}

// RulesEngine orchestrates rule evaluation.
type RulesEngine struct {
	rules      []Rule
	observer   RuleObserver
	observerMu sync.RWMutex
}

// NewRulesEngine constructs an engine instance.
func NewRulesEngine() *RulesEngine {
	return &RulesEngine{
		observer: noopRuleObserver{},
	}
}

// Register appends a rule to the engine.
func (e *RulesEngine) Register(rule Rule) {
	e.rules = append(e.rules, rule)
}

// RuleExecutionEvent captures one rule invocation outcome.
type RuleExecutionEvent struct {
	Rule                   string
	ChangeCount            int
	ViolationCount         int
	BlockingViolationCount int
	Duration               time.Duration
	Error                  error
}

// RuleObserver receives rule execution telemetry events.
type RuleObserver interface {
	RecordRuleExecution(ctx context.Context, event RuleExecutionEvent)
}

type noopRuleObserver struct{}

func (noopRuleObserver) RecordRuleExecution(context.Context, RuleExecutionEvent) {}

// SetObserver installs a rule observer. Passing nil disables callbacks.
func (e *RulesEngine) SetObserver(observer RuleObserver) {
	e.observerMu.Lock()
	defer e.observerMu.Unlock()
	if observer == nil {
		e.observer = noopRuleObserver{}
		return
	}
	e.observer = observer
}

// Evaluate executes all registered rules and aggregates their results.
func (e *RulesEngine) Evaluate(ctx context.Context, view RuleView, changes []Change) (Result, error) {
	var combined Result
	observer := e.ruleObserver()
	for _, rule := range e.rules {
		start := time.Now()
		res, err := rule.Evaluate(ctx, view, changes)
		observer.RecordRuleExecution(ctx, RuleExecutionEvent{
			Rule:                   rule.Name(),
			ChangeCount:            len(changes),
			ViolationCount:         len(res.Violations),
			BlockingViolationCount: countBlockingViolations(res),
			Duration:               time.Since(start),
			Error:                  err,
		})
		if err != nil {
			return Result{}, err
		}
		combined.Merge(res)
	}
	return combined, nil
}

func (e *RulesEngine) ruleObserver() RuleObserver {
	e.observerMu.RLock()
	defer e.observerMu.RUnlock()
	return e.observer
}

func countBlockingViolations(result Result) int {
	total := 0
	for _, violation := range result.Violations {
		if violation.Severity == SeverityBlock {
			total++
		}
	}
	return total
}
