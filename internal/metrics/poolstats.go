package metrics

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
)

// PoolStatsCollector exports pgxpool.Stat() gauges on scrape.
type PoolStatsCollector struct {
	pool *pgxpool.Pool

	totalConns    *prometheus.Desc
	idleConns     *prometheus.Desc
	acquiredConns *prometheus.Desc
	maxConns      *prometheus.Desc
	acquireCount  *prometheus.Desc
	acquireSecs   *prometheus.Desc
	emptyAcquire  *prometheus.Desc
}

func NewPoolStatsCollector(pool *pgxpool.Pool) *PoolStatsCollector {
	return &PoolStatsCollector{
		pool:          pool,
		totalConns:    prometheus.NewDesc("shop_db_pool_total_conns", "Total connections in the pool.", nil, nil),
		idleConns:     prometheus.NewDesc("shop_db_pool_idle_conns", "Idle connections in the pool.", nil, nil),
		acquiredConns: prometheus.NewDesc("shop_db_pool_acquired_conns", "Connections currently checked out.", nil, nil),
		maxConns:      prometheus.NewDesc("shop_db_pool_max_conns", "Configured max connections.", nil, nil),
		acquireCount:  prometheus.NewDesc("shop_db_pool_acquires_total", "Cumulative connection acquires.", nil, nil),
		acquireSecs:   prometheus.NewDesc("shop_db_pool_acquire_seconds_total", "Cumulative time spent acquiring connections.", nil, nil),
		emptyAcquire:  prometheus.NewDesc("shop_db_pool_empty_acquires_total", "Acquires that had to wait for a free connection.", nil, nil),
	}
}

func (c *PoolStatsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.totalConns
	ch <- c.idleConns
	ch <- c.acquiredConns
	ch <- c.maxConns
	ch <- c.acquireCount
	ch <- c.acquireSecs
	ch <- c.emptyAcquire
}

func (c *PoolStatsCollector) Collect(ch chan<- prometheus.Metric) {
	s := c.pool.Stat()
	ch <- prometheus.MustNewConstMetric(c.totalConns, prometheus.GaugeValue, float64(s.TotalConns()))
	ch <- prometheus.MustNewConstMetric(c.idleConns, prometheus.GaugeValue, float64(s.IdleConns()))
	ch <- prometheus.MustNewConstMetric(c.acquiredConns, prometheus.GaugeValue, float64(s.AcquiredConns()))
	ch <- prometheus.MustNewConstMetric(c.maxConns, prometheus.GaugeValue, float64(s.MaxConns()))
	ch <- prometheus.MustNewConstMetric(c.acquireCount, prometheus.CounterValue, float64(s.AcquireCount()))
	ch <- prometheus.MustNewConstMetric(c.acquireSecs, prometheus.CounterValue, s.AcquireDuration().Seconds())
	ch <- prometheus.MustNewConstMetric(c.emptyAcquire, prometheus.CounterValue, float64(s.EmptyAcquireCount()))
}
