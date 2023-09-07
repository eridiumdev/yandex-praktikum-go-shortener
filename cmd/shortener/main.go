package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/eridiumdev/yandex-praktikum-go-shortener/config"
	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/app"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Error loading config: %s", err)
	}

	a, err := app.NewApp(ctx, cfg)
	if err != nil {
		log.Printf("Error initing app: %s", err)
	}

	go func() {
		log.Printf("Starting app...")
		err := a.Run(ctx)
		if err != nil {
			log.Fatalf("Error running app: %s", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Printf("OS signal received: %s", sig)

	time.AfterFunc(cfg.App.ShutdownTimeout, func() {
		log.Fatalf("App force-stopped (shutdown timeout)")
	})

	log.Printf("Stopping app...")
	err = a.Stop(ctx)
	if err != nil {
		log.Fatalf("Error stopping app: %s", err)
	}

	log.Printf("App stopped")
}
