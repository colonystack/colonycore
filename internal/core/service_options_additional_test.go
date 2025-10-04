package core

import (
	"context"
	"testing"
	"time"

	"colonycore/pkg/domain"
)

type stubClock struct{ t time.Time }

func (s stubClock) Now() time.Time { return s.t }

type captureLogger struct{ calls []string }

func (c *captureLogger) Debug(msg string, _ ...any) { c.calls = append(c.calls, "d:"+msg) }
func (c *captureLogger) Info(msg string, _ ...any)  { c.calls = append(c.calls, "i:"+msg) }
func (c *captureLogger) Warn(msg string, _ ...any)  { c.calls = append(c.calls, "w:"+msg) }
func (c *captureLogger) Error(msg string, _ ...any) { c.calls = append(c.calls, "e:"+msg) }

// TestServiceOptionsCoversClockLogger ensures option overrides take effect (clock + logger coverage).
func TestServiceOptionsCoversClockLogger(t *testing.T) {
	fixed := time.Unix(123, 0).UTC()
	clk := stubClock{t: fixed}
	log := &captureLogger{}
	svc := NewInMemoryService(nil, WithClock(clk), WithLogger(log))
	// invoke a couple operations to trigger logger usage in run()
	if _, _, err := svc.CreateProject(context.Background(), domain.Project{Base: domain.Base{ID: "p1"}}); err != nil {
		t.Fatalf("create project: %v", err)
	}
	if svc.clock == nil || svc.clock.Now().Unix() != fixed.Unix() {
		t.Fatalf("expected clock override to be used")
	}
	if len(log.calls) == 0 {
		t.Fatalf("expected logger to record calls")
	}
}
