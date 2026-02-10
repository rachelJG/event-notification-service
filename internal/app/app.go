package app

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rachelJG/event-notification-service/internal/adapters/http"
	"github.com/rachelJG/event-notification-service/internal/adapters/postgres"
	"github.com/rachelJG/event-notification-service/internal/config"
	"github.com/rachelJG/event-notification-service/internal/core/usecases"
	"go.uber.org/zap"
)

type App struct {
	Server *http.Server
	DB     *pgxpool.Pool
	Logger *zap.Logger
}

func New(ctx context.Context, cfg config.Config, log *zap.Logger) (*App, error) {
	pool, err := pgxpool.New(ctx, cfg.PGDSN)
	if err != nil {
		return nil, err
	}

	repo := postgres.EventRepository{Pool: pool}
	uc := usecases.SubmitEvent{Repo: repo}
	handler := httpadapter.Handler{SubmitEvent: uc, Logger: log}

	router := httpadapter.NewRouter(handler, log)
	server := &http.Server{
		Addr:    cfg.APIAddr,
		Handler: router,
	}

	return &App{Server: server, DB: pool, Logger: log}, nil
}

func (a *App) Shutdown(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	err := a.Server.Shutdown(ctx)
	a.DB.Close()
	_ = a.Logger.Sync()
	return err
}
