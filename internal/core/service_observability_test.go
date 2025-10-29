package core

import (
	"bytes"
	"context"
	"expvar"
	"strings"
	"testing"
	"time"

	"colonycore/pkg/domain"
)

type captureAuditRecorder struct {
	entries []AuditEntry
}

func (c *captureAuditRecorder) Record(_ context.Context, entry AuditEntry) {
	c.entries = append(c.entries, entry)
}

func (c *captureAuditRecorder) has(op string, status AuditStatus, predicate func(AuditEntry) bool) bool {
	for _, entry := range c.entries {
		if entry.Operation == op && entry.Status == status {
			if predicate == nil || predicate(entry) {
				return true
			}
		}
	}
	return false
}

type metricsCall struct {
	op       string
	success  bool
	duration time.Duration
}

type captureMetricsRecorder struct {
	calls []metricsCall
}

func (c *captureMetricsRecorder) Observe(_ context.Context, op string, success bool, duration time.Duration) {
	c.calls = append(c.calls, metricsCall{op: op, success: success, duration: duration})
}

func (c *captureMetricsRecorder) has(op string, success bool) bool {
	for _, call := range c.calls {
		if call.op == op && call.success == success {
			return true
		}
	}
	return false
}

type captureTracer struct {
	started []string
	ended   []spanRecord
}

type spanRecord struct {
	op  string
	err error
}

func (c *captureTracer) Start(ctx context.Context, op string) (context.Context, TraceSpan) {
	c.started = append(c.started, op)
	return ctx, &captureSpan{tracer: c, op: op}
}

func (c *captureTracer) has(op string, success bool) bool {
	for _, record := range c.ended {
		if record.op == op {
			if success && record.err == nil {
				return true
			}
			if !success && record.err != nil {
				return true
			}
		}
	}
	return false
}

type captureSpan struct {
	tracer *captureTracer
	op     string
}

func (s *captureSpan) End(err error) {
	s.tracer.ended = append(s.tracer.ended, spanRecord{op: s.op, err: err})
}

func TestServiceObservabilityComplianceEntities(t *testing.T) {
	ctx := context.Background()
	audit := &captureAuditRecorder{}
	metrics := &captureMetricsRecorder{}
	tracer := &captureTracer{}

	svc := NewInMemoryService(NewRulesEngine(),
		WithAuditRecorder(audit),
		WithMetricsRecorder(metrics),
		WithTracer(tracer),
	)

	const updatedDesc = "updated"

	facility, _, err := svc.CreateFacility(ctx, domain.Facility{Name: "Main Facility"})
	if err != nil {
		t.Fatalf("create facility: %v", err)
	}
	if !audit.has("create_facility", AuditStatusSuccess, func(entry AuditEntry) bool { return entry.EntityID == facility.ID }) {
		t.Fatalf("expected audit entry for create_facility success")
	}

	if _, _, err := svc.UpdateFacility(ctx, facility.ID, func(f *domain.Facility) error {
		f.Zone = "Zone-A"
		return nil
	}); err != nil {
		t.Fatalf("update facility: %v", err)
	}
	if !audit.has("update_facility", AuditStatusSuccess, nil) {
		t.Fatalf("expected audit entry for update_facility success")
	}

	if _, err := svc.DeleteFacility(ctx, "missing-facility"); err == nil {
		t.Fatalf("expected delete_facility error for missing id")
	}
	if !audit.has("delete_facility", AuditStatusError, nil) {
		t.Fatalf("expected audit error entry for delete_facility")
	}
	if !metrics.has("delete_facility", false) {
		t.Fatalf("expected metrics entry for failed delete_facility")
	}
	if !tracer.has("delete_facility", false) {
		t.Fatalf("expected trace span for failed delete_facility")
	}

	protocol, _, err := svc.CreateProtocol(ctx, domain.Protocol{Code: "PR-1", Title: "Protocol", MaxSubjects: 5})
	if err != nil {
		t.Fatalf("create protocol: %v", err)
	}
	procedure, _, err := svc.CreateProcedure(ctx, domain.Procedure{
		Name:        "Procedure",
		Status:      domain.ProcedureStatusScheduled,
		ScheduledAt: time.Now().UTC(),
		ProtocolID:  protocol.ID,
	})
	if err != nil {
		t.Fatalf("create procedure: %v", err)
	}

	treatment, _, err := svc.CreateTreatment(ctx, domain.Treatment{
		Name:        "Treatment",
		ProcedureID: procedure.ID,
	})
	if err != nil {
		t.Fatalf("create treatment: %v", err)
	}
	if !audit.has("create_treatment", AuditStatusSuccess, func(entry AuditEntry) bool { return entry.EntityID == treatment.ID }) {
		t.Fatalf("expected audit entry for create_treatment")
	}
	if _, _, err := svc.UpdateTreatment(ctx, treatment.ID, func(t *domain.Treatment) error {
		t.DosagePlan = updatedDesc
		return nil
	}); err != nil {
		t.Fatalf("update treatment: %v", err)
	}

	observationInput := domain.Observation{
		ProcedureID: &procedure.ID,
		RecordedAt:  time.Now().UTC(),
		Observer:    "tech",
	}
	observationInput.SetData(map[string]any{"key": "value"})
	obs, _, err := svc.CreateObservation(ctx, observationInput)
	if err != nil {
		t.Fatalf("create observation: %v", err)
	}
	if _, _, err := svc.UpdateObservation(ctx, obs.ID, func(o *domain.Observation) error {
		o.Notes = strPtr(updatedDesc)
		return nil
	}); err != nil {
		t.Fatalf("update observation: %v", err)
	}

	org, _, err := svc.CreateOrganism(ctx, domain.Organism{Name: "Org", Species: "Frog"})
	if err != nil {
		t.Fatalf("create organism: %v", err)
	}
	sample, _, err := svc.CreateSample(ctx, domain.Sample{
		Identifier:      "S-1",
		SourceType:      "blood",
		OrganismID:      &org.ID,
		FacilityID:      facility.ID,
		CollectedAt:     time.Now().UTC(),
		Status:          domain.SampleStatusStored,
		StorageLocation: "Freezer-1",
		AssayType:       "PCR",
	})
	if err != nil {
		t.Fatalf("create sample: %v", err)
	}
	if _, _, err := svc.UpdateSample(ctx, sample.ID, func(s *domain.Sample) error {
		s.Status = domain.SampleStatusDisposed
		return nil
	}); err != nil {
		t.Fatalf("update sample: %v", err)
	}

	permit, _, err := svc.CreatePermit(ctx, domain.Permit{
		PermitNumber: "PER-1",
		Authority:    "Regulator",
		ValidFrom:    time.Now().UTC(),
		ValidUntil:   time.Now().UTC().Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("create permit: %v", err)
	}
	if _, _, err := svc.UpdatePermit(ctx, permit.ID, func(p *domain.Permit) error {
		p.Notes = strPtr(updatedDesc)
		return nil
	}); err != nil {
		t.Fatalf("update permit: %v", err)
	}

	item, _, err := svc.CreateSupplyItem(ctx, domain.SupplyItem{
		SKU:            "SKU-1",
		Name:           "Supply",
		QuantityOnHand: 10,
		FacilityIDs:    []string{facility.ID},
	})
	if err != nil {
		t.Fatalf("create supply item: %v", err)
	}
	if _, _, err := svc.UpdateSupplyItem(ctx, item.ID, func(s *domain.SupplyItem) error {
		s.QuantityOnHand = 8
		return nil
	}); err != nil {
		t.Fatalf("update supply item: %v", err)
	}

	if _, err := svc.DeleteObservation(ctx, obs.ID); err != nil {
		t.Fatalf("delete observation: %v", err)
	}
	if _, err := svc.DeleteTreatment(ctx, treatment.ID); err != nil {
		t.Fatalf("delete treatment: %v", err)
	}
	if _, err := svc.DeleteSample(ctx, sample.ID); err != nil {
		t.Fatalf("delete sample: %v", err)
	}
	if _, err := svc.DeletePermit(ctx, permit.ID); err != nil {
		t.Fatalf("delete permit: %v", err)
	}
	if _, err := svc.DeleteSupplyItem(ctx, item.ID); err != nil {
		t.Fatalf("delete supply item: %v", err)
	}
	if _, err := svc.DeleteFacility(ctx, facility.ID); err != nil {
		t.Fatalf("delete facility success: %v", err)
	}

	successOps := []string{
		"create_facility",
		"update_facility",
		"delete_facility",
		"create_treatment",
		"update_treatment",
		"delete_treatment",
		"create_observation",
		"update_observation",
		"delete_observation",
		"create_sample",
		"update_sample",
		"delete_sample",
		"create_permit",
		"update_permit",
		"delete_permit",
		"create_supply_item",
		"update_supply_item",
		"delete_supply_item",
	}

	for _, op := range successOps {
		if !metrics.has(op, true) {
			t.Fatalf("expected metrics success entry for %s", op)
		}
		if !tracer.has(op, true) {
			t.Fatalf("expected finished span for %s", op)
		}
		if !audit.has(op, AuditStatusSuccess, nil) {
			t.Fatalf("expected audit success entry for %s", op)
		}
	}
}

const entryStatusSuccess = "success"
const entryStatusError = "error"

func TestExpvarMetricsRecorderExports(t *testing.T) {
	recorder := NewExpvarMetricsRecorder("")
	if recorder.Name() == "" {
		t.Fatalf("expected recorder to have export name")
	}
	recorder.Observe(context.Background(), "test_op", true, 10*time.Millisecond)
	recorder.Observe(context.Background(), "test_op", false, 5*time.Millisecond)

	snapshot := recorder.Snapshot()
	if snapshot.DurationsMS["test_op"] <= 0 {
		t.Fatalf("expected positive duration, snapshot=%+v", snapshot)
	}
	if snapshot.Results["test_op"][entryStatusSuccess] != 1 || snapshot.Results["test_op"][entryStatusError] != 1 {
		t.Fatalf("unexpected results snapshot=%+v", snapshot)
	}

	if v := expvar.Get(recorder.Name()); v == nil {
		t.Fatalf("expected expvar export to be registered")
	} else if !strings.Contains(v.String(), "test_op") {
		t.Fatalf("expected expvar output to contain operation: %s", v.String())
	}
}

func TestJSONTraceTracerExports(t *testing.T) {
	var buf bytes.Buffer
	tracer := NewJSONTracer(&buf)
	_, span := tracer.Start(context.Background(), "trace_op")
	span.End(nil)

	entries := tracer.Entries()
	if len(entries) != 1 {
		t.Fatalf("expected single span entry, got %d", len(entries))
	}
	if entries[0].Operation != "trace_op" || entries[0].Status != entryStatusSuccess {
		t.Fatalf("unexpected span entry: %+v", entries[0])
	}
	if !strings.Contains(buf.String(), "\"operation\":\"trace_op\"") {
		t.Fatalf("expected JSON output to contain operation: %q", buf.String())
	}
}
