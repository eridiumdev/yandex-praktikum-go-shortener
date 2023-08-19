package repository

import (
	"context"

	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/entity"
)

type ShortlinkRepo interface {
	SaveShortlink(ctx context.Context, link *entity.Shortlink) error
	FindShortlink(ctx context.Context, id string) (*entity.Shortlink, error)
}
