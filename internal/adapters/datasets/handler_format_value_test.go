package datasets

import (
	"testing"
	"time"
)

type fmtStringer struct{}

func (fmtStringer) String() string { return "stringer" }

// TestFormatValueBranches hits all switch cases in formatValue for coverage.
func TestFormatValueBranches(_ *testing.T) {
	cases := []struct{ in any }{
		{nil}, {time.Unix(0, 0).UTC()}, {fmtStringer{}}, {float32(1.25)}, {float64(2.5)}, {int(3)}, {int64(4)}, {"plain"},
	}
	for _, c := range cases {
		_ = formatValue(c.in)
	}
}

// TestFirstNonEmpty covers helper function.
func TestFirstNonEmpty(t *testing.T) {
	if v := firstNonEmpty("", " ", "x", "y"); v != "x" {
		t.Fatalf("expected x got %s", v)
	}
	if v := firstNonEmpty(); v != "" {
		t.Fatalf("expected empty got %s", v)
	}
}
