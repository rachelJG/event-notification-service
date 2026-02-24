package postgres

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
)

// PoolStatsCollector implements prometheus.Collector and exposes pgxpool connection stats.
type PoolStatsCollector struct {
	pool *pgxpool.Pool

	totalConns    *prometheus.Desc
	idleConns     *prometheus.Desc
	acquiredConns *prometheus.Desc
	maxConns      *prometheus.Desc
	acquireCount  *prometheus.Desc
	emptyAcquire  *prometheus.Desc
}

// NewPoolStatsCollector creates a Prometheus collector for pgxpool stats.
func NewPoolStatsCollector(pool *pgxpool.Pool) *PoolStatsCollector {
	return &PoolStatsCollector{
		pool: pool,
		totalConns: prometheus.NewDesc(
			"pgxpool_connections_total",
			"Total number of connections in the pool",
			nil, nil,
		),
		idleConns: prometheus.NewDesc(
			"pgxpool_connections_idle",
			"Number of idle connections in the pool",
			nil, nil,
		),
		acquiredConns: prometheus.NewDesc(
			"pgxpool_connections_acquired",
			"Number of acquired connections in the pool",
			nil, nil,
		),
		maxConns: prometheus.NewDesc(
			"pgxpool_connections_max",
			"Maximum number of connections allowed in the pool",
			nil, nil,
		),
		acquireCount: prometheus.NewDesc(
			"pgxpool_acquire_total",
			"Total number of connection acquires from the pool",
			nil, nil,
		),
		emptyAcquire: prometheus.NewDesc(
			"pgxpool_empty_acquire_total",
			"Total number of acquires that had to create a new connection",
			nil, nil,
		),
	}
}

func (c *PoolStatsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.totalConns
	ch <- c.idleConns
	ch <- c.acquiredConns
	ch <- c.maxConns
	ch <- c.acquireCount
	ch <- c.emptyAcquire
}

func (c *PoolStatsCollector) Collect(ch chan<- prometheus.Metric) {
	s := c.pool.Stat()
	ch <- prometheus.MustNewConstMetric(c.totalConns, prometheus.GaugeValue, float64(s.TotalConns()))
	ch <- prometheus.MustNewConstMetric(c.idleConns, prometheus.GaugeValue, float64(s.IdleConns()))
	ch <- prometheus.MustNewConstMetric(c.acquiredConns, prometheus.GaugeValue, float64(s.AcquiredConns()))
	ch <- prometheus.MustNewConstMetric(c.maxConns, prometheus.GaugeValue, float64(s.MaxConns()))
	ch <- prometheus.MustNewConstMetric(c.acquireCount, prometheus.CounterValue, float64(s.AcquireCount()))
	ch <- prometheus.MustNewConstMetric(c.emptyAcquire, prometheus.CounterValue, float64(s.EmptyAcquireCount()))
}
