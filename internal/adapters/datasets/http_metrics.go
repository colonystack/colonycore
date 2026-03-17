package datasets

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// HTTPMetrics captures Prometheus request counters and latency histograms for
// dataset HTTP traffic.
type HTTPMetrics struct {
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
}

var (
	defaultHTTPMetricsOnce sync.Once
	defaultHTTPMetricsInst *HTTPMetrics
)

func defaultHTTPMetrics() *HTTPMetrics {
	defaultHTTPMetricsOnce.Do(func() {
		metrics, err := NewHTTPMetrics(prometheus.DefaultRegisterer)
		if err != nil {
			log.Printf("datasets http metrics disabled: %v", err)
			defaultHTTPMetricsInst = nil
			return
		}
		defaultHTTPMetricsInst = metrics
	})
	return defaultHTTPMetricsInst
}

// NewHTTPMetrics constructs dataset HTTP metrics and registers them when a
// registerer is supplied.
func NewHTTPMetrics(registerer prometheus.Registerer) (*HTTPMetrics, error) {
	requests := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests handled by the dataset adapter.",
		},
		[]string{"method", "route", "status_code"},
	)
	duration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests handled by the dataset adapter.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "route", "status_code"},
	)

	if registerer == nil {
		return &HTTPMetrics{requests: requests, duration: duration}, nil
	}

	registeredRequests, err := registerCounterVec(registerer, requests)
	if err != nil {
		return nil, err
	}
	registeredDuration, err := registerHistogramVec(registerer, duration)
	if err != nil {
		return nil, err
	}

	return &HTTPMetrics{
		requests: registeredRequests,
		duration: registeredDuration,
	}, nil
}

func registerCounterVec(registerer prometheus.Registerer, collector *prometheus.CounterVec) (*prometheus.CounterVec, error) {
	if err := registerer.Register(collector); err != nil {
		alreadyRegistered, ok := err.(prometheus.AlreadyRegisteredError)
		if !ok {
			return nil, err
		}
		existing, ok := alreadyRegistered.ExistingCollector.(*prometheus.CounterVec)
		if !ok {
			return nil, fmt.Errorf("existing http_requests_total collector has unexpected type %T", alreadyRegistered.ExistingCollector)
		}
		return existing, nil
	}
	return collector, nil
}

func registerHistogramVec(registerer prometheus.Registerer, collector *prometheus.HistogramVec) (*prometheus.HistogramVec, error) {
	if err := registerer.Register(collector); err != nil {
		alreadyRegistered, ok := err.(prometheus.AlreadyRegisteredError)
		if !ok {
			return nil, err
		}
		existing, ok := alreadyRegistered.ExistingCollector.(*prometheus.HistogramVec)
		if !ok {
			return nil, fmt.Errorf("existing http_request_duration_seconds collector has unexpected type %T", alreadyRegistered.ExistingCollector)
		}
		return existing, nil
	}
	return collector, nil
}

// Observe records one completed dataset HTTP request.
func (m *HTTPMetrics) Observe(method, route string, status int, duration time.Duration) {
	if m == nil {
		return
	}
	if method == "" {
		method = http.MethodGet
	}
	if route == "" {
		route = unmatchedRoute
	}
	if status == 0 {
		status = http.StatusOK
	}

	labels := prometheus.Labels{
		"method":      method,
		"route":       route,
		"status_code": strconv.Itoa(status),
	}
	counter, err := m.requests.GetMetricWith(labels)
	if err != nil {
		log.Printf("datasets http request counter skipped: %v", err)
		return
	}
	histogram, err := m.duration.GetMetricWith(labels)
	if err != nil {
		log.Printf("datasets http request histogram skipped: %v", err)
		return
	}
	counter.Inc()
	histogram.Observe(duration.Seconds())
}

func (h *Handler) httpMetrics() *HTTPMetrics {
	if h == nil || h.Metrics == nil {
		return defaultHTTPMetrics()
	}
	return h.Metrics
}
