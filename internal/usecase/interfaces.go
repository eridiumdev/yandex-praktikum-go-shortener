package usecase

import (
	"context"

	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/entity"
)

type Shortener interface {
	CreateShortlink(ctx context.Context, length int, url string) (*entity.Shortlink, error)
	GetShortlink(ctx context.Context, id string) (*entity.Shortlink, error)
}
