package crypto

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/entity"
)

type EncryptorDecryptor interface {
	Encrypt(context.Context, *entity.AuthToken) (string, error)
	Decrypt(context.Context, string) (*entity.AuthToken, error)
}

const pkg = "crypto"

func log(ctx context.Context, e *zerolog.Event, module, format string, v ...any) {
	e.Ctx(ctx).Str("pkg", pkg).Str("mod", module).Msgf(format, v...)
}

func wrap(err error, module, format string, v ...any) error {
	return fmt.Errorf("[%s] [%s] %s - %w", pkg, module, fmt.Sprintf(format, v...), err)
}
