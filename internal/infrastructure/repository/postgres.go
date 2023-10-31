package repository

import (
	"context"
	"database/sql"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pkg/errors"

	"github.com/eridiumdev/yandex-praktikum-go-shortener/config"
	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/entity"
	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/infrastructure/storage"
)

type PostgresRepo struct {
	cfg    config.PostgreSQL
	db     *sql.DB
	backup storage.Storage
}

func NewPostgresRepo(cfg config.PostgreSQL, backup storage.Storage) (*PostgresRepo, error) {
	db, err := sql.Open("pgx", cfg.ConnString)
	if err != nil {
		return nil, errors.Wrapf(err, "[postgres] open")
	}

	r := &PostgresRepo{
		cfg:    cfg,
		db:     db,
		backup: backup,
	}

	return r, r.Ping(context.Background())
}

func (r *PostgresRepo) Ping(ctx context.Context) error {
	cancelCtx, cancel := context.WithTimeout(ctx, r.cfg.PingTimeout)
	defer cancel()

	err := r.db.PingContext(cancelCtx)
	if err != nil {
		return errors.Wrapf(err, "[postgres] ping")
	}
	return nil
}

func (r *PostgresRepo) SaveShortlink(ctx context.Context, link *entity.Shortlink) error {
	return nil
}

func (r *PostgresRepo) FindShortlink(ctx context.Context, userID, linkID string) (*entity.Shortlink, error) {
	return nil, nil
}

func (r *PostgresRepo) GetShortlinks(ctx context.Context, userID string) ([]*entity.Shortlink, error) {
	var links []*entity.Shortlink

	return links, nil
}

func (r *PostgresRepo) Backup(ctx context.Context) error {
	return nil
}

func (r *PostgresRepo) Restore(ctx context.Context) error {
	return nil
}

func (r *PostgresRepo) Close(ctx context.Context) error {
	err := r.backup.Close(ctx)
	if err != nil {
		return err
	}

	err = r.db.Close()
	if err != nil {
		return err
	}

	return nil
}
