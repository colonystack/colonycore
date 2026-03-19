// Package datasets provides dataset export scheduling and artifact management
// adapters used internally by the colonycore service.
package datasets

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"strings"
	"sync"
	"time"

	"colonycore/internal/observability"
	"colonycore/pkg/datasetapi"
)

// ExportStatus describes the lifecycle stage of an export request.
type ExportStatus string

// Possible export lifecycle statuses.
const (
	// ExportStatusQueued indicates the export request is queued for processing.
	ExportStatusQueued ExportStatus = "queued"
	// ExportStatusRunning indicates the worker is currently generating artifacts.
	ExportStatusRunning ExportStatus = "running"
	// ExportStatusSucceeded indicates all requested artifacts were generated successfully.
	ExportStatusSucceeded ExportStatus = "succeeded"
	// ExportStatusFailed indicates the export terminated with an error.
	ExportStatusFailed ExportStatus = "failed"
)

// ExportProgressState describes the finer-grained stage of export processing.
type ExportProgressState string

// Possible progress states for an export.
const (
	ExportProgressStateQueued                 ExportProgressState = "queued"
	ExportProgressStateValidatingParameters   ExportProgressState = "validating_parameters"
	ExportProgressStateExecutingTemplate      ExportProgressState = "executing_template"
	ExportProgressStateMaterializingArtifacts ExportProgressState = "materializing_artifacts"
	ExportProgressStateCompleted              ExportProgressState = "completed"
	ExportProgressStateFailed                 ExportProgressState = "failed"
)

// ExportArtifactReadiness describes whether export artifacts are available yet.
type ExportArtifactReadiness string

// Possible artifact readiness states.
const (
	ExportArtifactReadinessPending     ExportArtifactReadiness = "pending"
	ExportArtifactReadinessPartial     ExportArtifactReadiness = "partial"
	ExportArtifactReadinessReady       ExportArtifactReadiness = "ready"
	ExportArtifactReadinessUnavailable ExportArtifactReadiness = "unavailable"
)

const (
	exportProgressValidatePct        = 15
	exportProgressExecutePct         = 45
	exportProgressMaterializeBasePct = 70
)

// ExportArtifact captures a stored dataset artifact.
type ExportArtifact struct {
	ID          string            `json:"id"`
	Format      datasetapi.Format `json:"format"`
	ContentType string            `json:"content_type"`
	SizeBytes   int64             `json:"size_bytes"`
	URL         string            `json:"url"`
	Metadata    map[string]any    `json:"metadata,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
}

// ExportRecord tracks an export request and resulting artifacts.
type ExportRecord struct {
	ID                string                        `json:"id"`
	Template          datasetapi.TemplateDescriptor `json:"template"`
	Scope             datasetapi.Scope              `json:"scope"`
	Parameters        map[string]any                `json:"parameters"`
	Formats           []datasetapi.Format           `json:"formats"`
	Status            ExportStatus                  `json:"status"`
	ProgressPct       int                           `json:"progress_pct"`
	ETASeconds        *int                          `json:"eta_seconds"`
	ProgressState     ExportProgressState           `json:"progress_state"`
	ArtifactReadiness ExportArtifactReadiness       `json:"artifact_readiness"`
	Error             string                        `json:"error,omitempty"`
	Artifacts         []ExportArtifact              `json:"artifacts,omitempty"`
	RequestedBy       string                        `json:"requested_by"`
	Reason            string                        `json:"reason,omitempty"`
	ProjectID         string                        `json:"project_id,omitempty"`
	ProtocolID        string                        `json:"protocol_id,omitempty"`
	CreatedAt         time.Time                     `json:"created_at"`
	UpdatedAt         time.Time                     `json:"updated_at"`
	CompletedAt       *time.Time                    `json:"completed_at,omitempty"`
	StartedAt         *time.Time                    `json:"-"`
}

// ExportInput represents an enqueue request for the worker.
type ExportInput struct {
	TemplateSlug string
	Parameters   map[string]any
	Formats      []datasetapi.Format
	Scope        datasetapi.Scope
	RequestedBy  string
	ProjectID    string
	ProtocolID   string
	Reason       string
}

// ExportScheduler queues dataset export requests and exposes status.
type ExportScheduler interface {
	EnqueueExport(ctx context.Context, input ExportInput) (ExportRecord, error)
	GetExport(id string) (ExportRecord, bool)
}

// ObjectStore persists export artifacts.
type ObjectStore interface {
	// Put stores a new immutable object. Implementations SHOULD fail if key exists.
	Put(ctx context.Context, key string, payload []byte, contentType string, metadata map[string]any) (ExportArtifact, error)
	// Get returns the artifact metadata and full payload bytes.
	Get(ctx context.Context, key string) (ExportArtifact, []byte, error)
	// Delete removes the object; returns true if it existed. Idempotent.
	Delete(ctx context.Context, key string) (bool, error)
	// List returns artifacts whose IDs start with the provided prefix. Empty prefix lists all.
	List(ctx context.Context, prefix string) ([]ExportArtifact, error)
}

// AuditLogger records export audit entries.
type AuditLogger interface {
	Record(ctx context.Context, entry AuditEntry)
}

// AuditEntry captures audit trail metadata for exports.
type AuditEntry struct {
	ID         string           `json:"id"`
	Action     string           `json:"action"`
	Actor      string           `json:"actor"`
	Template   string           `json:"template"`
	Status     ExportStatus     `json:"status"`
	Scope      datasetapi.Scope `json:"scope"`
	Reason     string           `json:"reason,omitempty"`
	Metadata   map[string]any   `json:"metadata,omitempty"`
	OccurredAt time.Time        `json:"occurred_at"`
}

// Worker executes dataset exports asynchronously.
type Worker struct {
	catalog Catalog
	store   ObjectStore
	audit   AuditLogger
	events  observability.Recorder

	queue chan exportTask
	mu    sync.RWMutex
	jobs  map[string]*ExportRecord

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

type exportTask struct {
	id    string
	input ExportInput
}

type renderedArtifact struct {
	Artifact ExportArtifact
	Payload  []byte
}

// NewWorker constructs an export worker.
func NewWorker(c Catalog, store ObjectStore, audit AuditLogger) *Worker {
	ctx, cancel := context.WithCancel(context.Background())
	return &Worker{
		catalog: c,
		store:   store,
		audit:   audit,
		events:  observability.NoopRecorder{},
		queue:   make(chan exportTask, 32),
		jobs:    make(map[string]*ExportRecord),
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Start begins processing export requests.
func (w *Worker) Start() {
	w.wg.Add(1)
	go w.loop()
}

// Stop signals the worker to halt and waits for completion.
func (w *Worker) Stop(ctx context.Context) error {
	w.cancel()
	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (w *Worker) loop() {
	defer w.wg.Done()
	for {
		select {
		case <-w.ctx.Done():
			return
		case task := <-w.queue:
			w.process(task)
		}
	}
}

// EnqueueExport schedules an export job and returns the queued record.
func (w *Worker) EnqueueExport(ctx context.Context, input ExportInput) (ExportRecord, error) {
	formatProvider := datasetapi.GetFormatProvider()

	if w.catalog == nil {
		err := fmt.Errorf("export catalog not configured")
		w.emitExportEvent(ctx, "catalog.export.enqueue", observability.StatusError, "", input.TemplateSlug, err.Error(), 0, nil)
		return ExportRecord{}, err
	}

	slug := input.TemplateSlug
	if strings.TrimSpace(slug) == "" {
		err := fmt.Errorf("template slug required")
		w.emitExportEvent(ctx, "catalog.export.enqueue", observability.StatusError, "", slug, err.Error(), 0, nil)
		return ExportRecord{}, err
	}
	template, ok := w.catalog.ResolveDatasetTemplate(slug)
	if !ok {
		err := fmt.Errorf("dataset template %s not found", slug)
		w.emitExportEvent(ctx, "catalog.export.enqueue", observability.StatusError, "", slug, err.Error(), 0, nil)
		return ExportRecord{}, err
	}

	formats := input.Formats
	if len(formats) == 0 {
		formats = []datasetapi.Format{formatProvider.JSON(), formatProvider.CSV()}
	}
	uniqFormats := make([]datasetapi.Format, 0, len(formats))
	seen := make(map[datasetapi.Format]struct{})
	for _, format := range formats {
		if _, duplicate := seen[format]; duplicate {
			continue
		}
		if !template.SupportsFormat(format) {
			err := fmt.Errorf("format %s not supported by template", format)
			w.emitExportEvent(ctx, "catalog.export.enqueue", observability.StatusError, "", slug, err.Error(), 0, nil)
			return ExportRecord{}, err
		}
		uniqFormats = append(uniqFormats, format)
		seen[format] = struct{}{}
	}

	id := newID()
	now := time.Now().UTC()
	record := ExportRecord{
		ID:                id,
		Template:          template.Descriptor(),
		Scope:             input.Scope,
		Parameters:        cloneMap(input.Parameters),
		Formats:           uniqFormats,
		Status:            ExportStatusQueued,
		ProgressPct:       0,
		ProgressState:     ExportProgressStateQueued,
		ArtifactReadiness: ExportArtifactReadinessPending,
		RequestedBy:       input.RequestedBy,
		Reason:            input.Reason,
		ProjectID:         input.ProjectID,
		ProtocolID:        input.ProtocolID,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	w.mu.Lock()
	w.jobs[id] = &record
	queuedSnapshot := record.copy()
	w.mu.Unlock()

	if w.audit != nil {
		w.audit.Record(ctx, AuditEntry{
			ID:         newID(),
			Action:     "dataset_export",
			Actor:      input.RequestedBy,
			Template:   slug,
			Status:     ExportStatusQueued,
			Scope:      input.Scope,
			Reason:     input.Reason,
			OccurredAt: now,
		})
	}

	select {
	case w.queue <- exportTask{id: id, input: input}:
	default:
		err := fmt.Errorf("export queue full")
		w.emitExportEvent(ctx, "catalog.export.enqueue", observability.StatusError, id, slug, err.Error(), 0, nil)
		return ExportRecord{}, err
	}
	w.emitExportEvent(ctx, "catalog.export.enqueue", observability.StatusQueued, id, slug, "", 0, map[string]float64{
		"formats_total": float64(len(uniqFormats)),
	})

	return queuedSnapshot, nil
}

// GetExport returns a snapshot of the export record.
func (w *Worker) GetExport(id string) (ExportRecord, bool) {
	w.mu.RLock()
	record, ok := w.jobs[id]
	if !ok {
		w.mu.RUnlock()
		return ExportRecord{}, false
	}
	snapshot := record.copy()
	w.mu.RUnlock()
	return snapshot, true
}

func (w *Worker) process(task exportTask) {
	formatProvider := datasetapi.GetFormatProvider()
	started := time.Now()

	record := w.snapshot(task.id)
	if record == nil {
		return
	}

	template, ok := w.catalog.ResolveDatasetTemplate(task.input.TemplateSlug)
	if !ok {
		w.fail(task.id, fmt.Sprintf("template %s missing", task.input.TemplateSlug), time.Since(started))
		return
	}

	w.updateStatus(task.id, ExportStatusRunning, "")
	w.setProgress(task.id, ExportProgressStateValidatingParameters, exportProgressValidatePct)

	cleaned, errs := template.ValidateParameters(task.input.Parameters)
	if len(errs) > 0 {
		w.fail(task.id, fmt.Sprintf("parameter validation failed: %v", errs), time.Since(started))
		return
	}

	w.setProgress(task.id, ExportProgressStateExecutingTemplate, exportProgressExecutePct)
	result, paramErrs, err := template.Run(w.ctx, cleaned, task.input.Scope, formatProvider.JSON())
	if err != nil {
		w.fail(task.id, fmt.Sprintf("dataset run failed: %v", err), time.Since(started))
		return
	}
	if len(paramErrs) > 0 {
		w.fail(task.id, fmt.Sprintf("parameter validation failed: %v", paramErrs), time.Since(started))
		return
	}

	exportArtifacts := make([]ExportArtifact, 0, len(record.Formats))
	w.setProgress(task.id, ExportProgressStateMaterializingArtifacts, exportProgressMaterializeBasePct)
	for _, format := range record.Formats {
		rendered, err := w.materialize(format, template, result)
		if err != nil {
			w.fail(task.id, err.Error(), time.Since(started))
			return
		}
		if w.store != nil {
			stored, err := w.store.Put(w.ctx, rendered.Artifact.ID, rendered.Payload, rendered.Artifact.ContentType, rendered.Artifact.Metadata)
			if err != nil {
				w.fail(task.id, fmt.Sprintf("store artifact failed: %v", err), time.Since(started))
				return
			}
			stored.Format = rendered.Artifact.Format
			if stored.ContentType == "" {
				stored.ContentType = rendered.Artifact.ContentType
			}
			if stored.SizeBytes == 0 {
				stored.SizeBytes = rendered.Artifact.SizeBytes
			}
			if stored.CreatedAt.IsZero() {
				stored.CreatedAt = rendered.Artifact.CreatedAt
			}
			stored.Metadata = mergeMetadata(rendered.Artifact.Metadata, stored.Metadata)
			exportArtifacts = append(exportArtifacts, stored)
			w.appendArtifact(task.id, stored, len(record.Formats))
		} else {
			exportArtifacts = append(exportArtifacts, rendered.Artifact)
			w.appendArtifact(task.id, rendered.Artifact, len(record.Formats))
		}
	}

	w.complete(task.id, exportArtifacts, time.Since(started))
}

func (w *Worker) snapshot(id string) *ExportRecord {
	w.mu.RLock()
	record, ok := w.jobs[id]
	w.mu.RUnlock()
	if !ok {
		return nil
	}
	return record
}

func (w *Worker) updateStatus(id string, status ExportStatus, message string) {
	now := time.Now().UTC()
	w.mu.Lock()
	if record, ok := w.jobs[id]; ok {
		record.Status = status
		record.Error = message
		record.UpdatedAt = now
		if status == ExportStatusRunning && record.StartedAt == nil {
			startedAt := now
			record.StartedAt = &startedAt
		}
		refreshExportProgress(record, now)
	}
	w.mu.Unlock()
	if w.audit != nil {
		w.audit.Record(w.ctx, AuditEntry{
			ID:         newID(),
			Action:     "dataset_export",
			Actor:      w.actorFor(id),
			Template:   w.templateFor(id),
			Status:     status,
			Scope:      w.scopeFor(id),
			Metadata:   map[string]any{"note": message},
			OccurredAt: now,
		})
	}
	w.emitExportEvent(w.ctx, "catalog.export.process", exportStatusToEventStatus(status), id, w.templateFor(id), message, 0, nil)
}

func (w *Worker) setProgress(id string, state ExportProgressState, progressPct int) {
	now := time.Now().UTC()
	w.mu.Lock()
	if record, ok := w.jobs[id]; ok {
		record.ProgressState = state
		record.ProgressPct = clampProgressPct(progressPct)
		record.UpdatedAt = now
		refreshExportProgress(record, now)
	}
	w.mu.Unlock()
}

func (w *Worker) appendArtifact(id string, artifact ExportArtifact, totalFormats int) {
	now := time.Now().UTC()
	w.mu.Lock()
	if record, ok := w.jobs[id]; ok {
		record.Artifacts = append(record.Artifacts, artifact)
		record.ProgressState = ExportProgressStateMaterializingArtifacts
		record.ProgressPct = materializationProgressPct(totalFormats, len(record.Artifacts))
		record.UpdatedAt = now
		refreshExportProgress(record, now)
	}
	w.mu.Unlock()
}

func (w *Worker) complete(id string, artifacts []ExportArtifact, duration time.Duration) {
	now := time.Now().UTC()
	w.mu.Lock()
	if record, ok := w.jobs[id]; ok {
		record.Status = ExportStatusSucceeded
		record.Error = ""
		record.ProgressPct = 100
		record.ProgressState = ExportProgressStateCompleted
		record.Artifacts = cloneArtifacts(artifacts)
		record.UpdatedAt = now
		record.CompletedAt = &now
		refreshExportProgress(record, now)
	}
	w.mu.Unlock()
	if w.audit != nil {
		w.audit.Record(w.ctx, AuditEntry{
			ID:         newID(),
			Action:     "dataset_export",
			Actor:      w.actorFor(id),
			Template:   w.templateFor(id),
			Status:     ExportStatusSucceeded,
			Scope:      w.scopeFor(id),
			OccurredAt: now,
		})
	}
	w.emitExportEvent(w.ctx, "catalog.export.process", observability.StatusSuccess, id, w.templateFor(id), "", duration, map[string]float64{
		"artifacts_total": float64(len(artifacts)),
	})
}

func (w *Worker) fail(id, reason string, duration time.Duration) {
	now := time.Now().UTC()
	w.mu.Lock()
	if record, ok := w.jobs[id]; ok {
		record.Status = ExportStatusFailed
		record.Error = reason
		record.ProgressState = ExportProgressStateFailed
		record.UpdatedAt = now
		record.CompletedAt = &now
		refreshExportProgress(record, now)
	}
	w.mu.Unlock()
	if w.audit != nil {
		w.audit.Record(w.ctx, AuditEntry{
			ID:         newID(),
			Action:     "dataset_export",
			Actor:      w.actorFor(id),
			Template:   w.templateFor(id),
			Status:     ExportStatusFailed,
			Scope:      w.scopeFor(id),
			Metadata:   map[string]any{"error": reason},
			OccurredAt: now,
		})
	}
	w.emitExportEvent(w.ctx, "catalog.export.process", observability.StatusError, id, w.templateFor(id), reason, duration, nil)
}

func (w *Worker) actorFor(id string) string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	if record, ok := w.jobs[id]; ok {
		return record.RequestedBy
	}
	return ""
}

func (w *Worker) templateFor(id string) string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	if record, ok := w.jobs[id]; ok {
		return record.Template.Slug
	}
	return ""
}

func (w *Worker) scopeFor(id string) datasetapi.Scope {
	w.mu.RLock()
	defer w.mu.RUnlock()
	if record, ok := w.jobs[id]; ok {
		return record.Scope
	}
	return datasetapi.Scope{}
}

func (w *Worker) emitExportEvent(ctx context.Context, name, status, exportID, templateID, errMessage string, duration time.Duration, measures map[string]float64) {
	if w == nil || w.events == nil {
		return
	}
	labels := map[string]string{}
	if exportID != "" {
		labels["export_id"] = exportID
	}
	if templateID != "" {
		labels["template_id"] = templateID
	}
	w.events.Record(ctx, observability.Event{
		Category:   observability.CategoryCatalogOperation,
		Name:       name,
		Status:     status,
		DurationMS: observability.DurationMS(duration),
		Error:      errMessage,
		Labels:     labels,
		Measures:   measures,
	})
}

func exportStatusToEventStatus(status ExportStatus) string {
	switch status {
	case ExportStatusQueued:
		return observability.StatusQueued
	case ExportStatusRunning:
		return observability.StatusRunning
	case ExportStatusSucceeded:
		return observability.StatusSuccess
	case ExportStatusFailed:
		return observability.StatusError
	default:
		return observability.StatusError
	}
}

func (w *Worker) materialize(format datasetapi.Format, template datasetapi.TemplateRuntime, result datasetapi.RunResult) (renderedArtifact, error) {
	formatProvider := datasetapi.GetFormatProvider()

	descriptor := template.Descriptor()
	switch format {
	case formatProvider.JSON():
		payload, err := json.Marshal(result)
		if err != nil {
			return renderedArtifact{}, fmt.Errorf("marshal json: %w", err)
		}
		return renderedArtifact{
			Artifact: ExportArtifact{
				ID:          newID(),
				Format:      formatProvider.JSON(),
				ContentType: "application/json",
				SizeBytes:   int64(len(payload)),
				Metadata: map[string]any{
					"rows": len(result.Rows),
				},
				CreatedAt: time.Now().UTC(),
			},
			Payload: payload,
		}, nil
	case formatProvider.CSV():
		buf := &bytes.Buffer{}
		writer := csv.NewWriter(buf)
		columns := result.Schema
		if len(columns) == 0 {
			columns = descriptor.Columns
		}
		headers := make([]string, len(columns))
		for i, column := range columns {
			headers[i] = column.Name
		}
		if err := writer.Write(headers); err != nil {
			return renderedArtifact{}, err
		}
		for _, row := range result.Rows {
			record := make([]string, len(columns))
			for i, column := range columns {
				record[i] = formatValue(row[column.Name])
			}
			if err := writer.Write(record); err != nil {
				return renderedArtifact{}, err
			}
		}
		writer.Flush()
		if err := writer.Error(); err != nil {
			return renderedArtifact{}, err
		}
		payload := buf.Bytes()
		return renderedArtifact{
			Artifact: ExportArtifact{
				ID:          newID(),
				Format:      formatProvider.CSV(),
				ContentType: "text/csv",
				SizeBytes:   int64(len(payload)),
				Metadata: map[string]any{
					"rows": len(result.Rows),
				},
				CreatedAt: time.Now().UTC(),
			},
			Payload: payload,
		}, nil
	case formatProvider.HTML():
		payload := buildHTML(descriptor, result)
		return renderedArtifact{
			Artifact: ExportArtifact{
				ID:          newID(),
				Format:      formatProvider.HTML(),
				ContentType: "text/html",
				SizeBytes:   int64(len(payload)),
				Metadata:    map[string]any{"rows": len(result.Rows)},
				CreatedAt:   time.Now().UTC(),
			},
			Payload: payload,
		}, nil
	case formatProvider.Parquet():
		payload, err := json.Marshal(result.Rows)
		if err != nil {
			return renderedArtifact{}, fmt.Errorf("marshall parquet surrogate: %w", err)
		}
		return renderedArtifact{
			Artifact: ExportArtifact{
				ID:          newID(),
				Format:      formatProvider.Parquet(),
				ContentType: "application/octet-stream",
				SizeBytes:   int64(len(payload)),
				Metadata: map[string]any{
					"note": "parquet placeholder encodes rows as JSON; replace with true writer",
					"rows": len(result.Rows),
				},
				CreatedAt: time.Now().UTC(),
			},
			Payload: payload,
		}, nil
	case formatProvider.PNG():
		payload, err := buildPNG(result)
		if err != nil {
			return renderedArtifact{}, err
		}
		return renderedArtifact{
			Artifact: ExportArtifact{
				ID:          newID(),
				Format:      formatProvider.PNG(),
				ContentType: "image/png",
				SizeBytes:   int64(len(payload)),
				Metadata:    map[string]any{"rows": len(result.Rows)},
				CreatedAt:   time.Now().UTC(),
			},
			Payload: payload,
		}, nil
	default:
		return renderedArtifact{}, fmt.Errorf("unsupported export format %s", format)
	}
}

func buildHTML(descriptor datasetapi.TemplateDescriptor, result datasetapi.RunResult) []byte {
	columns := result.Schema
	if len(columns) == 0 {
		columns = descriptor.Columns
	}
	buf := &strings.Builder{}
	buf.WriteString("<!DOCTYPE html><html><head><meta charset=\"utf-8\"><title>")
	buf.WriteString(descriptor.Title)
	buf.WriteString("</title></head><body><table>")
	buf.WriteString("<thead><tr>")
	for _, column := range columns {
		buf.WriteString("<th>")
		buf.WriteString(column.Name)
		buf.WriteString("</th>")
	}
	buf.WriteString("</tr></thead><tbody>")
	for _, row := range result.Rows {
		buf.WriteString("<tr>")
		for _, column := range columns {
			buf.WriteString("<td>")
			buf.WriteString(formatValue(row[column.Name]))
			buf.WriteString("</td>")
		}
		buf.WriteString("</tr>")
	}
	buf.WriteString("</tbody></table></body></html>")
	return []byte(buf.String())
}

func buildPNG(result datasetapi.RunResult) ([]byte, error) {
	width := 400
	height := 200
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)

	rowCount := len(result.Rows)
	if rowCount == 0 {
		rowCount = 1
	}
	barWidth := width / rowCount
	if barWidth < 1 {
		barWidth = 1
	}
	for i := 0; i < len(result.Rows); i++ {
		x0 := i * barWidth
		x1 := x0 + barWidth - 2
		if x1 <= x0 {
			x1 = x0 + 1
		}
		heightFactor := int(float64(height-20) * 0.6)
		y0 := height - heightFactor
		y1 := height - 10
		rect := image.Rect(x0, y0, x1, y1)
		draw.Draw(img, rect, &image.Uniform{color.RGBA{0, 102, 204, 255}}, image.Point{}, draw.Src)
	}
	buf := &bytes.Buffer{}
	if err := png.Encode(buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func mergeMetadata(base map[string]any, extra map[string]any) map[string]any {
	if len(base) == 0 && len(extra) == 0 {
		return nil
	}
	out := make(map[string]any, len(base)+len(extra))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range extra {
		out[k] = v
	}
	return out
}

func (r ExportRecord) copy() ExportRecord {
	dup := r
	dup.Parameters = cloneMap(r.Parameters)
	dup.Formats = append([]datasetapi.Format(nil), r.Formats...)
	dup.Artifacts = cloneArtifacts(r.Artifacts)
	dup.ETASeconds = cloneIntPointer(r.ETASeconds)
	dup.CompletedAt = cloneTimePointer(r.CompletedAt)
	dup.StartedAt = cloneTimePointer(r.StartedAt)
	return dup
}

func cloneArtifacts(in []ExportArtifact) []ExportArtifact {
	if len(in) == 0 {
		return nil
	}
	out := make([]ExportArtifact, len(in))
	copy(out, in)
	for i := range out {
		out[i].Metadata = cloneMap(out[i].Metadata)
	}
	return out
}

func cloneIntPointer(in *int) *int {
	if in == nil {
		return nil
	}
	value := *in
	return &value
}

func cloneTimePointer(in *time.Time) *time.Time {
	if in == nil {
		return nil
	}
	value := *in
	return &value
}

func refreshExportProgress(record *ExportRecord, now time.Time) {
	if record == nil {
		return
	}
	record.ProgressPct = clampProgressPct(record.ProgressPct)
	record.ArtifactReadiness = deriveArtifactReadiness(*record)
	record.ETASeconds = estimateETASeconds(record.Status, record.StartedAt, record.ProgressPct, now)
}

func clampProgressPct(progressPct int) int {
	switch {
	case progressPct < 0:
		return 0
	case progressPct > 100:
		return 100
	default:
		return progressPct
	}
}

func estimateETASeconds(status ExportStatus, startedAt *time.Time, progressPct int, now time.Time) *int {
	if status != ExportStatusRunning || startedAt == nil || progressPct <= 0 || progressPct >= 100 {
		return nil
	}
	elapsedSeconds := int(now.Sub(*startedAt).Seconds())
	if elapsedSeconds < 0 {
		elapsedSeconds = 0
	}
	remaining := (elapsedSeconds*(100-progressPct) + progressPct - 1) / progressPct
	return &remaining
}

func materializationProgressPct(totalFormats, readyArtifacts int) int {
	if totalFormats <= 0 {
		return exportProgressMaterializeBasePct
	}
	return clampProgressPct(exportProgressMaterializeBasePct + (readyArtifacts*(100-exportProgressMaterializeBasePct))/(totalFormats+1))
}

func deriveArtifactReadiness(record ExportRecord) ExportArtifactReadiness {
	readyArtifacts := len(record.Artifacts)
	totalFormats := len(record.Formats)

	switch {
	case readyArtifacts == 0 && record.Status == ExportStatusFailed:
		return ExportArtifactReadinessUnavailable
	case readyArtifacts == 0:
		return ExportArtifactReadinessPending
	// Failed exports may still expose persisted artifacts from earlier formats, so
	// preserve partial/ready readiness when materialization made useful output available.
	case totalFormats > 0 && readyArtifacts < totalFormats:
		return ExportArtifactReadinessPartial
	case totalFormats > 0:
		return ExportArtifactReadinessReady
	case record.Status == ExportStatusFailed:
		return ExportArtifactReadinessUnavailable
	default:
		return ExportArtifactReadinessPending
	}
}

func cloneMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func newID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return fmt.Sprintf("%x", b[:])
}

// MemoryObjectStore is an in-memory implementation of ObjectStore for tests.
type MemoryObjectStore struct {
	mu      sync.RWMutex
	objects map[string]storedObject
}

type storedObject struct {
	artifact ExportArtifact
	payload  []byte
}

// NewMemoryObjectStore constructs an in-memory object store.
func NewMemoryObjectStore() *MemoryObjectStore {
	return &MemoryObjectStore{objects: make(map[string]storedObject)}
}

// Put stores payload metadata and returns a signed URL for retrieval.
func (s *MemoryObjectStore) Put(_ context.Context, key string, payload []byte, contentType string, metadata map[string]any) (ExportArtifact, error) {
	now := time.Now().UTC()
	s.mu.Lock()
	if _, exists := s.objects[key]; exists {
		s.mu.Unlock()
		return ExportArtifact{}, fmt.Errorf("object %s already exists", key)
	}
	artifact := ExportArtifact{
		ID:          key,
		ContentType: contentType,
		SizeBytes:   int64(len(payload)),
		Metadata:    cloneMap(metadata),
		CreatedAt:   now,
		URL:         fmt.Sprintf("https://object-store.local/%s?token=stub", key),
	}
	// store defensive copy of payload
	cp := make([]byte, len(payload))
	copy(cp, payload)
	s.objects[key] = storedObject{artifact: artifact, payload: cp}
	s.mu.Unlock()
	return artifact, nil
}

// Get retrieves an object payload and its metadata; returns error if key not found.
func (s *MemoryObjectStore) Get(_ context.Context, key string) (ExportArtifact, []byte, error) {
	s.mu.RLock()
	obj, ok := s.objects[key]
	s.mu.RUnlock()
	if !ok {
		return ExportArtifact{}, nil, fmt.Errorf("object %s not found", key)
	}
	payloadCopy := make([]byte, len(obj.payload))
	copy(payloadCopy, obj.payload)
	artCopy := obj.artifact
	if artCopy.Metadata != nil {
		artCopy.Metadata = cloneMap(artCopy.Metadata)
	}
	return artCopy, payloadCopy, nil
}

// Delete removes an object if it exists returning whether it was present.
func (s *MemoryObjectStore) Delete(_ context.Context, key string) (bool, error) {
	s.mu.Lock()
	_, existed := s.objects[key]
	if existed {
		delete(s.objects, key)
	}
	s.mu.Unlock()
	return existed, nil
}

// List returns artifacts whose IDs share the provided prefix (or all when prefix empty).
func (s *MemoryObjectStore) List(_ context.Context, prefix string) ([]ExportArtifact, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]ExportArtifact, 0, len(s.objects))
	for key, obj := range s.objects {
		if prefix == "" || strings.HasPrefix(key, prefix) {
			artCopy := obj.artifact
			if artCopy.Metadata != nil {
				artCopy.Metadata = cloneMap(artCopy.Metadata)
			}
			out = append(out, artCopy)
		}
	}
	return out, nil
}

// Objects returns stored artifacts for inspection in tests.
func (s *MemoryObjectStore) Objects() []ExportArtifact { // legacy test helper
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]ExportArtifact, 0, len(s.objects))
	for _, obj := range s.objects {
		artCopy := obj.artifact
		if artCopy.Metadata != nil {
			artCopy.Metadata = cloneMap(artCopy.Metadata)
		}
		out = append(out, artCopy)
	}
	return out
}

// MemoryAuditLog captures audit entries in-memory for assertions.
type MemoryAuditLog struct {
	mu      sync.Mutex
	entries []AuditEntry
}

// Record stores an audit entry.
func (l *MemoryAuditLog) Record(_ context.Context, entry AuditEntry) {
	l.mu.Lock()
	l.entries = append(l.entries, entry)
	l.mu.Unlock()
}

// Entries returns a defensive copy of recorded audit entries.
func (l *MemoryAuditLog) Entries() []AuditEntry {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]AuditEntry, len(l.entries))
	copy(out, l.entries)
	return out
}
