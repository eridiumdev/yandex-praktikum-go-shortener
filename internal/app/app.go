package app

import (
	"context"
	"log"
	nethttp "net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/pkg/errors"

	"github.com/eridiumdev/yandex-praktikum-go-shortener/config"
	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/controller/http"
	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/controller/http/middleware"
	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/infrastructure/crypto"
	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/infrastructure/repository"
	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/infrastructure/storage"
	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/usecase"
)

type App struct {
	server     *fiber.App
	serverAddr string

	repo repository.ShortlinkRepo
}

func NewApp(ctx context.Context, cfg *config.Config) (*App, error) {
	app := &App{}

	server := fiber.New()
	server.Use(compress.New(compress.Config{
		Level: compress.LevelBestSpeed,
	}))

	cipher, err := crypto.NewAES256(cfg.App.AuthSecret)
	if err != nil {
		return nil, errors.Wrap(err, "initing crypto cipher")
	}

	authMiddleware, err := middleware.CookieAuth(middleware.CookieAuthConfig{
		Cipher: cipher,
	})
	if err != nil {
		return nil, errors.Wrap(err, "initing auth middleware")
	}

	server.Use(authMiddleware)

	app.server = server
	app.serverAddr = cfg.Server.Addr

	backup, err := storage.NewFileStorage(cfg.Storage.Filepath)
	if err != nil {
		return nil, errors.Wrap(err, "initing backup storage")
	}

	shortlinkRepo := repository.NewInMemShortlinkRepo(backup)
	app.repo = shortlinkRepo

	err = shortlinkRepo.Restore(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "restoring from backup")
	}

	shortenerUC := usecase.NewShortener(cfg.Shortener, shortlinkRepo)
	http.NewShortenerController(server, shortenerUC)

	return app, nil
}

func (a *App) Run(ctx context.Context) error {
	log.Printf("Listening on %s", a.serverAddr)
	if err := a.server.Listen(a.serverAddr); err != nil && err != nethttp.ErrServerClosed {
		return err
	}
	return nil
}

func (a *App) Stop(ctx context.Context) error {
	err := a.repo.Backup(ctx)
	if err != nil {
		return errors.Wrap(err, "saving links to backup")
	}

	err = a.repo.Close(ctx)
	if err != nil {
		return errors.Wrap(err, "closing repo")
	}

	err = a.server.Shutdown()
	if err != nil {
		return errors.Wrap(err, "shutting down server")
	}

	return nil
}
