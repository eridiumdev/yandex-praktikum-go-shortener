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

	cfg := &config.Config{
		App: config.App{
			ShutdownTimeout: time.Second * 3,
		},
		Server: config.Server{
			Port: 8080,
		},
		Shortener: config.Shortener{
			BaseURL:       "http://localhost:8080/",
			DefaultLength: 5,
		},
	}

	a := app.NewApp(ctx, cfg)

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
	err := a.Stop(ctx)
	if err != nil {
		log.Fatalf("Error stopping app: %s", err)
	}

	log.Printf("App stopped")
}
