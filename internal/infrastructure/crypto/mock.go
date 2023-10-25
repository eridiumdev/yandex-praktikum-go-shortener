package crypto

import (
	"context"

	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/entity"
)

type Mock struct{}

func NewMock() *Mock {
	return &Mock{}
}

func (m *Mock) Encrypt(ctx context.Context, token *entity.AuthToken) (string, error) {
	// Return as plaintext
	return token.UserID, nil
}

func (m *Mock) Decrypt(ctx context.Context, encrypted string) (*entity.AuthToken, error) {
	return &entity.AuthToken{
		UserID: encrypted,
	}, nil
}
