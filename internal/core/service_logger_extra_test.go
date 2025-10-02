package core

import "testing"

// TestNoopLoggerMethods directly invokes noopLogger methods to cover them.
func TestNoopLoggerMethods(_ *testing.T) {
	var l noopLogger
	l.Debug("d", "k", 1)
	l.Info("i", "k2", 2)
	l.Warn("w", "k3", 3)
	l.Error("e", "k4", 4)
}

// TestDefaultServiceOptions ensures default options wiring (clock + logger) executes without nil derefs.
func TestDefaultServiceOptions(t *testing.T) {
	opts := defaultServiceOptions()
	if opts.clock == nil || opts.logger == nil {
		t.Fatalf("expected defaults populated")
	}
	_ = opts.clock.Now() // cover ClockFunc.Now
}
