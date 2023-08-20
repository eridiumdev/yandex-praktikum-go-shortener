package app

import (
	"context"
	"fmt"
	"log"
	nethttp "net/http"

	"github.com/gofiber/fiber/v2"

	"github.com/eridiumdev/yandex-praktikum-go-shortener/config"
	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/controller/http"
	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/infrastructure/repository"
	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/usecase"
)

type App struct {
	server     *fiber.App
	serverAddr string
}

func NewApp(ctx context.Context, cfg *config.Config) *App {
	app := &App{}

	server := fiber.New()
	app.server = server
	app.serverAddr = fmt.Sprintf(":%d", cfg.Server.Port)

	shortlinkRepo := repository.NewInMemShortlinkRepo()
	shortenerUC := usecase.NewShortener(cfg.Shortener, shortlinkRepo)

	http.NewShortenerController(server, shortenerUC)

	return app
}

func (a *App) Run(ctx context.Context) error {
	log.Printf("Listening on %s", a.serverAddr)
	if err := a.server.Listen(a.serverAddr); err != nil && err != nethttp.ErrServerClosed {
		return err
	}
	return nil
}

func (a *App) Stop(ctx context.Context) error {
	return a.server.Shutdown()
}
