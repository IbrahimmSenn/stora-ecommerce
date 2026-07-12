package metrics

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/prometheus/client_golang/prometheus"
)

type queryStartKey struct{}

type queryStart struct {
	at time.Time
	op string
}

// QueryTracer implements pgx.QueryTracer, timing every query by SQL verb.
// Set on pgxpool.Config.ConnConfig.Tracer — no repository code involved.
type QueryTracer struct {
	duration *prometheus.HistogramVec
	total    *prometheus.CounterVec
}

func NewQueryTracer(reg prometheus.Registerer) *QueryTracer {
	t := &QueryTracer{
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "shop_db_query_duration_seconds",
			Help:    "Database query duration by SQL operation.",
			Buckets: []float64{.001, .0025, .005, .01, .025, .05, .1, .25, .5, 1, 2.5},
		}, []string{"operation"}),
		total: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "shop_db_queries_total",
			Help: "Database queries by SQL operation.",
		}, []string{"operation"}),
	}
	reg.MustRegister(t.duration, t.total)
	return t
}

func (t *QueryTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	return context.WithValue(ctx, queryStartKey{}, queryStart{at: time.Now(), op: sqlOperation(data.SQL)})
}

func (t *QueryTracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, _ pgx.TraceQueryEndData) {
	start, ok := ctx.Value(queryStartKey{}).(queryStart)
	if !ok {
		return
	}
	t.total.WithLabelValues(start.op).Inc()
	t.duration.WithLabelValues(start.op).Observe(time.Since(start.at).Seconds())
}

// sqlOperation maps a statement to a bounded label set by its leading keyword.
func sqlOperation(sql string) string {
	fields := strings.Fields(sql)
	if len(fields) == 0 {
		return "other"
	}
	switch op := strings.ToLower(fields[0]); op {
	case "select", "insert", "update", "delete", "begin", "commit":
		return op
	case "with":
		return "select"
	default:
		return "other"
	}
}
