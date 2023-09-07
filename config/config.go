package config

import (
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
		Addr string `env:"SERVER_ADDRESS" envDefault:":8080"`
	}
	Storage struct {
		Filepath string `env:"FILE_STORAGE_PATH"`
	}
	Shortener struct {
		BaseURL       string `env:"BASE_URL" envDefault:"http://localhost:8080"`
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

	err := env.Parse(cfg)

	return cfg, err
}
