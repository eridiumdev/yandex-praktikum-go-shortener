package app

import (
	"context"
	"fmt"
	"log"
	nethttp "net/http"

	"github.com/eridiumdev/yandex-praktikum-go-shortener/config"
	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/controller/http"
	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/infrastructure/repository"
	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/usecase"
)

type App struct {
	server *nethttp.Server
}

func NewApp(ctx context.Context, cfg *config.Config) *App {
	app := &App{}

	handler := http.NewHandler()
	server := &nethttp.Server{
		Addr: fmt.Sprintf(":%d", cfg.Server.Port),
	}
	server.Handler = handler
	app.server = server

	shortlinkRepo := repository.NewInMemShortlinkRepo()
	shortenerUC := usecase.NewShortener(cfg.Shortener, shortlinkRepo)

	http.NewShortenerController(handler.Router, shortenerUC)

	return app
}

func (a *App) Run(ctx context.Context) error {
	log.Printf("Listening on %s", a.server.Addr)
	if err := a.server.ListenAndServe(); err != nil && err != nethttp.ErrServerClosed {
		return err
	}
	return nil
}

func (a *App) Stop(ctx context.Context) error {
	return a.server.Shutdown(ctx)
}
