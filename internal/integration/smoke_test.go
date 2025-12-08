package integration

import (
	"bytes"
	"colonycore/internal/blob"
	"context"
	"os"
	"path/filepath"
	"testing"

	core "colonycore/internal/core"
	domain "colonycore/pkg/domain"
	entitymodel "colonycore/pkg/domain/entitymodel"
)

// TestIntegrationSmoke exercises a minimal end-to-end write/read cycle for
// each supported in-process storage and blob adapter. It intentionally keeps
// scope tiny so it can act as a fast CI health check.
func TestIntegrationSmoke(t *testing.T) {
	ctx := context.Background()

	// Define core persistent store variants to exercise.
	coreVariants := []struct {
		name string
		open func(t *testing.T) domain.PersistentStore
	}{
		{
			name: "memory-store",
			open: func(_ *testing.T) domain.PersistentStore {
				return core.NewMemoryStore(core.NewDefaultRulesEngine())
			},
		},
		{
			name: "sqlite-store",
			open: func(t *testing.T) domain.PersistentStore {
				dir := t.TempDir()
				path := filepath.Join(dir, "core.db")
				s, err := core.NewSQLiteStore(path, core.NewDefaultRulesEngine())
				if err != nil {
					t.Fatalf("new sqlite store: %v", err)
				}
				return s
			},
		},
	}

	// Define blob adapters to exercise. Include a lightweight mocked S3 transport
	// (similar to unit test) so the smoke test covers all adapters in one place.
	blobVariants := []struct {
		name string
		open func(t *testing.T) blob.Store
	}{
		{
			name: "memory-blob",
			open: func(_ *testing.T) blob.Store { return blob.NewMemory() },
		},
		{
			name: "filesystem-blob",
			open: func(t *testing.T) blob.Store {
				dir := t.TempDir()
				fs, err := blob.NewFilesystem(dir)
				if err != nil {
					t.Fatalf("new filesystem blob: %v", err)
				}
				return fs
			},
		},
		{
			name: "mock-s3-blob",
			open: func(_ *testing.T) blob.Store { return blob.NewMockS3ForTests() },
		},
	}

	for _, cv := range coreVariants {
		t.Run(cv.name, func(t *testing.T) {
			store := cv.open(t)
			metricsRecorder := core.NewExpvarMetricsRecorder("")
			var traceBuffer bytes.Buffer
			tracer := core.NewJSONTracer(&traceBuffer)
			svc := core.NewService(
				store,
				core.WithMetricsRecorder(metricsRecorder),
				core.WithTracer(tracer),
			)
			facility, _, err := svc.CreateFacility(ctx, domain.Facility{Facility: entitymodel.Facility{Name: "Lab"}})
			if err != nil {
				t.Fatalf("create facility: %v", err)
			}
			// Write one housing unit and one organism referencing it.
			created, res, err := svc.CreateHousingUnit(ctx, domain.HousingUnit{HousingUnit: entitymodel.HousingUnit{Name: "Tank", FacilityID: facility.ID, Capacity: 1}})
			if err != nil {
				t.Fatalf("create housing: %v", err)
			}
			if res.HasBlocking() {
				t.Fatalf("unexpected blocking violations: %+v", res.Violations)
			}
			org, res, err := svc.CreateOrganism(ctx, domain.Organism{Organism: entitymodel.Organism{Name: "Specimen", Species: "Testus"}})
			if err != nil {
				t.Fatalf("create organism: %v", err)
			}
			if res.HasBlocking() {
				t.Fatalf("unexpected blocking violations organism: %+v", res.Violations)
			}
			// Assign organism to housing
			if _, res, err := svc.AssignOrganismHousing(ctx, org.ID, created.ID); err != nil {
				t.Fatalf("assign housing: %v", err)
			} else if res.HasBlocking() {
				t.Fatalf("unexpected violations on assignment: %+v", res.Violations)
			}
			// Ensure persisted via store view.
			found := false
			units := store.ListHousingUnits()
			for _, u := range units {
				if u.ID == created.ID {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("expected housing unit %s in listing", created.ID)
			}
			// Validate organism reflects assignment
			if got, ok := store.GetOrganism(org.ID); !ok || got.HousingID == nil || *got.HousingID != created.ID {
				t.Fatalf("expected organism housing assignment persisted")
			}

			// Validate observability exporters captured core operations.
			snapshot := metricsRecorder.Snapshot()
			if len(snapshot.DurationsMS) == 0 {
				t.Fatalf("expected metrics durations for operations, got empty")
			}
			if snapshot.Results["create_facility"]["success"] == 0 {
				t.Fatalf("expected create_facility success metric recorded: %+v", snapshot.Results)
			}
			if traceBuffer.Len() == 0 {
				t.Fatalf("expected trace exporter to emit spans")
			}
			var foundSpan bool
			for _, entry := range tracer.Entries() {
				if entry.Operation == "create_facility" && entry.Status == "success" {
					foundSpan = true
					break
				}
			}
			if !foundSpan {
				t.Fatalf("expected trace entry for create_facility, entries=%+v", tracer.Entries())
			}
		})
	}

	for _, bv := range blobVariants {
		t.Run(bv.name, func(t *testing.T) {
			bs := bv.open(t)
			key := "alpha/test.txt"
			payload := []byte("hello")
			info, err := bs.Put(ctx, key, bytes.NewReader(payload), blob.PutOptions{ContentType: "text/plain"})
			if err != nil {
				t.Fatalf("blob put: %v", err)
			}
			if info.Key != key {
				t.Fatalf("unexpected blob key info: %+v", info)
			}
			// Some adapters (mock S3) may report a transformed size (e.g., aws-chunked encoding simulation);
			// accept any non-zero size for smoke coverage instead of exact length equality.
			if info.Size <= 0 {
				t.Fatalf("expected positive blob size, got %d (info=%+v)", info.Size, info)
			}
			// Read it back
			_, rc, err := bs.Get(ctx, key)
			if err != nil {
				t.Fatalf("blob get: %v", err)
			}
			got := make([]byte, len(payload))
			if _, err := rc.Read(got); err != nil && err.Error() != "EOF" { // tolerate EOF sentinel
				// we purposefully avoid io.ReadAll to keep allocations tiny
				t.Fatalf("read payload: %v", err)
			}
			_ = rc.Close()
			if string(got) != string(payload) {
				t.Fatalf("payload mismatch got=%q want=%q", string(got), string(payload))
			}
			// Basic deletion for completeness
			if ok, err := bs.Delete(ctx, key); err != nil || !ok {
				t.Fatalf("blob delete: %v ok=%v", err, ok)
			}
		})
	}

	// Sanity: ensure no environment leakage (none set here, but guard for future edits)
	if os.Getenv("COLONYCORE_BLOB_DRIVER") != "" || os.Getenv("COLONYCORE_STORAGE_DRIVER") != "" {
		t.Fatalf("expected no test-induced env leakage")
	}
}
