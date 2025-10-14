package core

import "testing"

func TestNoopLoggerCoverage(_ *testing.T) {
	var l Logger = noopLogger{}
	l.Debug("message")
	l.Info("message")
	l.Warn("message")
	l.Error("message")
}
