package config

import (
	"flag"
	"time"

	"github.com/caarlos0/env/v6"
)

type (
	Config struct {
		App       App
		Server    Server
		Storage   Storage
		Shortener Shortener
	}
	App struct {
		ShutdownTimeout time.Duration
	}
	Server struct {
		Addr string `env:"SERVER_ADDRESS"`
	}
	Storage struct {
		Filepath string `env:"FILE_STORAGE_PATH"`
	}
	Shortener struct {
		BaseURL       string `env:"BASE_URL"`
		DefaultLength int
	}
)

func Load() (*Config, error) {
	cfg := &Config{
		App: App{
			ShutdownTimeout: time.Second * 3,
		},
		Shortener: Shortener{
			DefaultLength: 5,
		},
	}

	flag.StringVar(&cfg.Server.Addr, "a", ":8080", "server address")
	flag.StringVar(&cfg.Storage.Filepath, "f", "backup.json", "backup file path")
	flag.StringVar(&cfg.Shortener.BaseURL, "b", "http://localhost:8080", "shortlink base URL")
	flag.Parse()

	// Env vars take priority
	err := env.Parse(cfg)

	return cfg, err
}
