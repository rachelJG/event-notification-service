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
	appports "github.com/rachelJG/event-notification-service/internal/application/ports"
	"github.com/rachelJG/event-notification-service/internal/application/usecases"
	"github.com/rachelJG/event-notification-service/internal/config"
	httpadapter "github.com/rachelJG/event-notification-service/internal/infrastructure/http"
	"github.com/rachelJG/event-notification-service/internal/infrastructure/logger"
	"github.com/rachelJG/event-notification-service/internal/infrastructure/postgres"
	"go.uber.org/zap"
)

// pgHealthChecker adapts *pgxpool.Pool to the ports.HealthChecker interface.
type pgHealthChecker struct {
	pool *pgxpool.Pool
}

func (h pgHealthChecker) Ping(ctx context.Context) error {
	return h.pool.Ping(ctx)
}

// app struct that holds the application components
type app struct {
	server *http.Server
	db     *pgxpool.Pool
	logger *zap.Logger
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

	repo := postgres.EventRepository{Pool: pool}
	var uc appports.SubmitEventUseCase = &usecases.SubmitEvent{Repo: repo}
	handler := httpadapter.Handler{SubmitEvent: uc, Logger: log}

	health := pgHealthChecker{pool: pool}
	opts := httpadapter.RouterOptions{
		JWTSecret:           cfg.JWTSecret,
		MaxBodyBytes:        cfg.MaxBodyBytes,
		RateLimitRPS:        cfg.RateLimitRPS,
		RateLimitBurst:      cfg.RateLimitBurst,
		CORSAllowAllOrigins: cfg.CORSAllowAllOrigins,
		CORSAllowedOrigins:  cfg.CORSAllowedOrigins,
		CORSAllowedHeaders:  cfg.CORSAllowedHeaders,
		EnableHSTS:          cfg.EnableHSTS,
		HSTSMaxAgeSeconds:   cfg.HSTSMaxAgeSeconds,
	}

	router := httpadapter.NewRouter(handler, health, log, opts)
	server := &http.Server{
		Addr:              cfg.APIAddr,
		Handler:           router,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
	}

	return &app{server: server, db: pool, logger: log}, nil
}

func (a *app) shutdown(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
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

	go func() {
		application.logger.Info("api listening", zap.String("addr", cfg.APIAddr))
		if err := application.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			application.logger.Fatal("server error", zap.Error(err))
		}
	}()

	waitForShutdown(application)
}

func waitForShutdown(application *app) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := application.shutdown(ctx); err != nil {
		application.logger.Error("shutdown error", zap.Error(err))
	}
}
