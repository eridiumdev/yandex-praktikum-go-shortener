package logger

import (
	"io"

	"github.com/rs/zerolog/log"
)

func NewMockLogger() *Logger {
	return &Logger{
		logger: log.Output(io.Discard),
	}
}
