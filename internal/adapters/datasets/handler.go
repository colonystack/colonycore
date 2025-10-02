package datasets

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"colonycore/internal/core"
)

// Catalog exposes dataset templates for HTTP handlers.
type Catalog interface {
	DatasetTemplates() []core.DatasetTemplateDescriptor
	ResolveDatasetTemplate(slug string) (core.DatasetTemplate, bool)
}

// Handler provides HTTP access to dataset templates and exports.
type Handler struct {
	Catalog Catalog
	Exports ExportScheduler
}

// NewHandler constructs a dataset HTTP handler.
func NewHandler(c Catalog) *Handler {
	return &Handler{Catalog: c}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.Catalog == nil {
		writeError(w, http.StatusInternalServerError, "dataset catalog not configured")
		return
	}

	path := strings.TrimSuffix(r.URL.Path, "/")
	switch {
	case r.Method == http.MethodGet && path == "/api/v1/datasets/templates":
		h.handleListTemplates(w, r)
		return
	case strings.HasPrefix(path, "/api/v1/datasets/exports"):
		if h.Exports == nil {
			http.NotFound(w, r)
			return
		}
		h.handleExports(w, r, path)
		return
	case strings.HasPrefix(path, "/api/v1/datasets/templates/"):
		h.handleTemplate(w, r, strings.TrimPrefix(path, "/api/v1/datasets/templates/"))
		return
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) handleListTemplates(w http.ResponseWriter, _ *http.Request) {
	templates := h.Catalog.DatasetTemplates()
	sort.Sort(core.DatasetTemplateCollection(templates))
	writeJSON(w, http.StatusOK, map[string]any{"templates": templates})
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
	if path == "/api/v1/datasets/exports" {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		h.handleExportCreate(w, r)
		return
	}

	if !strings.HasPrefix(path, "/api/v1/datasets/exports/") {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	id := strings.TrimPrefix(path, "/api/v1/datasets/exports/")
	if id == "" {
		http.NotFound(w, r)
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

type validationResponse struct {
	Template   core.DatasetTemplateDescriptor `json:"template"`
	Valid      bool                           `json:"valid"`
	Parameters map[string]any                 `json:"parameters"`
	Errors     []core.DatasetParameterError   `json:"errors,omitempty"`
}

const emptyBodySentinel = "EOF"

func (h *Handler) handleValidate(w http.ResponseWriter, r *http.Request, template core.DatasetTemplate) {
	var req validationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err.Error() != emptyBodySentinel {
		writeError(w, http.StatusBadRequest, "invalid validation request payload")
		return
	}
	cleaned, errs := template.ValidateParameters(req.Parameters)
	writeJSON(w, http.StatusOK, validationResponse{
		Template:   template.Descriptor(),
		Valid:      len(errs) == 0,
		Parameters: cleaned,
		Errors:     errs,
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
	Template   core.DatasetTemplateDescriptor `json:"template"`
	Scope      core.DatasetScope              `json:"scope"`
	Parameters map[string]any                 `json:"parameters"`
	Result     core.DatasetRunResult          `json:"result"`
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

func (h *Handler) handleRun(w http.ResponseWriter, r *http.Request, template core.DatasetTemplate) {
	var req runRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err.Error() != "EOF" {
		writeError(w, http.StatusBadRequest, "invalid run request payload")
		return
	}

	scope := core.DatasetScope{
		Requestor:   req.Scope.Requestor,
		Roles:       req.Scope.Roles,
		ProjectIDs:  req.Scope.ProjectIDs,
		ProtocolIDs: req.Scope.ProtocolIDs,
	}

	cleaned, errs := template.ValidateParameters(req.Parameters)
	if len(errs) > 0 {
		writeJSON(w, http.StatusBadRequest, validationResponse{
			Template:   template.Descriptor(),
			Valid:      false,
			Parameters: cleaned,
			Errors:     errs,
		})
		return
	}

	format := negotiateFormat(r, template.OutputFormats)
	if format == "" {
		writeError(w, http.StatusNotAcceptable, "requested format not supported")
		return
	}

	result, paramErrs, err := template.Run(r.Context(), cleaned, scope, core.DatasetFormat(format))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if len(paramErrs) > 0 {
		writeJSON(w, http.StatusBadRequest, validationResponse{
			Template:   template.Descriptor(),
			Valid:      false,
			Parameters: cleaned,
			Errors:     paramErrs,
		})
		return
	}

	switch core.DatasetFormat(format) {
	case core.FormatCSV:
		streamCSV(w, template, result)
	default:
		writeJSON(w, http.StatusOK, runResponse{
			Template:   template.Descriptor(),
			Scope:      scope,
			Parameters: cleaned,
			Result:     result,
		})
	}
}

func (h *Handler) handleExportCreate(w http.ResponseWriter, r *http.Request) {
	var req exportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err.Error() != "EOF" {
		writeError(w, http.StatusBadRequest, "invalid export request payload")
		return
	}

	slug := strings.TrimSpace(req.Template.Slug)
	if slug == "" {
		if req.Template.Plugin == "" || req.Template.Key == "" || req.Template.Version == "" {
			writeError(w, http.StatusBadRequest, "template slug or plugin/key/version required")
			return
		}
		slug = fmt.Sprintf("%s/%s@%s", req.Template.Plugin, req.Template.Key, req.Template.Version)
	}

	formats := make([]core.DatasetFormat, 0, len(req.Formats))
	for _, f := range req.Formats {
		switch strings.ToLower(strings.TrimSpace(f)) {
		case "json":
			formats = append(formats, core.FormatJSON)
		case "csv":
			formats = append(formats, core.FormatCSV)
		case "parquet":
			formats = append(formats, core.FormatParquet)
		case "png":
			formats = append(formats, core.FormatPNG)
		case "html":
			formats = append(formats, core.FormatHTML)
		default:
			writeError(w, http.StatusBadRequest, "unsupported export format")
			return
		}
	}

	scope := core.DatasetScope{
		Requestor:   req.Scope.Requestor,
		Roles:       req.Scope.Roles,
		ProjectIDs:  req.Scope.ProjectIDs,
		ProtocolIDs: req.Scope.ProtocolIDs,
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
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]any{"export": record})
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func negotiateFormat(r *http.Request, supported []core.DatasetFormat) string {
	wanted := strings.ToLower(r.URL.Query().Get("format"))
	if wanted == "" {
		accept := r.Header.Get("Accept")
		if strings.Contains(accept, "text/csv") {
			wanted = string(core.FormatCSV)
		} else {
			wanted = string(core.FormatJSON)
		}
	}
	switch core.DatasetFormat(wanted) {
	case core.FormatCSV, core.FormatJSON:
		for _, candidate := range supported {
			if string(candidate) == wanted {
				return wanted
			}
		}
	}
	return ""
}

func streamCSV(w http.ResponseWriter, template core.DatasetTemplate, result core.DatasetRunResult) {
	descriptor := template.Descriptor()
	filename := fmt.Sprintf("%s-%s.csv", descriptor.Key, time.Now().UTC().Format("20060102T150405Z"))

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	writer := csv.NewWriter(w)
	defer writer.Flush()

	var columns []core.DatasetColumn
	if len(result.Schema) > 0 {
		columns = result.Schema
	} else {
		columns = descriptor.Columns
	}

	headers := make([]string, len(columns))
	for i, column := range columns {
		headers[i] = column.Name
	}
	if err := writer.Write(headers); err != nil {
		return
	}

	for _, row := range result.Rows {
		record := make([]string, len(columns))
		for i, column := range columns {
			record[i] = formatValue(row[column.Name])
		}
		if err := writer.Write(record); err != nil {
			return
		}
	}
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
	writeJSON(w, status, map[string]any{"error": message})
}
