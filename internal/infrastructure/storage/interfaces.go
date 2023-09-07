package storage

import (
	"context"

	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/entity"
)

type Storage interface {
	Backup(ctx context.Context, links []*entity.Shortlink) error
	Restore(ctx context.Context) ([]*entity.Shortlink, error)
	Close(ctx context.Context) error
}
