package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rachelJG/event-notification-service/internal/application/usecases"
	"github.com/rachelJG/event-notification-service/internal/config"
	"github.com/rachelJG/event-notification-service/internal/infrastructure/email"
	"github.com/rachelJG/event-notification-service/internal/infrastructure/logger"
	"github.com/rachelJG/event-notification-service/internal/infrastructure/postgres"
	"go.uber.org/zap"
)

var (
	Version = "dev"
	Commit  = "unknown"
)

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "invalid config: %v\n", err)
		os.Exit(1)
	}

	log, err := logger.New(cfg.AppEnv, cfg.LogLevel)
	if err != nil {
		zap.S().Fatalf("init logger: %v", err)
	}
	defer func() { _ = log.Sync() }()

	log.Info("starting worker",
		zap.String("version", Version),
		zap.String("commit", Commit),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	poolCfg, err := pgxpool.ParseConfig(cfg.PGDSN)
	if err != nil {
		log.Fatal("parse pg dsn", zap.Error(err))
	}
	poolCfg.MaxConns = cfg.DBPoolMaxConns
	poolCfg.MinConns = cfg.DBPoolMinConns
	poolCfg.MaxConnLifetime = cfg.DBPoolMaxConnLifetime
	poolCfg.MaxConnIdleTime = cfg.DBPoolMaxConnIdleTime

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		log.Fatal("connect to database", zap.Error(err))
	}

	eventRepo := postgres.EventRepository{Pool: pool, QueryTimeout: cfg.DBQueryTimeout}
	notifRepo := postgres.NotificationRepository{Pool: pool, QueryTimeout: cfg.DBQueryTimeout}
	sender := email.NewSMTPSender(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPassword, cfg.SMTPFrom)

	processUC := usecases.ProcessEvents{
		EventRepo:        eventRepo,
		NotificationRepo: notifRepo,
		BatchSize:        cfg.WorkerBatchSize,
	}
	deliverUC := usecases.DeliverNotifications{
		NotificationRepo: notifRepo,
		Sender:           sender,
		BatchSize:        cfg.WorkerBatchSize,
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		runLoop(ctx, log, "process", cfg.WorkerProcessInterval, func(ctx context.Context) {
			count, err := processUC.Handle(ctx)
			if err != nil {
				log.Error("process events error", zap.Error(err))
				return
			}
			if count > 0 {
				log.Info("processed events", zap.Int("count", count))
			}
		})
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		runLoop(ctx, log, "deliver", cfg.WorkerDeliverInterval, func(ctx context.Context) {
			count, err := deliverUC.Handle(ctx)
			if err != nil {
				log.Error("deliver notifications error", zap.Error(err))
				return
			}
			if count > 0 {
				log.Info("delivered notifications", zap.Int("count", count))
			}
		})
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	sig := <-stop
	log.Info("received shutdown signal", zap.String("signal", sig.String()))

	cancel()
	wg.Wait()
	pool.Close()
	log.Info("worker stopped")
}

func runLoop(ctx context.Context, log *zap.Logger, name string, interval time.Duration, fn func(context.Context)) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Info("loop started", zap.String("loop", name), zap.Duration("interval", interval))

	for {
		select {
		case <-ctx.Done():
			log.Info("loop stopped", zap.String("loop", name))
			return
		case <-ticker.C:
			fn(ctx)
		}
	}
}
