// Package tracing wires OpenTelemetry through the app: an HTTP server span
// per request, pgx query spans, and trace-context propagation over AMQP.
// Everything no-ops when OTEL_ENABLED is unset — the global provider stays
// the default no-op one and the middleware/tracers add nothing measurable.
package tracing

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strconv"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Setup installs the global tracer provider exporting OTLP/HTTP to endpoint
// (Tempo in the compose monitoring profile). The returned shutdown func
// flushes pending spans; it is safe to call even when tracing is disabled.
// Sampling is parent-based with a ratio from OTEL_SAMPLE_RATIO (default 1.0 —
// fine for a demo; production would sample down).
func Setup(ctx context.Context, enabled bool, endpoint string) (func(context.Context) error, error) {
	noop := func(context.Context) error { return nil }
	if !enabled || endpoint == "" {
		return noop, nil
	}

	u, err := url.Parse(endpoint)
	if err != nil || u.Host == "" {
		return noop, fmt.Errorf("invalid OTEL_EXPORTER_OTLP_ENDPOINT %q: expected e.g. http://tempo:4318", endpoint)
	}
	opts := []otlptracehttp.Option{otlptracehttp.WithEndpoint(u.Host)}
	if u.Scheme != "https" {
		opts = append(opts, otlptracehttp.WithInsecure())
	}
	exp, err := otlptracehttp.New(ctx, opts...)
	if err != nil {
		return noop, fmt.Errorf("create OTLP trace exporter: %w", err)
	}

	ratio := 1.0
	if v := os.Getenv("OTEL_SAMPLE_RATIO"); v != "" {
		if f, perr := strconv.ParseFloat(v, 64); perr == nil && f >= 0 && f <= 1 {
			ratio = f
		}
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(resource.NewSchemaless(
			attribute.String("service.name", "shop-api"),
		)),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(ratio))),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{},
	))
	return tp.Shutdown, nil
}
