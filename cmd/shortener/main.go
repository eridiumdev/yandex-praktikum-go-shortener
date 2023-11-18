package main

import (
	"context"
	"errors"
	stdlog "log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/eridiumdev/yandex-praktikum-go-shortener/config"
	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/app"
	"github.com/eridiumdev/yandex-praktikum-go-shortener/pkg/logger"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		stdlog.Fatalf("Error loading config: %s", err)
	}

	log := logger.NewZerologLogger(ctx, "app", cfg.Logger.Level, cfg.Logger.Pretty, os.Stdout)

	appl, err := app.NewShortener(ctx, cfg, log)
	if err != nil {
		log.Fatal(ctx, err).Msg("Init app")
	}

	go func() {
		log.Info(ctx).Msg("Starting app...")
		err := appl.Run(ctx)
		if err != nil {
			log.Fatal(ctx, err).Msg("Run app")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Info(ctx).Msgf("OS signal received: %s", sig)

	time.AfterFunc(cfg.App.ShutdownTimeout, func() {
		log.Fatal(ctx, errors.New("shutdown timeout")).Msg("App force-stopped")
	})

	log.Info(ctx).Msg("Stopping app...")
	err = appl.Stop(ctx)
	if err != nil {
		log.Fatal(ctx, err).Msg("Stop app")
	}

	log.Info(ctx).Msg("App stopped")
}
