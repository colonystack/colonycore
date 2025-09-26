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

	"colonycore/internal/core"
)

// ExportStatus describes the lifecycle stage of an export request.
type ExportStatus string

const (
	ExportStatusQueued    ExportStatus = "queued"
	ExportStatusRunning   ExportStatus = "running"
	ExportStatusSucceeded ExportStatus = "succeeded"
	ExportStatusFailed    ExportStatus = "failed"
)

// ExportArtifact captures a stored dataset artifact.
type ExportArtifact struct {
	ID          string             `json:"id"`
	Format      core.DatasetFormat `json:"format"`
	ContentType string             `json:"content_type"`
	SizeBytes   int64              `json:"size_bytes"`
	URL         string             `json:"url"`
	Metadata    map[string]any     `json:"metadata,omitempty"`
	CreatedAt   time.Time          `json:"created_at"`
}

// ExportRecord tracks an export request and resulting artifacts.
type ExportRecord struct {
	ID          string                         `json:"id"`
	Template    core.DatasetTemplateDescriptor `json:"template"`
	Scope       core.DatasetScope              `json:"scope"`
	Parameters  map[string]any                 `json:"parameters"`
	Formats     []core.DatasetFormat           `json:"formats"`
	Status      ExportStatus                   `json:"status"`
	Error       string                         `json:"error,omitempty"`
	Artifacts   []ExportArtifact               `json:"artifacts,omitempty"`
	RequestedBy string                         `json:"requested_by"`
	Reason      string                         `json:"reason,omitempty"`
	ProjectID   string                         `json:"project_id,omitempty"`
	ProtocolID  string                         `json:"protocol_id,omitempty"`
	CreatedAt   time.Time                      `json:"created_at"`
	UpdatedAt   time.Time                      `json:"updated_at"`
	CompletedAt *time.Time                     `json:"completed_at,omitempty"`
}

// ExportInput represents an enqueue request for the worker.
type ExportInput struct {
	TemplateSlug string
	Parameters   map[string]any
	Formats      []core.DatasetFormat
	Scope        core.DatasetScope
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
	ID         string            `json:"id"`
	Action     string            `json:"action"`
	Actor      string            `json:"actor"`
	Template   string            `json:"template"`
	Status     ExportStatus      `json:"status"`
	Scope      core.DatasetScope `json:"scope"`
	Reason     string            `json:"reason,omitempty"`
	Metadata   map[string]any    `json:"metadata,omitempty"`
	OccurredAt time.Time         `json:"occurred_at"`
}

// Worker executes dataset exports asynchronously.
type Worker struct {
	catalog Catalog
	store   ObjectStore
	audit   AuditLogger

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
	if w.catalog == nil {
		return ExportRecord{}, fmt.Errorf("export catalog not configured")
	}

	slug := input.TemplateSlug
	if strings.TrimSpace(slug) == "" {
		return ExportRecord{}, fmt.Errorf("template slug required")
	}
	template, ok := w.catalog.ResolveDatasetTemplate(slug)
	if !ok {
		return ExportRecord{}, fmt.Errorf("dataset template %s not found", slug)
	}

	formats := input.Formats
	if len(formats) == 0 {
		formats = []core.DatasetFormat{core.FormatJSON, core.FormatCSV}
	}
	uniqFormats := make([]core.DatasetFormat, 0, len(formats))
	seen := make(map[core.DatasetFormat]struct{})
	for _, format := range formats {
		if _, duplicate := seen[format]; duplicate {
			continue
		}
		if !template.SupportsFormat(format) {
			return ExportRecord{}, fmt.Errorf("format %s not supported by template", format)
		}
		uniqFormats = append(uniqFormats, format)
		seen[format] = struct{}{}
	}

	id := newID()
	now := time.Now().UTC()
	record := ExportRecord{
		ID:          id,
		Template:    template.Descriptor(),
		Scope:       input.Scope,
		Parameters:  cloneMap(input.Parameters),
		Formats:     uniqFormats,
		Status:      ExportStatusQueued,
		RequestedBy: input.RequestedBy,
		Reason:      input.Reason,
		ProjectID:   input.ProjectID,
		ProtocolID:  input.ProtocolID,
		CreatedAt:   now,
		UpdatedAt:   now,
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
		return ExportRecord{}, fmt.Errorf("export queue full")
	}

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
	record := w.snapshot(task.id)
	if record == nil {
		return
	}

	template, ok := w.catalog.ResolveDatasetTemplate(task.input.TemplateSlug)
	if !ok {
		w.fail(task.id, fmt.Sprintf("template %s missing", task.input.TemplateSlug))
		return
	}

	w.updateStatus(task.id, ExportStatusRunning, "")

	cleaned, errs := template.ValidateParameters(task.input.Parameters)
	if len(errs) > 0 {
		w.fail(task.id, fmt.Sprintf("parameter validation failed: %v", errs))
		return
	}

	result, paramErrs, err := template.Run(w.ctx, cleaned, task.input.Scope, core.FormatJSON)
	if err != nil {
		w.fail(task.id, fmt.Sprintf("dataset run failed: %v", err))
		return
	}
	if len(paramErrs) > 0 {
		w.fail(task.id, fmt.Sprintf("parameter validation failed: %v", paramErrs))
		return
	}

	exportArtifacts := make([]ExportArtifact, 0, len(record.Formats))
	for _, format := range record.Formats {
		rendered, err := w.materialize(format, template, result)
		if err != nil {
			w.fail(task.id, err.Error())
			return
		}
		if w.store != nil {
			stored, err := w.store.Put(w.ctx, rendered.Artifact.ID, rendered.Payload, rendered.Artifact.ContentType, rendered.Artifact.Metadata)
			if err != nil {
				w.fail(task.id, fmt.Sprintf("store artifact failed: %v", err))
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
		} else {
			exportArtifacts = append(exportArtifacts, rendered.Artifact)
		}
	}

	w.complete(task.id, exportArtifacts)
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
}

func (w *Worker) complete(id string, artifacts []ExportArtifact) {
	now := time.Now().UTC()
	w.mu.Lock()
	if record, ok := w.jobs[id]; ok {
		record.Status = ExportStatusSucceeded
		record.Error = ""
		record.Artifacts = artifacts
		record.UpdatedAt = now
		record.CompletedAt = &now
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
}

func (w *Worker) fail(id, reason string) {
	now := time.Now().UTC()
	w.mu.Lock()
	if record, ok := w.jobs[id]; ok {
		record.Status = ExportStatusFailed
		record.Error = reason
		record.UpdatedAt = now
		record.CompletedAt = &now
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

func (w *Worker) scopeFor(id string) core.DatasetScope {
	w.mu.RLock()
	defer w.mu.RUnlock()
	if record, ok := w.jobs[id]; ok {
		return record.Scope
	}
	return core.DatasetScope{}
}

func (w *Worker) materialize(format core.DatasetFormat, template core.DatasetTemplate, result core.DatasetRunResult) (renderedArtifact, error) {
	switch format {
	case core.FormatJSON:
		payload, err := json.Marshal(result)
		if err != nil {
			return renderedArtifact{}, fmt.Errorf("marshal json: %w", err)
		}
		return renderedArtifact{
			Artifact: ExportArtifact{
				ID:          newID(),
				Format:      core.FormatJSON,
				ContentType: "application/json",
				SizeBytes:   int64(len(payload)),
				Metadata: map[string]any{
					"rows": len(result.Rows),
				},
				CreatedAt: time.Now().UTC(),
			},
			Payload: payload,
		}, nil
	case core.FormatCSV:
		buf := &bytes.Buffer{}
		writer := csv.NewWriter(buf)
		columns := result.Schema
		if len(columns) == 0 {
			columns = template.Descriptor().Columns
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
				Format:      core.FormatCSV,
				ContentType: "text/csv",
				SizeBytes:   int64(len(payload)),
				Metadata: map[string]any{
					"rows": len(result.Rows),
				},
				CreatedAt: time.Now().UTC(),
			},
			Payload: payload,
		}, nil
	case core.FormatHTML:
		payload := buildHTML(template, result)
		return renderedArtifact{
			Artifact: ExportArtifact{
				ID:          newID(),
				Format:      core.FormatHTML,
				ContentType: "text/html",
				SizeBytes:   int64(len(payload)),
				Metadata:    map[string]any{"rows": len(result.Rows)},
				CreatedAt:   time.Now().UTC(),
			},
			Payload: payload,
		}, nil
	case core.FormatParquet:
		payload, err := json.Marshal(result.Rows)
		if err != nil {
			return renderedArtifact{}, fmt.Errorf("marshall parquet surrogate: %w", err)
		}
		return renderedArtifact{
			Artifact: ExportArtifact{
				ID:          newID(),
				Format:      core.FormatParquet,
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
	case core.FormatPNG:
		payload, err := buildPNG(result)
		if err != nil {
			return renderedArtifact{}, err
		}
		return renderedArtifact{
			Artifact: ExportArtifact{
				ID:          newID(),
				Format:      core.FormatPNG,
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

func buildHTML(template core.DatasetTemplate, result core.DatasetRunResult) []byte {
	columns := result.Schema
	if len(columns) == 0 {
		columns = template.Descriptor().Columns
	}
	buf := &strings.Builder{}
	buf.WriteString("<!DOCTYPE html><html><head><meta charset=\"utf-8\"><title>")
	buf.WriteString(template.Title)
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

func buildPNG(result core.DatasetRunResult) ([]byte, error) {
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
	dup.Formats = append([]core.DatasetFormat(nil), r.Formats...)
	if len(r.Artifacts) > 0 {
		dup.Artifacts = append([]ExportArtifact(nil), r.Artifacts...)
	}
	return dup
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
func (s *MemoryObjectStore) Put(ctx context.Context, key string, payload []byte, contentType string, metadata map[string]any) (ExportArtifact, error) {
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

func (s *MemoryObjectStore) Get(ctx context.Context, key string) (ExportArtifact, []byte, error) {
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

func (s *MemoryObjectStore) Delete(ctx context.Context, key string) (bool, error) {
	s.mu.Lock()
	_, existed := s.objects[key]
	if existed {
		delete(s.objects, key)
	}
	s.mu.Unlock()
	return existed, nil
}

func (s *MemoryObjectStore) List(ctx context.Context, prefix string) ([]ExportArtifact, error) {
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
func (l *MemoryAuditLog) Record(ctx context.Context, entry AuditEntry) {
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
