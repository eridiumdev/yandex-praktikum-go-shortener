package config

import "time"

type (
	Config struct {
		App       App
		Server    Server
		Shortener Shortener
	}
	App struct {
		ShutdownTimeout time.Duration
	}
	Server struct {
		Port int
	}
	Shortener struct {
		BaseURL       string
		DefaultLength int
	}
)
