package core_test

import (
	"context"
	"testing"

	"colonycore/internal/core"
	"colonycore/pkg/domain"
	entitymodel "colonycore/pkg/domain/entitymodel"
)

type captureLogger struct {
	debugs int
	infos  int
	errors int
}

func (l *captureLogger) Debug(string, ...any) { l.debugs++ }
func (l *captureLogger) Info(string, ...any)  { l.infos++ }
func (l *captureLogger) Warn(string, ...any)  {}
func (l *captureLogger) Error(string, ...any) { l.errors++ }

// TestServiceLoggerDebugAndError covers debug (success) and error (failure) logging paths.
func TestServiceLoggerDebugAndError(t *testing.T) {
	logger := &captureLogger{}
	svc := core.NewInMemoryService(core.NewDefaultRulesEngine(), core.WithLogger(logger))
	ctx := context.Background()
	facility, _, err := svc.CreateFacility(ctx, domain.Facility{Facility: entitymodel.Facility{Name: "Logger Facility"}})
	if err != nil {
		t.Fatalf("create facility: %v", err)
	}
	// success path: create project => run() succeeds => Debug call
	if _, _, err := svc.CreateProject(ctx, domain.Project{Project: entitymodel.Project{Code: "PRJ-LOG", Title: "Logging", FacilityIDs: []string{facility.ID}}}); err != nil {
		// shouldn't happen; but fail early
		t.Fatalf("create project: %v", err)
	}
	if logger.debugs == 0 {
		t.Fatalf("expected debug log on success")
	}
	// error path: assign protocol to non-existent organism or create error by assigning housing to missing organism
	if _, _, err := svc.AssignOrganismHousing(ctx, "missing", "also-missing"); err == nil {
		// should error inside transaction (update organism not found)
		// but to guarantee error path, attempt assigning protocol as alternative
		if _, _, err2 := svc.AssignOrganismProtocol(ctx, "missing", "missing-protocol"); err2 == nil {
			// if still no error something is wrong
			t.Fatalf("expected error from invalid assignment operations")
		}
	}
	if logger.errors == 0 {
		t.Fatalf("expected error log on failure path")
	}
}
