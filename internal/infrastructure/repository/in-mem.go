package repository

import (
	"context"
	"errors"
	"io"
	"sync"

	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/entity"
	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/infrastructure/storage"
)

type InMemShortlinkRepo struct {
	backup storage.Storage
	links  map[string]*entity.Shortlink
	mutex  sync.RWMutex
}

func NewInMemShortlinkRepo(backup storage.Storage) *InMemShortlinkRepo {
	return &InMemShortlinkRepo{
		backup: backup,
		links:  make(map[string]*entity.Shortlink),
		mutex:  sync.RWMutex{},
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

func (r *InMemShortlinkRepo) Backup(ctx context.Context) error {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var links []*entity.Shortlink

	for _, link := range r.links {
		links = append(links, link)
	}

	return r.backup.Backup(ctx, links)
}

func (r *InMemShortlinkRepo) Restore(ctx context.Context) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	links, err := r.backup.Restore(ctx)
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}

	for _, link := range links {
		r.links[link.ID] = link
	}

	return nil
}

func (r *InMemShortlinkRepo) Close(ctx context.Context) error {
	return r.backup.Close(ctx)
}
