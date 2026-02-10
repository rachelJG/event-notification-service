package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rachelJG/event-notification-service/internal/adapters/logger"
	"github.com/rachelJG/event-notification-service/internal/app"
	"github.com/rachelJG/event-notification-service/internal/config"
	"go.uber.org/zap"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	log, err := logger.New(cfg.AppEnv, cfg.LogLevel)
	if err != nil {
		zap.S().Fatalf("init logger: %v", err)
	}

	application, err := app.New(ctx, cfg, log)
	if err != nil {
		log.Fatal("init app", zap.Error(err))
	}

	go func() {
		application.Logger.Info("api listening", zap.String("addr", cfg.APIAddr))
		if err := application.Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			application.Logger.Fatal("server error", zap.Error(err))
		}
	}()

	waitForShutdown(application)
}

func waitForShutdown(application *app.App) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := application.Shutdown(ctx); err != nil {
		application.Logger.Error("shutdown error", zap.Error(err))
	}
}
