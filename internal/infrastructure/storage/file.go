package storage

import (
	"context"
	"encoding/json"
	"os"

	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/entity"
)

type FileStorage struct {
	file *os.File
}

func NewFileStorage(filepath string) (*FileStorage, error) {
	if filepath == "" {
		return &FileStorage{}, nil
	}

	file, err := os.OpenFile(filepath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}

	return &FileStorage{
		file: file,
	}, nil
}

func (fs *FileStorage) Backup(ctx context.Context, links []*entity.Shortlink) error {
	if fs.file == nil {
		return nil
	}

	// Reset file
	err := fs.file.Truncate(0)
	if err != nil {
		return err
	}
	_, err = fs.file.Seek(0, 0)
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(fs.file)

	return encoder.Encode(links)
}

func (fs *FileStorage) Restore(ctx context.Context) ([]*entity.Shortlink, error) {
	var links []*entity.Shortlink

	if fs.file == nil {
		return links, nil
	}

	decoder := json.NewDecoder(fs.file)

	return links, decoder.Decode(&links)
}

func (fs *FileStorage) Close(ctx context.Context) error {
	if fs.file == nil {
		return nil
	}
	return fs.file.Close()
}
