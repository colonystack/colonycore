package domain

import "testing"

func TestResultMerge(t *testing.T) {
	r := Result{Violations: []Violation{{Rule: "a"}}}
	other := Result{Violations: []Violation{{Rule: "b"}}}

	r.Merge(other)

	if len(r.Violations) != 2 {
		t.Fatalf("expected 2 violations, got %d", len(r.Violations))
	}
	if r.Violations[1].Rule != "b" {
		t.Fatalf("expected violation appended")
	}
}

func TestResultMergeNoopEmpty(t *testing.T) {
	r := Result{Violations: []Violation{{Rule: "existing"}}}
	empty := Result{}

	r.Merge(empty)

	if len(r.Violations) != 1 {
		t.Fatalf("expected existing violations unchanged")
	}
}

func TestResultHasBlocking(t *testing.T) {
	cases := []struct {
		name   string
		input  Result
		expect bool
	}{
		{
			name:   "no violations",
			input:  Result{},
			expect: false,
		},
		{
			name:   "warn only",
			input:  Result{Violations: []Violation{{Severity: SeverityWarn}}},
			expect: false,
		},
		{
			name:   "contains block",
			input:  Result{Violations: []Violation{{Severity: SeverityBlock}}},
			expect: true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Helper()
			if got := tc.input.HasBlocking(); got != tc.expect {
				t.Fatalf("HasBlocking=%v want %v", got, tc.expect)
			}
		})
	}
}

func TestRuleViolationError(t *testing.T) {
	err := RuleViolationError{Result: Result{Violations: []Violation{{Rule: "rule"}}}}
	if err.Error() == "" {
		t.Fatalf("expected error string")
	}
}
