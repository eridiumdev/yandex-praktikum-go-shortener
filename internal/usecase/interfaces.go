package usecase

import (
	"context"

	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/entity"
)

type (
	CreateShortlinksIn struct {
		Links   []CreateShortlinksInLink
		UserUID string
		Length  int
	}
	CreateShortlinksInLink struct {
		URL           string
		CorrelationID string
	}
)

type Shortener interface {
	Ping(ctx context.Context) error

	CreateShortlink(ctx context.Context, userUID string, length int, url string) (*entity.Shortlink, error)
	CreateShortlinks(ctx context.Context, data CreateShortlinksIn) ([]*entity.Shortlink, error)
	GetShortlink(ctx context.Context, linkUID string) (*entity.Shortlink, error)
	GetUserShortlink(ctx context.Context, userUID, linkUID string) (*entity.Shortlink, error)
	ListUserShortlinks(ctx context.Context, userUID string) ([]*entity.Shortlink, error)
	DeleteUserShortlinks(ctx context.Context, userUID string, linkUIDs []string) error
}
