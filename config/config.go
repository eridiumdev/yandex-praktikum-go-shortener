package config

import (
	"flag"
	"time"

	"github.com/caarlos0/env/v6"
)

type (
	Config struct {
		App        App
		Logger     Logger
		Server     Server
		PostgreSQL PostgreSQL
		Storage    Storage
		Shortener  Shortener
	}
	App struct {
		ShutdownTimeout time.Duration
		AuthSecret      string
	}
	Logger struct {
		Level  string
		Pretty bool
	}
	Server struct {
		Addr string `env:"SERVER_ADDRESS"`
	}
	PostgreSQL struct {
		ConnString     string `env:"DATABASE_DSN"`
		MigrationsPath string
		PingTimeout    time.Duration
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
		Logger: Logger{
			Level:  "debug",
			Pretty: true,
		},
		PostgreSQL: PostgreSQL{
			PingTimeout:    time.Second,
			MigrationsPath: "./migrations",
		},
		Shortener: Shortener{
			DefaultLength: 5,
		},
	}

	flag.StringVar(&cfg.Server.Addr, "a", ":8080", "server address")
	flag.StringVar(&cfg.Storage.Filepath, "f", "backup.json", "backup file path")
	flag.StringVar(&cfg.Shortener.BaseURL, "b", "http://localhost:8080", "shortlink base URL")
	flag.StringVar(&cfg.PostgreSQL.ConnString, "d", "postgresql://postgres:qwerty123@127.0.0.1:15432/shortener?sslmode=disable", "database connection string")
	flag.Parse()

	// Env vars take priority
	err := env.Parse(cfg)

	return cfg, err
}
