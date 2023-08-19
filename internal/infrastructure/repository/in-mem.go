package repository

import (
	"context"
	"sync"

	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/entity"
)

type InMemShortlinkRepo struct {
	links map[string]*entity.Shortlink
	mutex sync.RWMutex
}

func NewInMemShortlinkRepo() *InMemShortlinkRepo {
	return &InMemShortlinkRepo{
		links: make(map[string]*entity.Shortlink),
		mutex: sync.RWMutex{},
	}
}

func (r *InMemShortlinkRepo) SaveShortlink(ctx context.Context, link *entity.Shortlink) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.links[link.ID] = link
	return nil
}

func (r *InMemShortlinkRepo) FindShortlink(ctx context.Context, id string) (*entity.Shortlink, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return r.links[id], nil
}
