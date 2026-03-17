package datasets

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"time"

	"colonycore/internal/observability"
)

const (
	correlationIDHeader       = "X-Correlation-ID"
	datasetTemplatesPath      = "/api/v1/datasets/templates"
	entityModelOpenAPIPath    = "/admin/entity-model/openapi"
	datasetExportsPath        = "/api/v1/datasets/exports"
	parameterValidationFailed = "parameter validation failed"
)

// RequestLogger captures dataset adapter request logs using structured fields.
type RequestLogger interface {
	Info(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

type noopRequestLogger struct{}

func (noopRequestLogger) Info(string, ...interface{})  {}
func (noopRequestLogger) Error(string, ...interface{}) {}

type correlationIDContextKey struct{}

func withCorrelationID(ctx context.Context, id string) context.Context {
	if strings.TrimSpace(id) == "" {
		return ctx
	}
	return context.WithValue(ctx, correlationIDContextKey{}, id)
}

// CorrelationIDFromContext returns the request correlation ID when present.
func CorrelationIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	id, _ := ctx.Value(correlationIDContextKey{}).(string)
	return strings.TrimSpace(id)
}

func newCorrelationID() string {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err == nil {
		return hex.EncodeToString(raw[:])
	}
	return strconv.FormatInt(time.Now().UTC().UnixNano(), 16)
}

type statusCapturingResponseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (w *statusCapturingResponseWriter) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}
	w.status = status
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusCapturingResponseWriter) Write(body []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(body)
}

func (w *statusCapturingResponseWriter) StatusCode() int {
	if w == nil || w.status == 0 {
		return http.StatusOK
	}
	return w.status
}

func captureStatusWriter(w http.ResponseWriter) *statusCapturingResponseWriter {
	if existing, ok := w.(*statusCapturingResponseWriter); ok {
		return existing
	}
	return &statusCapturingResponseWriter{ResponseWriter: w}
}

func (h *Handler) correlationIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(r.Header.Get(correlationIDHeader))
		if id == "" {
			generator := h.correlationIDGenerator()
			id = generator()
		}

		w.Header().Set(correlationIDHeader, id)
		next.ServeHTTP(w, r.WithContext(withCorrelationID(r.Context(), id)))
	})
}

func (h *Handler) requestLoggingMiddleware(next http.Handler) http.Handler {
	logger := h.logger()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		route := routePattern(r.URL.Path)
		recorder := captureStatusWriter(w)

		next.ServeHTTP(recorder, r)

		args := []interface{}{
			"correlation_id", CorrelationIDFromContext(r.Context()),
			"method", r.Method,
			"route", route,
			"path", r.URL.Path,
			"status", recorder.StatusCode(),
			"duration_ms", observability.DurationMS(time.Since(started)),
		}

		if recorder.StatusCode() >= http.StatusInternalServerError {
			logger.Error("dataset http request completed", args...)
			return
		}
		logger.Info("dataset http request completed", args...)
	})
}

func (h *Handler) requestMetricsMiddleware(next http.Handler) http.Handler {
	metrics := h.httpMetrics()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		route := routePattern(r.URL.Path)
		recorder := captureStatusWriter(w)

		next.ServeHTTP(recorder, r)
		metrics.Observe(r.Method, route, recorder.StatusCode(), time.Since(started))
	})
}

func routePattern(path string) string {
	trimmed := strings.TrimSuffix(path, "/")
	switch {
	case trimmed == datasetTemplatesPath:
		return datasetTemplatesPath
	case trimmed == entityModelOpenAPIPath:
		return entityModelOpenAPIPath
	case trimmed == datasetExportsPath:
		return datasetExportsPath
	case strings.HasPrefix(trimmed, datasetExportsPath+"/"):
		return datasetExportsPath + "/{exportId}"
	case strings.HasPrefix(trimmed, datasetTemplatesPath+"/"):
		remainder := strings.TrimPrefix(trimmed, datasetTemplatesPath+"/")
		segments := strings.Split(remainder, "/")
		switch {
		case len(segments) == 3:
			return datasetTemplatesPath + "/{plugin}/{key}/{version}"
		case len(segments) == 4 && segments[3] == "validate":
			return datasetTemplatesPath + "/{plugin}/{key}/{version}/validate"
		case len(segments) == 4 && segments[3] == "run":
			return datasetTemplatesPath + "/{plugin}/{key}/{version}/run"
		default:
			return datasetTemplatesPath + "/{plugin}/{key}/{version}/*"
		}
	default:
		return "unmatched"
	}
}

func (h *Handler) logger() RequestLogger {
	if h == nil || h.Logger == nil {
		return noopRequestLogger{}
	}
	return h.Logger
}

func (h *Handler) correlationIDGenerator() func() string {
	if h == nil || h.CorrelationIDGenerator == nil {
		return newCorrelationID
	}
	return h.CorrelationIDGenerator
}
