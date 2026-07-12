package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/otel/trace"
)

// AccessLog logs one slog record per request. With LOG_FORMAT=json the output
// is machine-parseable, which is what Promtail/Loki index in the monitoring
// stack (replaces chi's plain-text Logger).
func AccessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww, ok := w.(chimw.WrapResponseWriter)
		if !ok {
			ww = chimw.NewWrapResponseWriter(w, r.ProtoMajor)
		}
		start := time.Now()
		next.ServeHTTP(ww, r)

		route := ""
		if rc := chi.RouteContext(r.Context()); rc != nil {
			route = rc.RoutePattern()
		}
		status := ww.Status()
		if status == 0 {
			status = http.StatusOK
		}
		attrs := []any{
			"request_id", chimw.GetReqID(r.Context()),
			"method", r.Method,
			"path", r.URL.Path,
			"route", route,
			"status", status,
			"duration_ms", float64(time.Since(start).Microseconds())/1000.0,
			"ip", clientIP(r),
			"user_agent", r.UserAgent(),
			"bytes", ww.BytesWritten(),
		}
		// With tracing on, trace_id makes the log line clickable into Tempo
		// (Loki derived field). OTel API only — no-op span context otherwise.
		if sc := trace.SpanContextFromContext(r.Context()); sc.IsValid() {
			attrs = append(attrs, "trace_id", sc.TraceID().String(), "span_id", sc.SpanID().String())
		}
		slog.Info("http_request", attrs...)
	})
}
