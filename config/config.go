package config

import (
	"flag"
	"time"

	"github.com/caarlos0/env/v6"
)

type (
	Config struct {
		App        App
		Server     Server
		PostgreSQL PostgreSQL
		Storage    Storage
		Shortener  Shortener
	}
	App struct {
		ShutdownTimeout time.Duration
		AuthSecret      string
	}
	Server struct {
		Addr string `env:"SERVER_ADDRESS"`
	}
	PostgreSQL struct {
		ConnString  string `env:"DATABASE_DSN"`
		PingTimeout time.Duration
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
			AuthSecret:      "U2ahPqQAQiWUxfdT7SDBNFrgcGFkJ6Tq",
		},
		PostgreSQL: PostgreSQL{
			PingTimeout: time.Second,
		},
		Shortener: Shortener{
			DefaultLength: 5,
		},
	}

	flag.StringVar(&cfg.Server.Addr, "a", ":8080", "server address")
	flag.StringVar(&cfg.Storage.Filepath, "f", "backup.json", "backup file path")
	flag.StringVar(&cfg.Shortener.BaseURL, "b", "http://localhost:8080", "shortlink base URL")
	flag.StringVar(&cfg.PostgreSQL.ConnString, "d", "postgresql://postgres:qwerty123@127.0.0.1:15432/shortener", "database connection string")
	flag.Parse()

	// Env vars take priority
	err := env.Parse(cfg)

	return cfg, err
}
