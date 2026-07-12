package tracing

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// Middleware starts one server span per request. Hand-rolled instead of
// otelhttp so the span name uses the Chi route pattern (bounded cardinality,
// resolved after the handler runs) rather than the raw URL. Must be mounted
// before the metrics middleware so exemplars can read the span context.
func Middleware(next http.Handler) http.Handler {
	prop := otel.GetTextMapPropagator()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := prop.Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		ctx, span := otel.Tracer("shop-api/http").Start(ctx, r.Method,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				attribute.String("http.request.method", r.Method),
				attribute.String("url.path", r.URL.Path),
			),
		)
		defer span.End()

		ww, ok := w.(chimw.WrapResponseWriter)
		if !ok {
			ww = chimw.NewWrapResponseWriter(w, r.ProtoMajor)
		}
		next.ServeHTTP(ww, r.WithContext(ctx))

		route := "unmatched"
		if rc := chi.RouteContext(r.Context()); rc != nil {
			if p := rc.RoutePattern(); p != "" {
				route = p
			}
		}
		status := ww.Status()
		if status == 0 {
			status = http.StatusOK
		}
		span.SetName(r.Method + " " + route)
		span.SetAttributes(
			attribute.String("http.route", route),
			attribute.Int("http.response.status_code", status),
		)
		if status >= 500 {
			span.SetStatus(codes.Error, http.StatusText(status))
		}
	})
}

// ExemplarTraceID reports the active sampled trace ID, for linking Prometheus
// histogram exemplars to Tempo. Matches the metrics.HTTPMetrics.ExemplarFn
// signature so the metrics package needs no OTel import.
func ExemplarTraceID(ctx context.Context) (string, bool) {
	sc := trace.SpanContextFromContext(ctx)
	if sc.IsValid() && sc.IsSampled() {
		return sc.TraceID().String(), true
	}
	return "", false
}
