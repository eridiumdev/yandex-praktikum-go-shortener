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
	//		  userUID   linkUID
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

func (r *InMemShortlinkRepo) SaveShortlink(ctx context.Context, link *entity.Shortlink) (*entity.Shortlink, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, ok := r.links[link.UserUID]; !ok {
		r.links[link.UserUID] = make(map[string]*entity.Shortlink)
	}

	r.links[link.UserUID][link.UID] = link

	return link, nil
}

func (r *InMemShortlinkRepo) SaveShortlinks(ctx context.Context, links []*entity.Shortlink) ([]*entity.Shortlink, error) {
	for _, link := range links {
		if _, err := r.SaveShortlink(ctx, link); err != nil {
			return nil, err
		}
	}

	return links, nil
}

func (r *InMemShortlinkRepo) FindShortlink(ctx context.Context, userUID, linkUID string) (*entity.Shortlink, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// If userUID is not specified, search all links
	if userUID == "" {
		for _, userLinks := range r.links {
			if link, ok := userLinks[linkUID]; ok {
				return link, nil
			}
		}
		return nil, nil
	}

	if _, ok := r.links[userUID]; !ok {
		return nil, nil
	}

	return r.links[userUID][linkUID], nil
}

func (r *InMemShortlinkRepo) FindShortlinks(ctx context.Context, linkUIDs []string) ([]*entity.Shortlink, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var links []*entity.Shortlink

	for _, linkUID := range linkUIDs {
	next:
		for _, userLinks := range r.links {
			if link, ok := userLinks[linkUID]; ok {
				links = append(links, link)
				goto next
			}
		}
	}

	return links, nil
}

func (r *InMemShortlinkRepo) GetShortlinks(ctx context.Context, userUID string) ([]*entity.Shortlink, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if _, ok := r.links[userUID]; !ok {
		return nil, nil
	}

	var links []*entity.Shortlink

	for _, link := range r.links[userUID] {
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
		if _, ok := r.links[link.UserUID]; !ok {
			r.links[link.UserUID] = make(map[string]*entity.Shortlink)
		}
		r.links[link.UserUID][link.UID] = link
	}

	return nil
}

func (r *InMemShortlinkRepo) Close(ctx context.Context) error {
	return r.backup.Close(ctx)
}
