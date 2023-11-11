package repository

import (
	"context"
	"database/sql"
	"log"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
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

	err = r.Ping(context.Background())
	if err != nil {
		return nil, errors.Wrapf(err, "[postgres] ping")
	}

	// Migrations
	migrator, err := migrate.New("file://"+cfg.MigrationsPath, cfg.ConnString)
	if err != nil {
		return nil, errors.Wrapf(err, "[postgres] init migrator")
	}

	err = migrator.Up()
	if err == nil {
		log.Printf("[postgres] migrations: done")
	} else {
		if !errors.Is(err, migrate.ErrNoChange) {
			return nil, errors.Wrapf(err, "[postgres] migrations")
		}
		log.Printf("[postgres] migrations: no changes")
	}

	return r, nil
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
	stmt, err := r.db.PrepareContext(ctx, "INSERT INTO shortlinks(link_uid, user_uid, short, long) VALUES($1, $2, $3, $4)")
	if err != nil {
		return errors.Wrapf(err, "[postgres] prepare stmt")
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, link.UID, link.UserUID, link.Short, link.Long)
	if err != nil {
		return errors.Wrapf(err, "[postgres] insert")
	}
	return nil
}

func (r *PostgresRepo) FindShortlink(ctx context.Context, userUID, linkUID string) (*entity.Shortlink, error) {
	query := "SELECT link_uid, user_uid, short, long FROM shortlinks WHERE link_uid = $1"
	if userUID != "" {
		query += " AND user_uid = $2"
	}

	stmt, err := r.db.PrepareContext(ctx, query)
	if err != nil {
		return nil, errors.Wrapf(err, "[postgres] prepare stmt")
	}
	defer stmt.Close()

	var row *sql.Row

	if userUID != "" {
		row = stmt.QueryRowContext(ctx, linkUID, userUID)
	} else {
		row = stmt.QueryRowContext(ctx, linkUID)
	}

	link := new(entity.Shortlink)

	err = row.Scan(&link.UID, &link.UserUID, &link.Short, &link.Long)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "[postgres] scan")
	}

	return link, nil
}

func (r *PostgresRepo) GetShortlinks(ctx context.Context, userUID string) ([]*entity.Shortlink, error) {
	var links []*entity.Shortlink

	stmt, err := r.db.PrepareContext(ctx,
		"SELECT link_uid, user_uid, short, long FROM shortlinks WHERE user_uid = $1")
	if err != nil {
		return nil, errors.Wrapf(err, "[postgres] prepare stmt")
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, userUID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return links, nil
		}
		return nil, errors.Wrapf(err, "[postgres] select")
	}
	defer rows.Close()

	for rows.Next() {
		link := new(entity.Shortlink)
		err := rows.Scan(&link.UID, &link.UserUID, &link.Short, &link.Long)
		if err != nil {
			return nil, errors.Wrapf(err, "[postgres] scan")
		}
		links = append(links, link)
	}

	err = rows.Err()
	if err != nil {
		return nil, errors.Wrapf(err, "[postgres] rows next")
	}

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
