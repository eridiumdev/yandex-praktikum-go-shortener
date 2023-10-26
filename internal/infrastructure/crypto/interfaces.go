package crypto

import (
	"context"

	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/entity"
)

type EncryptorDecryptor interface {
	Encrypt(context.Context, *entity.AuthToken) (string, error)
	Decrypt(context.Context, string) (*entity.AuthToken, error)
}
