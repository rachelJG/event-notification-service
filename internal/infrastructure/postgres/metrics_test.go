package postgres

import (
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
)

func TestNewPoolStatsCollector(t *testing.T) {
	// pgxpool.New with an invalid DSN still returns a pool (lazy connect).
	pool, err := pgxpool.New(t.Context(), "postgres://invalid:5432/nonexistent?connect_timeout=1")
	if err != nil {
		t.Fatalf("unexpected error creating pool: %v", err)
	}
	defer pool.Close()

	collector := NewPoolStatsCollector(pool)
	if collector == nil {
		t.Fatal("expected non-nil collector")
	}
	if collector.pool != pool {
		t.Fatal("expected collector to hold the provided pool")
	}
}

func TestPoolStatsCollectorDescribe(t *testing.T) {
	pool, err := pgxpool.New(t.Context(), "postgres://invalid:5432/nonexistent?connect_timeout=1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer pool.Close()

	collector := NewPoolStatsCollector(pool)
	ch := make(chan *prometheus.Desc, 10)
	collector.Describe(ch)
	close(ch)

	count := 0
	for range ch {
		count++
	}
	if count != 6 {
		t.Errorf("expected 6 descriptors, got %d", count)
	}
}

func TestPoolStatsCollectorCollect(t *testing.T) {
	pool, err := pgxpool.New(t.Context(), "postgres://invalid:5432/nonexistent?connect_timeout=1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer pool.Close()

	collector := NewPoolStatsCollector(pool)
	ch := make(chan prometheus.Metric, 10)
	collector.Collect(ch)
	close(ch)

	count := 0
	for range ch {
		count++
	}
	if count != 6 {
		t.Errorf("expected 6 metrics, got %d", count)
	}
}
