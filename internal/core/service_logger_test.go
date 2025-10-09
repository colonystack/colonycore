package core

import (
	"testing"
)

func TestNoopLogger(t *testing.T) {
	logger := noopLogger{}

	// These methods should not panic and should be no-ops
	t.Run("Debug does not panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Debug method panicked: %v", r)
			}
		}()
		logger.Debug("test message", "arg1", "arg2")
	})

	t.Run("Info does not panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Info method panicked: %v", r)
			}
		}()
		logger.Info("test message", "arg1", "arg2")
	})

	t.Run("Warn does not panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Warn method panicked: %v", r)
			}
		}()
		logger.Warn("test message", "arg1", "arg2")
	})

	t.Run("Error does not panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Error method panicked: %v", r)
			}
		}()
		logger.Error("test message", "arg1", "arg2")
	})
}
