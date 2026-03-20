package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	appports "github.com/rachelJG/event-notification-service/internal/application/ports"
	"github.com/rachelJG/event-notification-service/internal/application/usecases"
	"github.com/rachelJG/event-notification-service/internal/config"
	httpadapter "github.com/rachelJG/event-notification-service/internal/infrastructure/http"
	"github.com/rachelJG/event-notification-service/internal/infrastructure/logger"
	"github.com/rachelJG/event-notification-service/internal/infrastructure/postgres"
	"go.uber.org/zap"
)

// Version and Commit are injected at build time via -ldflags.
// Example: go build -ldflags="-X main.Version=1.0.0 -X main.Commit=abc1234"
var (
	Version = "dev"
	Commit  = "unknown"
)

// pgHealthChecker adapts *pgxpool.Pool to the httpadapter.HealthChecker interface.
type pgHealthChecker struct {
	pool *pgxpool.Pool
}

func (h pgHealthChecker) Ping(ctx context.Context) error {
	return h.pool.Ping(ctx)
}

func (h pgHealthChecker) Stats() httpadapter.DBStats {
	s := h.pool.Stat()
	return httpadapter.DBStats{
		MaxConns:      s.MaxConns(),
		TotalConns:    s.TotalConns(),
		IdleConns:     s.IdleConns(),
		AcquiredConns: s.AcquiredConns(),
	}
}

// app struct that holds the application components
type app struct {
	server          *http.Server
	db              *pgxpool.Pool
	logger          *zap.Logger
	tlsCertFile     string
	tlsKeyFile      string
	shutdownTimeout time.Duration
}

func newApp(ctx context.Context, cfg config.Config, log *zap.Logger) (*app, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.PGDSN)
	if err != nil {
		return nil, fmt.Errorf("parse pg dsn: %w", err)
	}
	poolCfg.MaxConns = cfg.DBPoolMaxConns
	poolCfg.MinConns = cfg.DBPoolMinConns
	poolCfg.MaxConnLifetime = cfg.DBPoolMaxConnLifetime
	poolCfg.MaxConnIdleTime = cfg.DBPoolMaxConnIdleTime

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, err
	}

	prometheus.MustRegister(postgres.NewPoolStatsCollector(pool))

	repo := postgres.EventRepository{Pool: pool, QueryTimeout: cfg.DBQueryTimeout}
	apiKeyRepo := postgres.APIKeyRepository{Pool: pool, QueryTimeout: cfg.DBQueryTimeout}

	var eventService appports.EventService = &usecases.EventService{Repo: repo}
	handler := httpadapter.Handler{EventService: eventService, Logger: log}
	adminHandler := httpadapter.AdminHandler{APIKeyRepo: apiKeyRepo, Logger: log}

	health := pgHealthChecker{pool: pool}
	opts := httpadapter.RouterOptions{
		MaxBodyBytes:        cfg.MaxBodyBytes,
		RateLimitRPS:        cfg.RateLimitRPS,
		RateLimitBurst:      cfg.RateLimitBurst,
		CORSAllowAllOrigins: cfg.CORSAllowAllOrigins,
		CORSAllowedOrigins:  cfg.CORSAllowedOrigins,
		CORSAllowedHeaders:  cfg.CORSAllowedHeaders,
		EnableHSTS:          cfg.EnableHSTS,
		HSTSMaxAgeSeconds:   cfg.HSTSMaxAgeSeconds,
		Version:             Version,
		Commit:              Commit,
		TrustedProxies:      cfg.TrustedProxies,
	}

	router := httpadapter.NewRouter(handler, adminHandler, health, apiKeyRepo, log, opts)
	server := &http.Server{
		Addr:              cfg.APIAddr,
		Handler:           router,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
	}

	return &app{server: server, db: pool, logger: log, tlsCertFile: cfg.TLSCertFile, tlsKeyFile: cfg.TLSKeyFile, shutdownTimeout: cfg.ShutdownTimeout}, nil
}

func (a *app) shutdown(ctx context.Context) error {
	inner := max(a.shutdownTimeout-5*time.Second, 5*time.Second)
	ctx, cancel := context.WithTimeout(ctx, inner)
	defer cancel()

	err := a.server.Shutdown(ctx)
	a.db.Close()
	_ = a.logger.Sync()
	return err
}

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "invalid config: %v\n", err)
		os.Exit(1)
	}
	ctx := context.Background()

	log, err := logger.New(cfg.AppEnv, cfg.LogLevel)
	if err != nil {
		zap.S().Fatalf("init logger: %v", err)
	}

	application, err := newApp(ctx, cfg, log)
	if err != nil {
		log.Fatal("init app", zap.Error(err))
	}

	// Start the server in a separate goroutine so that it doesn't block the main thread,
	// allowing us to wait for termination signals and perform a graceful shutdown.
	go func() {
		application.logger.Info("api listening", zap.String("addr", cfg.APIAddr))
		var serveErr error
		if application.tlsCertFile != "" {
			serveErr = application.server.ListenAndServeTLS(application.tlsCertFile, application.tlsKeyFile)
		} else {
			serveErr = application.server.ListenAndServe()
		}
		if serveErr != nil && serveErr != http.ErrServerClosed {
			application.logger.Fatal("server error", zap.Error(serveErr))
		}
	}()

	waitForShutdown(application)
}

func waitForShutdown(application *app) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), application.shutdownTimeout)
	defer cancel()

	if err := application.shutdown(ctx); err != nil {
		application.logger.Error("shutdown error", zap.Error(err))
	}
}
