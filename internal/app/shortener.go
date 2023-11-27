package app

import (
	"context"
	"errors"
	nethttp "net/http"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"

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
	server *nethttp.Server
	repo   repository.ShortlinkRepo
	log    *logger.Logger
}

func NewShortener(ctx context.Context, cfg *config.Config, log *logger.Logger) (*Shortener, error) {
	app := &Shortener{
		log: log,
	}

	handler := gin.New()
	handler.ContextWithFallback = true

	handler.Use(gin.Recovery())
	handler.Use(gzip.Gzip(gzip.BestSpeed, gzip.WithDecompressFn(gzip.DefaultDecompressHandle)))

	cipher, err := crypto.NewAES256(cfg.App.AuthSecret, log)
	if err != nil {
		return nil, log.Wrap(err, "init crypto cipher")
	}

	authMiddleware := middleware.CookieAuth(middleware.CookieAuthConfig{
		Cipher: cipher,
	}, log)

	handler.Use(authMiddleware)

	handler.Use(middleware.RequestID(log))
	handler.Use(middleware.Logger(log.SubLogger("http_requests")))

	handler.Handler()
	app.server = &nethttp.Server{
		Handler: handler,
		//ReadTimeout:  60 * time.Second,
		//WriteTimeout: 60 * time.Second,
		Addr: cfg.Server.Addr,
	}

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
	http.NewShortenerController(handler, shortenerUC, log.SubLogger("shortener_controller"))

	return app, nil
}

func (a *Shortener) Run(ctx context.Context) error {
	a.log.Info(ctx).Msgf("Listening on %s", a.server.Addr)
	if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, nethttp.ErrServerClosed) {
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

	err = a.server.Shutdown(ctx)
	if err != nil {
		return a.log.Wrap(err, "shutdown server")
	}

	return nil
}
