package logger

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Logger struct {
	logger zerolog.Logger
	label  string
}

func NewZerologLogger(ctx context.Context,
	label string,
	level string,
	pretty bool,
	writer io.Writer) *Logger {

	var zerologLogger zerolog.Logger

	zerolog.TimeFieldFormat = time.RFC3339Nano

	switch level {
	case "debug":
		zerologLogger = log.Level(zerolog.DebugLevel)
	default:
		zerologLogger = log.Level(zerolog.InfoLevel)
	}

	if pretty {
		zerologLogger = zerologLogger.Output(zerolog.ConsoleWriter{
			Out:        writer,
			TimeFormat: "15:04:05.000000",
		})
	} else {
		zerologLogger = zerologLogger.Output(writer)
	}

	l := &Logger{
		logger: zerologLogger,
		label:  label,
	}

	l.Info(ctx).Msgf("Logger started, level: %s, pretty: %t", level, pretty)

	return l
}

func (l *Logger) Debug(ctx context.Context) *zerolog.Event {
	return l.enrich(l.logger.Debug().Ctx(ctx))
}

func (l *Logger) Info(ctx context.Context) *zerolog.Event {
	return l.enrich(l.logger.Info().Ctx(ctx))
}

func (l *Logger) Warn(ctx context.Context) *zerolog.Event {
	return l.enrich(l.logger.Warn().Ctx(ctx))
}

func (l *Logger) Error(ctx context.Context, err error) *zerolog.Event {
	return l.enrich(l.logger.Error().Ctx(ctx).Err(err))
}

func (l *Logger) Fatal(ctx context.Context, err error) *zerolog.Event {
	return l.enrich(l.logger.Fatal().Ctx(ctx).Err(err))
}

func (l *Logger) enrich(e *zerolog.Event) *zerolog.Event {
	return e.Str("label", l.label)
}

func (l *Logger) Wrap(err error, msg string) error {
	return fmt.Errorf("[%s] %s - %w", l.label, msg, err)
}

func (l *Logger) Wrapf(err error, format string, v ...any) error {
	return fmt.Errorf("[%s] %s - %w", l.label, fmt.Sprintf(format, v...), err)
}

func (l *Logger) SubLogger(label string) *Logger {
	return &Logger{
		logger: l.logger,
		label:  label,
	}
}

func (l *Logger) RegisterHook(fn func(ctx context.Context) (string, string)) {
	l.logger = l.logger.Hook(hook{fn: fn})
}

type hook struct {
	fn func(ctx context.Context) (string, string)
}

func (h hook) Run(e *zerolog.Event, level zerolog.Level, msg string) {
	ctx := e.GetCtx()
	key, val := h.fn(ctx)
	if key != "" {
		e.Str(key, val)
	}
}
