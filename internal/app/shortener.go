package app

import (
	"context"
	"errors"
	nethttp "net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"

	"github.com/eridiumdev/yandex-praktikum-go-shortener/config"
	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/controller/http"
	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/controller/http/middleware"
	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/infrastructure/crypto"
	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/infrastructure/repository"
	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/infrastructure/storage"
	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/usecase"
	"github.com/eridiumdev/yandex-praktikum-go-shortener/pkg/logger"
)

type Shortener struct {
	server     *fiber.App
	serverAddr string

	repo repository.ShortlinkRepo
	log  *logger.Logger
}

func NewShortener(ctx context.Context, cfg *config.Config, log *logger.Logger) (*Shortener, error) {
	app := &Shortener{
		log: log,
	}

	server := fiber.New()
	server.Use(compress.New(compress.Config{
		Level: compress.LevelBestSpeed,
	}))
	server.Use(middleware.Logger(log.SubLogger("http_requests")))

	cipher, err := crypto.NewAES256(cfg.App.AuthSecret, log)
	if err != nil {
		return nil, log.Wrap(err, "init crypto cipher")
	}

	authMiddleware, err := middleware.CookieAuth(middleware.CookieAuthConfig{
		Cipher: cipher,
	}, log)
	if err != nil {
		return nil, log.Wrap(err, "init auth middleware")
	}

	server.Use(authMiddleware)

	app.server = server
	app.serverAddr = cfg.Server.Addr

	backup, err := storage.NewFileStorage(cfg.Storage.Filepath)
	if err != nil {
		return nil, log.Wrap(err, "init backup storage")
	}
	log.Info(ctx).Msgf("Initialized backup storage @ %s", cfg.Storage.Filepath)

	var shortlinkRepo repository.ShortlinkRepo
	shortlinkRepo, err = repository.NewPostgresRepo(ctx, cfg.PostgreSQL, backup, log.SubLogger("shortlink_repo"))
	if err != nil {
		log.Error(ctx, err).Msg("init shortlink repo")
		// Fallback to in-mem repo
		shortlinkRepo = repository.NewInMemShortlinkRepo(backup)
		log.Info(ctx).Msgf("Initialized shortlink repo @ in-mem")
	} else {
		log.Info(ctx).Msgf("Initialized shortlink repo @ %s", cfg.PostgreSQL.ConnString)
	}
	app.repo = shortlinkRepo

	err = shortlinkRepo.Restore(ctx)
	if err != nil {
		return nil, log.Wrap(err, "restore from backup")
	}
	log.Info(ctx).Msgf("Restore from backup complete")

	shortenerUC := usecase.NewShortener(cfg.Shortener, shortlinkRepo, log.SubLogger("shortener_uc"))
	http.NewShortenerController(server, shortenerUC, log.SubLogger("shortener_controller"))

	return app, nil
}

func (a *Shortener) Run(ctx context.Context) error {
	a.log.Info(ctx).Msgf("Listening on %s", a.serverAddr)
	if err := a.server.Listen(a.serverAddr); err != nil && !errors.Is(err, nethttp.ErrServerClosed) {
		return err
	}
	return nil
}

func (a *Shortener) Stop(ctx context.Context) error {
	err := a.repo.Backup(ctx)
	if err != nil {
		return a.log.Wrap(err, "backup repo")
	}

	err = a.repo.Close(ctx)
	if err != nil {
		return a.log.Wrap(err, "close repo")
	}

	err = a.server.Shutdown()
	if err != nil {
		return a.log.Wrap(err, "shutdown server")
	}

	return nil
}
