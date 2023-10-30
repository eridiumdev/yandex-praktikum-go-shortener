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
	//		   userID     linkID
	links map[string]map[string]*entity.Shortlink
	mutex sync.RWMutex
}

func NewInMemShortlinkRepo(backup storage.Storage) *InMemShortlinkRepo {
	return &InMemShortlinkRepo{
		backup: backup,
		links:  make(map[string]map[string]*entity.Shortlink),
		mutex:  sync.RWMutex{},
	}
}

func (r *InMemShortlinkRepo) Ping(ctx context.Context) error {
	return nil
}

func (r *InMemShortlinkRepo) SaveShortlink(ctx context.Context, link *entity.Shortlink) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, ok := r.links[link.UserID]; !ok {
		r.links[link.UserID] = make(map[string]*entity.Shortlink)
	}

	r.links[link.UserID][link.ID] = link

	return nil
}

func (r *InMemShortlinkRepo) FindShortlink(ctx context.Context, userID, linkID string) (*entity.Shortlink, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// If userID is not specified, search all links
	if userID == "" {
		for _, userLinks := range r.links {
			if link, ok := userLinks[linkID]; ok {
				return link, nil
			}
		}
		return nil, nil
	}

	if _, ok := r.links[userID]; !ok {
		return nil, nil
	}

	return r.links[userID][linkID], nil
}

func (r *InMemShortlinkRepo) GetShortlinks(ctx context.Context, userID string) ([]*entity.Shortlink, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if _, ok := r.links[userID]; !ok {
		return nil, nil
	}

	var links []*entity.Shortlink

	for _, link := range r.links[userID] {
		links = append(links, link)
	}
	return links, nil
}

func (r *InMemShortlinkRepo) Backup(ctx context.Context) error {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var links []*entity.Shortlink

	for _, userLinks := range r.links {
		for _, link := range userLinks {
			links = append(links, link)
		}
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
		if _, ok := r.links[link.UserID]; !ok {
			r.links[link.UserID] = make(map[string]*entity.Shortlink)
		}
		r.links[link.UserID][link.ID] = link
	}

	return nil
}

func (r *InMemShortlinkRepo) Close(ctx context.Context) error {
	return r.backup.Close(ctx)
}
