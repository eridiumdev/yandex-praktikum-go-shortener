package usecase

import (
	"context"

	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/entity"
)

type Shortener interface {
	CreateShortlink(ctx context.Context, userID string, length int, url string) (*entity.Shortlink, error)
	GetShortlink(ctx context.Context, userID, linkID string) (*entity.Shortlink, error)
	ListShortlinks(ctx context.Context, userID string) ([]*entity.Shortlink, error)
}
