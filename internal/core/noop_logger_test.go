package core

import (
	"testing"
)

// TestNoopLoggerCoverageExtra tests all noop logger methods to increase coverage
func TestNoopLoggerCoverageExtra(_ *testing.T) {
	logger := noopLogger{}

	// Test all logger methods - they should not panic
	logger.Debug("test debug message", "key", "value")
	logger.Info("test info message", "key", "value")
	logger.Warn("test warn message", "key", "value")
	logger.Error("test error message", "key", "value")
}
