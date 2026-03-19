package datasets

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"colonycore/internal/entitymodel"
	"colonycore/internal/observability"
	"colonycore/pkg/datasetapi"
)

// Catalog exposes dataset templates for HTTP handlers.
type Catalog interface {
	DatasetTemplates() []datasetapi.TemplateDescriptor
	ResolveDatasetTemplate(slug string) (datasetapi.TemplateRuntime, bool)
}

// Handler provides HTTP access to dataset templates and exports.
type Handler struct {
	Catalog                Catalog
	Exports                ExportScheduler
	EntityModel            http.Handler
	Events                 observability.Recorder
	Logger                 RequestLogger
	Metrics                *HTTPMetrics
	CorrelationIDGenerator func() string
}

// NewHandler constructs a dataset HTTP handler.
func NewHandler(c Catalog) *Handler {
	return &Handler{
		Catalog:     c,
		EntityModel: entitymodel.NewOpenAPIHandler(),
		Events:      observability.NoopRecorder{},
		Logger:      noopRequestLogger{},
		Metrics:     defaultHTTPMetrics(),
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var handler http.Handler = http.HandlerFunc(h.serveHTTP)
	handler = h.requestLoggingMiddleware(handler)
	handler = h.requestMetricsMiddleware(handler)
	handler = h.correlationIDMiddleware(handler)
	handler.ServeHTTP(w, r)
}

func (h *Handler) serveHTTP(w http.ResponseWriter, r *http.Request) {
	if h.Catalog == nil {
		writeError(w, http.StatusInternalServerError, "dataset catalog not configured")
		return
	}

	path := strings.TrimSuffix(r.URL.Path, "/")
	switch {
	case r.Method == http.MethodGet && path == datasetTemplatesPath:
		h.handleListTemplates(w, r)
		return
	case r.Method == http.MethodGet && path == entityModelOpenAPIPath:
		h.handleEntityModelOpenAPI(w, r)
		return
	case strings.HasPrefix(path, datasetExportsPath):
		if h.Exports == nil {
			writeError(w, http.StatusNotFound, "dataset endpoint not found")
			return
		}
		h.handleExports(w, r, path)
		return
	case strings.HasPrefix(path, datasetTemplatesPath+"/"):
		h.handleTemplate(w, r, strings.TrimPrefix(path, datasetTemplatesPath+"/"))
		return
	default:
		writeError(w, http.StatusNotFound, "dataset endpoint not found")
	}
}

func (h *Handler) handleEntityModelOpenAPI(w http.ResponseWriter, r *http.Request) {
	handler := h.EntityModel
	if handler == nil {
		handler = entitymodel.NewOpenAPIHandler()
	}
	meta := entitymodel.MetadataInfo()
	if meta.Version != "" {
		w.Header().Set("X-Entity-Model-Version", meta.Version)
	}
	if meta.Status != "" {
		w.Header().Set("X-Entity-Model-Status", meta.Status)
	}
	if meta.Source != "" {
		w.Header().Set("X-Entity-Model-Source", meta.Source)
	}
	handler.ServeHTTP(w, r)
}

func (h *Handler) handleListTemplates(w http.ResponseWriter, r *http.Request) {
	page, pageSize, err := parseTemplatePagination(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	templates := h.Catalog.DatasetTemplates()
	datasetapi.SortTemplateDescriptors(templates)
	templates = filterTemplatesByScope(templates, datasetScopeFromRequest(r))
	templates, pagination := paginateTemplates(templates, page, pageSize)

	writeJSON(w, http.StatusOK, templateListResponse{
		Templates:  templates,
		Pagination: pagination,
	})
}

func (h *Handler) handleTemplate(w http.ResponseWriter, r *http.Request, remainder string) {
	segments := strings.Split(remainder, "/")
	if len(segments) < 3 {
		writeError(w, http.StatusNotFound, "dataset template not found")
		return
	}
	plugin, key, version := segments[0], segments[1], segments[2]
	slug := fmt.Sprintf("%s/%s@%s", plugin, key, version)

	template, ok := h.Catalog.ResolveDatasetTemplate(slug)
	if !ok {
		writeError(w, http.StatusNotFound, "dataset template not found")
		return
	}

	descriptor := template.Descriptor()

	if len(segments) == 3 {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"template": descriptor})
		return
	}

	if len(segments) != 4 {
		writeError(w, http.StatusNotFound, "dataset endpoint not found")
		return
	}

	action := segments[3]
	switch action {
	case "validate":
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		h.handleValidate(w, r, template)
	case "run":
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		h.handleRun(w, r, template)
	default:
		writeError(w, http.StatusNotFound, "dataset endpoint not found")
	}
}

func (h *Handler) handleExports(w http.ResponseWriter, r *http.Request, path string) {
	if path == datasetExportsPath {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		h.handleExportCreate(w, r)
		return
	}

	if !strings.HasPrefix(path, datasetExportsPath+"/") {
		writeError(w, http.StatusNotFound, "dataset endpoint not found")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	id := strings.TrimPrefix(path, datasetExportsPath+"/")
	if id == "" || strings.Contains(id, "/") {
		writeError(w, http.StatusNotFound, "dataset endpoint not found")
		return
	}
	record, ok := h.Exports.GetExport(id)
	if !ok {
		writeError(w, http.StatusNotFound, "export not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"export": record})
}

type validationRequest struct {
	Parameters map[string]any `json:"parameters"`
}

type templateListResponse struct {
	Templates  []datasetapi.TemplateDescriptor `json:"templates"`
	Pagination templatePagination              `json:"pagination"`
}

type templatePagination struct {
	Page       int  `json:"page"`
	PageSize   int  `json:"page_size"`
	TotalItems int  `json:"total_items"`
	TotalPages int  `json:"total_pages"`
	HasNext    bool `json:"has_next"`
	HasPrev    bool `json:"has_prev"`
}

type validationResponse struct {
	Template   datasetapi.TemplateDescriptor `json:"template"`
	Valid      bool                          `json:"valid"`
	Parameters map[string]any                `json:"parameters"`
}

const emptyBodySentinel = "EOF"

func (h *Handler) handleValidate(w http.ResponseWriter, r *http.Request, template datasetapi.TemplateRuntime) {
	var req validationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err.Error() != emptyBodySentinel {
		writeError(w, http.StatusBadRequest, "invalid validation request payload")
		return
	}
	cleaned, errs := template.ValidateParameters(req.Parameters)
	if len(errs) > 0 {
		writeParameterValidationProblem(w, http.StatusUnprocessableEntity, errs)
		return
	}
	writeJSON(w, http.StatusOK, validationResponse{
		Template:   template.Descriptor(),
		Valid:      true,
		Parameters: cleaned,
	})
}

type runRequest struct {
	Parameters map[string]any `json:"parameters"`
	Scope      struct {
		Requestor   string   `json:"requestor"`
		Roles       []string `json:"roles"`
		ProjectIDs  []string `json:"project_ids"`
		ProtocolIDs []string `json:"protocol_ids"`
	} `json:"scope"`
}

type runResponse struct {
	Template   datasetapi.TemplateDescriptor `json:"template"`
	Scope      datasetapi.Scope              `json:"scope"`
	Parameters map[string]any                `json:"parameters"`
	Result     datasetapi.RunResult          `json:"result"`
}

type exportRequest struct {
	Template struct {
		Slug    string `json:"slug"`
		Plugin  string `json:"plugin"`
		Key     string `json:"key"`
		Version string `json:"version"`
	} `json:"template"`
	Parameters map[string]any `json:"parameters"`
	Formats    []string       `json:"formats"`
	Scope      struct {
		Requestor   string   `json:"requestor"`
		Roles       []string `json:"roles"`
		ProjectIDs  []string `json:"project_ids"`
		ProtocolIDs []string `json:"protocol_ids"`
	} `json:"scope"`
	RequestedBy string `json:"requested_by"`
	Reason      string `json:"reason"`
	ProjectID   string `json:"project_id"`
	ProtocolID  string `json:"protocol_id"`
}

func (h *Handler) handleRun(w http.ResponseWriter, r *http.Request, template datasetapi.TemplateRuntime) {
	formatProvider := datasetapi.GetFormatProvider()
	descriptor := template.Descriptor()

	started := time.Now()
	status := observability.StatusSuccess
	errMessage := ""
	labels := map[string]string{
		"template_id": descriptor.Slug,
	}
	if correlationID := CorrelationIDFromContext(r.Context()); correlationID != "" {
		labels["correlation_id"] = correlationID
	}
	measures := map[string]float64{}
	defer func() {
		h.eventRecorder().Record(r.Context(), observability.Event{
			Category:   observability.CategoryCatalogOperation,
			Name:       "catalog.template.run",
			Status:     status,
			DurationMS: observability.DurationMS(time.Since(started)),
			Error:      errMessage,
			Labels:     labels,
			Measures:   measures,
		})
	}()

	var req runRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err.Error() != "EOF" {
		status = observability.StatusError
		errMessage = "invalid run request payload"
		writeError(w, http.StatusBadRequest, "invalid run request payload")
		return
	}

	scope := datasetapi.Scope{Requestor: req.Scope.Requestor}
	if len(req.Scope.Roles) > 0 {
		scope.Roles = append([]string(nil), req.Scope.Roles...)
	}
	if len(req.Scope.ProjectIDs) > 0 {
		scope.ProjectIDs = append([]string(nil), req.Scope.ProjectIDs...)
	}
	if len(req.Scope.ProtocolIDs) > 0 {
		scope.ProtocolIDs = append([]string(nil), req.Scope.ProtocolIDs...)
	}

	cleaned, errs := template.ValidateParameters(req.Parameters)
	if len(errs) > 0 {
		status = observability.StatusError
		errMessage = parameterValidationFailed
		measures["validation_errors_total"] = float64(len(errs))
		writeParameterValidationProblem(w, http.StatusUnprocessableEntity, errs)
		return
	}

	format := negotiateFormat(r, descriptor.OutputFormats)
	if format == "" {
		status = observability.StatusError
		errMessage = "requested format not supported"
		writeError(w, http.StatusNotAcceptable, "requested format not supported")
		return
	}

	selectedFormat := datasetapi.Format(strings.ToLower(format))
	labels["format"] = string(selectedFormat)
	result, paramErrs, err := template.Run(r.Context(), cleaned, scope, selectedFormat)
	if err != nil {
		status = observability.StatusError
		errMessage = err.Error()
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if len(paramErrs) > 0 {
		status = observability.StatusError
		errMessage = parameterValidationFailed
		measures["validation_errors_total"] = float64(len(paramErrs))
		writeParameterValidationProblem(w, http.StatusUnprocessableEntity, paramErrs)
		return
	}
	measures["rows_total"] = float64(len(result.Rows))

	switch selectedFormat {
	case formatProvider.CSV():
		if err := streamCSV(w, descriptor, result); err != nil {
			status = observability.StatusError
			errMessage = err.Error()
			h.logger().Error("dataset csv stream failed", "template", descriptor.Slug, "error", err.Error())
		}
	default:
		writeJSON(w, http.StatusOK, runResponse{
			Template:   descriptor,
			Scope:      scope,
			Parameters: cleaned,
			Result:     result,
		})
	}
}

func (h *Handler) handleExportCreate(w http.ResponseWriter, r *http.Request) {
	formatProvider := datasetapi.GetFormatProvider()
	started := time.Now()
	status := observability.StatusSuccess
	errMessage := ""
	labels := map[string]string{}
	if correlationID := CorrelationIDFromContext(r.Context()); correlationID != "" {
		labels["correlation_id"] = correlationID
	}
	measures := map[string]float64{}
	defer func() {
		h.eventRecorder().Record(r.Context(), observability.Event{
			Category:   observability.CategoryCatalogOperation,
			Name:       "catalog.export.create",
			Status:     status,
			DurationMS: observability.DurationMS(time.Since(started)),
			Error:      errMessage,
			Labels:     labels,
			Measures:   measures,
		})
	}()

	var req exportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err.Error() != "EOF" {
		status = observability.StatusError
		errMessage = "invalid export request payload"
		writeError(w, http.StatusBadRequest, "invalid export request payload")
		return
	}

	slug := strings.TrimSpace(req.Template.Slug)
	if slug == "" {
		if req.Template.Plugin == "" || req.Template.Key == "" || req.Template.Version == "" {
			status = observability.StatusError
			errMessage = "template slug or plugin/key/version required"
			writeError(w, http.StatusBadRequest, "template slug or plugin/key/version required")
			return
		}
		slug = fmt.Sprintf("%s/%s@%s", req.Template.Plugin, req.Template.Key, req.Template.Version)
	}
	labels["template_id"] = slug

	formats := make([]datasetapi.Format, 0, len(req.Formats))
	for _, f := range req.Formats {
		switch strings.ToLower(strings.TrimSpace(f)) {
		case "json":
			formats = append(formats, formatProvider.JSON())
		case "csv":
			formats = append(formats, formatProvider.CSV())
		case "parquet":
			formats = append(formats, formatProvider.Parquet())
		case "png":
			formats = append(formats, formatProvider.PNG())
		case "html":
			formats = append(formats, formatProvider.HTML())
		default:
			status = observability.StatusError
			errMessage = "unsupported export format"
			writeError(w, http.StatusBadRequest, "unsupported export format")
			return
		}
	}
	measures["formats_total"] = float64(len(formats))

	scope := datasetapi.Scope{Requestor: req.Scope.Requestor}
	if len(req.Scope.Roles) > 0 {
		scope.Roles = append([]string(nil), req.Scope.Roles...)
	}
	if len(req.Scope.ProjectIDs) > 0 {
		scope.ProjectIDs = append([]string(nil), req.Scope.ProjectIDs...)
	}
	if len(req.Scope.ProtocolIDs) > 0 {
		scope.ProtocolIDs = append([]string(nil), req.Scope.ProtocolIDs...)
	}

	record, err := h.Exports.EnqueueExport(r.Context(), ExportInput{
		TemplateSlug: slug,
		Parameters:   req.Parameters,
		Formats:      formats,
		Scope:        scope,
		RequestedBy:  firstNonEmpty(req.RequestedBy, req.Scope.Requestor),
		Reason:       req.Reason,
		ProjectID:    req.ProjectID,
		ProtocolID:   req.ProtocolID,
	})
	if err != nil {
		status = observability.StatusError
		errMessage = err.Error()
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	labels["export_id"] = record.ID

	writeJSON(w, http.StatusAccepted, map[string]any{"export": record})
}

func (h *Handler) eventRecorder() observability.Recorder {
	if h == nil || h.Events == nil {
		return observability.NoopRecorder{}
	}
	return h.Events
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func parameterValidationDetail(errs []datasetapi.ParameterError) string {
	if len(errs) == 0 {
		return parameterValidationFailed
	}

	details := make([]string, 0, len(errs))
	for _, err := range errs {
		switch {
		case err.Name != "" && err.Message != "":
			details = append(details, fmt.Sprintf("parameter %s: %s", err.Name, err.Message))
		case err.Message != "":
			details = append(details, err.Message)
		case err.Name != "":
			details = append(details, fmt.Sprintf("parameter %s: invalid value", err.Name))
		}
	}
	if len(details) == 0 {
		return parameterValidationFailed
	}
	return strings.Join(details, "; ")
}

func writeParameterValidationProblem(w http.ResponseWriter, status int, errs []datasetapi.ParameterError) {
	writeProblemWithErrors(w, status, parameterValidationDetail(errs), errs)
}

func negotiateFormat(r *http.Request, supported []datasetapi.Format) string {
	formatProvider := datasetapi.GetFormatProvider()

	wanted := strings.ToLower(r.URL.Query().Get("format"))
	if wanted == "" {
		accept := r.Header.Get("Accept")
		if strings.Contains(accept, "text/csv") {
			wanted = string(formatProvider.CSV())
		} else {
			wanted = string(formatProvider.JSON())
		}
	}
	switch datasetapi.Format(wanted) {
	case formatProvider.CSV(), formatProvider.JSON():
		for _, candidate := range supported {
			if string(candidate) == wanted {
				return wanted
			}
		}
	}
	return ""
}

func streamCSV(w http.ResponseWriter, descriptor datasetapi.TemplateDescriptor, result datasetapi.RunResult) error {
	filename := fmt.Sprintf("%s-%s.csv", descriptor.Key, time.Now().UTC().Format("20060102T150405Z"))
	columns := csvColumns(descriptor, result)
	totalBytes := estimateCSVSize(columns, result.Rows)

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Add("Trailer", streamErrorTrailer)
	if totalBytes > 0 {
		w.Header().Add("Trailer", streamProgressHeader)
		w.Header().Set(streamProgressHeader, fmt.Sprintf("bytes=0/%d", totalBytes))
	}

	counter := &countingResponseWriter{ResponseWriter: w}
	var streamErr error
	defer func() {
		if totalBytes <= 0 {
			if streamErr != nil {
				w.Header().Set(streamErrorTrailer, streamErr.Error())
			}
			return
		}
		progressTotal := totalBytes
		if counter.bytesWritten > progressTotal {
			progressTotal = counter.bytesWritten
		}
		w.Header().Set(streamProgressHeader, fmt.Sprintf("bytes=%d/%d", counter.bytesWritten, progressTotal))
		if streamErr != nil {
			w.Header().Set(streamErrorTrailer, streamErr.Error())
		}
	}()

	writer := csv.NewWriter(counter)
	flusher, _ := w.(http.Flusher)

	if err := writer.Write(csvHeaderRecord(columns)); err != nil {
		streamErr = fmt.Errorf("write csv header: %w", err)
		return streamErr
	}
	writer.Flush()
	if writer.Error() != nil {
		streamErr = fmt.Errorf("flush csv header: %w", writer.Error())
		return streamErr
	}
	if flusher != nil {
		flusher.Flush()
	}

	rowsSinceFlush := 0
	for i, row := range result.Rows {
		if err := writer.Write(csvRowRecord(columns, row)); err != nil {
			streamErr = fmt.Errorf("write csv row %d: %w", i, err)
			return streamErr
		}
		rowsSinceFlush++
		if rowsSinceFlush >= csvStreamFlushEveryRows {
			writer.Flush()
			if writer.Error() != nil {
				streamErr = fmt.Errorf("flush csv rows at row %d: %w", i, writer.Error())
				return streamErr
			}
			if flusher != nil {
				flusher.Flush()
			}
			rowsSinceFlush = 0
		}
	}
	writer.Flush()
	if writer.Error() != nil {
		streamErr = fmt.Errorf("flush final csv rows: %w", writer.Error())
		return streamErr
	}
	if flusher != nil {
		flusher.Flush()
	}
	return nil
}

func parseTemplatePagination(r *http.Request) (int, int, error) {
	page, err := parsePositiveQueryInt(r, "page", datasetListDefaultPage)
	if err != nil {
		return 0, 0, err
	}

	pageSize, err := parsePositiveQueryInt(r, "page_size", datasetListDefaultPageSize)
	if err != nil {
		return 0, 0, err
	}
	if pageSize > datasetListMaxPageSize {
		return 0, 0, fmt.Errorf("page_size must be between 1 and %d", datasetListMaxPageSize)
	}

	return page, pageSize, nil
}

func parsePositiveQueryInt(r *http.Request, key string, fallback int) (int, error) {
	if r == nil {
		return fallback, nil
	}
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", key)
	}
	return value, nil
}

func paginateTemplates(templates []datasetapi.TemplateDescriptor, page int, pageSize int) ([]datasetapi.TemplateDescriptor, templatePagination) {
	totalItems := len(templates)
	totalPages := 0
	if totalItems > 0 {
		totalPages = (totalItems + pageSize - 1) / pageSize
	}

	start64 := int64(page-1) * int64(pageSize)
	if start64 < 0 || start64 >= int64(totalItems) {
		return []datasetapi.TemplateDescriptor{}, templatePagination{
			Page:       page,
			PageSize:   pageSize,
			TotalItems: totalItems,
			TotalPages: totalPages,
			HasNext:    totalPages > 0 && page < totalPages,
			HasPrev:    totalPages > 0 && page > 1,
		}
	}
	end64 := start64 + int64(pageSize)
	if end64 < start64 || end64 > int64(totalItems) {
		end64 = int64(totalItems)
	}
	start := int(start64)
	end := int(end64)

	paged := make([]datasetapi.TemplateDescriptor, 0, end-start)
	if start < end {
		paged = append(paged, templates[start:end]...)
	}

	return paged, templatePagination{
		Page:       page,
		PageSize:   pageSize,
		TotalItems: totalItems,
		TotalPages: totalPages,
		HasNext:    totalPages > 0 && page < totalPages,
		HasPrev:    totalPages > 0 && page > 1,
	}
}

func filterTemplatesByScope(templates []datasetapi.TemplateDescriptor, scope datasetapi.Scope) []datasetapi.TemplateDescriptor {
	if len(templates) == 0 {
		return []datasetapi.TemplateDescriptor{}
	}

	filtered := make([]datasetapi.TemplateDescriptor, 0, len(templates))
	for _, template := range templates {
		if templateAllowedByScope(template, scope) {
			filtered = append(filtered, template)
		}
	}
	return filtered
}

func templateAllowedByScope(template datasetapi.TemplateDescriptor, scope datasetapi.Scope) bool {
	annotations := template.Metadata.Annotations
	if !matchesScopeAnnotation(annotations[templateRBACProjectsAnnotation], scope.ProjectIDs) {
		return false
	}
	if !matchesScopeAnnotation(annotations[templateRBACProtocolsAnnotation], scope.ProtocolIDs) {
		return false
	}
	return true
}

func matchesScopeAnnotation(annotation string, permitted []string) bool {
	values := parseDelimitedValues(annotation)
	if len(values) == 0 {
		return true
	}
	if len(values) == 1 && values[0] == wildcardScopeValue {
		return len(permitted) > 0
	}
	if len(permitted) == 0 {
		return false
	}

	allowed := make(map[string]struct{}, len(values))
	for _, value := range values {
		allowed[value] = struct{}{}
	}
	for _, candidate := range permitted {
		if _, ok := allowed[candidate]; ok {
			return true
		}
	}
	return false
}

func datasetScopeFromRequest(r *http.Request) datasetapi.Scope {
	if r == nil {
		return datasetapi.Scope{}
	}

	scope := datasetapi.Scope{
		Requestor:   strings.TrimSpace(r.Header.Get(datasetScopeRequestorHeader)),
		Roles:       parseDelimitedValues(r.Header.Get(datasetScopeRolesHeader)),
		ProjectIDs:  parseDelimitedValues(r.Header.Get(datasetScopeProjectIDsHeader)),
		ProtocolIDs: parseDelimitedValues(r.Header.Get(datasetScopeProtocolIDsHeader)),
	}
	if len(scope.Roles) == 0 {
		scope.Roles = nil
	}
	if len(scope.ProjectIDs) == 0 {
		scope.ProjectIDs = nil
	}
	if len(scope.ProtocolIDs) == 0 {
		scope.ProtocolIDs = nil
	}
	return scope
}

func parseDelimitedValues(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	seen := make(map[string]struct{})
	values := make([]string, 0, strings.Count(value, ",")+1)
	for _, part := range strings.Split(value, ",") {
		candidate := strings.TrimSpace(part)
		if candidate == "" {
			continue
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		values = append(values, candidate)
	}
	if len(values) == 0 {
		return nil
	}
	return values
}

type countingResponseWriter struct {
	http.ResponseWriter
	bytesWritten int
}

func (w *countingResponseWriter) Write(p []byte) (int, error) {
	n, err := w.ResponseWriter.Write(p)
	w.bytesWritten += n
	return n, err
}

func (w *countingResponseWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func csvColumns(descriptor datasetapi.TemplateDescriptor, result datasetapi.RunResult) []datasetapi.Column {
	if len(result.Schema) > 0 {
		return result.Schema
	}
	return descriptor.Columns
}

func csvHeaderRecord(columns []datasetapi.Column) []string {
	record := make([]string, len(columns))
	for i, column := range columns {
		record[i] = column.Name
	}
	return record
}

func csvRowRecord(columns []datasetapi.Column, row datasetapi.Row) []string {
	record := make([]string, len(columns))
	for i, column := range columns {
		record[i] = formatValue(row[column.Name])
	}
	return record
}

func estimateCSVSize(columns []datasetapi.Column, rows []datasetapi.Row) int {
	headerSize, err := estimateCSVSectionSize(columns, nil)
	if err != nil {
		return 0
	}
	if len(rows) == 0 {
		return headerSize
	}

	sampleRows := rows
	if len(sampleRows) > csvProgressSampleRows {
		sampleRows = sampleRows[:csvProgressSampleRows]
	}

	sampleSize, err := estimateCSVSectionSize(columns, sampleRows)
	if err != nil {
		return 0
	}
	if len(rows) == len(sampleRows) {
		return sampleSize
	}

	samplePayloadSize := sampleSize - headerSize
	if samplePayloadSize < 0 {
		samplePayloadSize = 0
	}
	averageRowSize := 0
	if len(sampleRows) > 0 {
		averageRowSize = (samplePayloadSize + len(sampleRows) - 1) / len(sampleRows)
	}

	totalSize := headerSize + averageRowSize*len(rows)
	if totalSize < sampleSize {
		totalSize = sampleSize
	}
	return totalSize
}

func estimateCSVSectionSize(columns []datasetapi.Column, rows []datasetapi.Row) (int, error) {
	buf := &bytes.Buffer{}
	writer := csv.NewWriter(buf)
	if err := writer.Write(csvHeaderRecord(columns)); err != nil {
		return 0, err
	}
	for _, row := range rows {
		if err := writer.Write(csvRowRecord(columns, row)); err != nil {
			return 0, err
		}
	}
	writer.Flush()
	if writer.Error() != nil {
		return 0, writer.Error()
	}
	return buf.Len(), nil
}

func formatValue(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case time.Time:
		return v.UTC().Format(time.RFC3339)
	case fmt.Stringer:
		return v.String()
	case float32:
		return fmt.Sprintf("%g", v)
	case float64:
		return fmt.Sprintf("%g", v)
	case int:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	default:
		return fmt.Sprint(v)
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeProblem(w, status, message)
}
