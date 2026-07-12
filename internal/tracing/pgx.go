package tracing

import (
	"context"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5"
)

// pgx accepts a single QueryTracer on the pool config, so the Prometheus
// tracer and the OTel span tracer are composed here. Safe because each
// tracer stashes its state under its own context key.
type multiQueryTracer struct {
	tracers []pgx.QueryTracer
}

func (m *multiQueryTracer) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	for _, t := range m.tracers {
		ctx = t.TraceQueryStart(ctx, conn, data)
	}
	return ctx
}

func (m *multiQueryTracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	for _, t := range m.tracers {
		t.TraceQueryEnd(ctx, conn, data)
	}
}

// WithPgxTracing returns base unchanged when tracing is off, otherwise base
// composed with otelpgx so every query also emits a child span.
func WithPgxTracing(base pgx.QueryTracer, enabled bool) pgx.QueryTracer {
	if !enabled {
		return base
	}
	return &multiQueryTracer{tracers: []pgx.QueryTracer{base, otelpgx.NewTracer()}}
}
