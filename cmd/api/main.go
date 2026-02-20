package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rachelJG/event-notification-service/internal/application"
	"github.com/rachelJG/event-notification-service/internal/infrastructure/config"
	"github.com/rachelJG/event-notification-service/internal/infrastructure/logger"
	"go.uber.org/zap"
)

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

	app, err := application.New(ctx, cfg, log)
	if err != nil {
		log.Fatal("init app", zap.Error(err))
	}

	go func() {
		app.Logger.Info("api listening", zap.String("addr", cfg.APIAddr))
		if err := app.Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			app.Logger.Fatal("server error", zap.Error(err))
		}
	}()

	waitForShutdown(app)
}

func waitForShutdown(app *application.App) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := app.Shutdown(ctx); err != nil {
		app.Logger.Error("shutdown error", zap.Error(err))
	}
}
