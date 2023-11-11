package repository

import (
	"context"

	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/entity"
)

type ShortlinkRepo interface {
	SaveShortlink(ctx context.Context, link *entity.Shortlink) error
	FindShortlink(ctx context.Context, userUID, linkUID string) (*entity.Shortlink, error)
	GetShortlinks(ctx context.Context, userUID string) ([]*entity.Shortlink, error)

	Ping(ctx context.Context) error

	Backup(ctx context.Context) error
	Restore(ctx context.Context) error
	Close(ctx context.Context) error
}
