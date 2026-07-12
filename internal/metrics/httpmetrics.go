package metrics

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
)

// HTTPMetrics is Chi middleware recording request rate, duration, and
// in-flight count. The route label is the Chi route pattern (e.g.
// /api/v1/products/{id}), a bounded set — never the raw URL path.
type HTTPMetrics struct {
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
	inFlight prometheus.Gauge

	// ExemplarFn, when set, supplies the current trace ID so duration
	// observations carry an exemplar linking the sample to its trace.
	// Injected from main.go (tracing.ExemplarTraceID) to keep this package
	// free of OTel imports.
	ExemplarFn func(ctx context.Context) (string, bool)
}

func NewHTTPMetrics(reg prometheus.Registerer) *HTTPMetrics {
	m := &HTTPMetrics{
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "shop_http_requests_total",
			Help: "HTTP requests by method, route pattern, and status code.",
		}, []string{"method", "route", "status"}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "shop_http_request_duration_seconds",
			Help:    "HTTP request duration by method and route pattern.",
			Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		}, []string{"method", "route"}),
		inFlight: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "shop_http_requests_in_flight",
			Help: "HTTP requests currently being served.",
		}),
	}
	reg.MustRegister(m.requests, m.duration, m.inFlight)
	return m
}

func (m *HTTPMetrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.inFlight.Inc()
		defer m.inFlight.Dec()

		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		start := time.Now()
		next.ServeHTTP(ww, r)

		route := "unmatched"
		if rc := chi.RouteContext(r.Context()); rc != nil {
			if p := rc.RoutePattern(); p != "" {
				route = p
			}
		}
		status := ww.Status()
		if status == 0 { // handler never wrote — net/http sends an implicit 200
			status = http.StatusOK
		}
		m.requests.WithLabelValues(r.Method, route, strconv.Itoa(status)).Inc()

		obs := m.duration.WithLabelValues(r.Method, route)
		secs := time.Since(start).Seconds()
		if m.ExemplarFn != nil {
			if traceID, ok := m.ExemplarFn(r.Context()); ok {
				if eo, ok := obs.(prometheus.ExemplarObserver); ok {
					eo.ObserveWithExemplar(secs, prometheus.Labels{"trace_id": traceID})
					return
				}
			}
		}
		obs.Observe(secs)
	})
}
