package repository

import (
	"context"
	"database/sql"
	"log"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pkg/errors"

	"github.com/eridiumdev/yandex-praktikum-go-shortener/config"
	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/entity"
	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/infrastructure/storage"
)

var (
	saveShortlinkStmt       *sql.Stmt
	findShortlinkStmt       *sql.Stmt
	findShortlinksStmt      *sql.Stmt
	findShortlinkByUserStmt *sql.Stmt
	getShortlinksByUserStmt *sql.Stmt
)

type PostgresRepo struct {
	cfg    config.PostgreSQL
	db     *sql.DB
	backup storage.Storage
}

func NewPostgresRepo(ctx context.Context, cfg config.PostgreSQL, backup storage.Storage) (*PostgresRepo, error) {
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

	// Statements
	err = r.initStatements(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "[postgres] init statements")
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
	_, err := saveShortlinkStmt.ExecContext(ctx, link.UID, link.UserUID, link.Short, link.Long)
	if err != nil {
		return errors.Wrapf(err, "[postgres] insert")
	}
	return nil
}

func (r *PostgresRepo) SaveShortlinks(ctx context.Context, links []*entity.Shortlink) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrapf(err, "[postgres] begin tx")
	}

	for _, link := range links {
		_, err := tx.StmtContext(ctx, saveShortlinkStmt).ExecContext(ctx, link.UID, link.UserUID, link.Short, link.Long, link.CorrelationID)
		if err != nil {
			rollbackErr := tx.Rollback()
			if rollbackErr != nil {
				log.Printf("[postgres] rollback after insert: %s", err)
			}
			return errors.Wrapf(err, "[postgres] insert")
		}
	}

	err = tx.Commit()
	if err != nil {
		return errors.Wrapf(err, "[postgres] commit after insert")
	}

	return nil
}

func (r *PostgresRepo) FindShortlink(ctx context.Context, userUID, linkUID string) (*entity.Shortlink, error) {
	var row *sql.Row

	if userUID != "" {
		row = findShortlinkByUserStmt.QueryRowContext(ctx, linkUID, userUID)
	} else {
		row = findShortlinkStmt.QueryRowContext(ctx, linkUID)
	}

	link := new(entity.Shortlink)
	var corrID sql.NullString

	err := row.Scan(&link.UID, &link.UserUID, &link.Short, &link.Long, &corrID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "[postgres] scan")
	}

	link.CorrelationID = corrID.String

	return link, nil
}

func (r *PostgresRepo) FindShortlinks(ctx context.Context, linkUIDs []string) ([]*entity.Shortlink, error) {
	var links []*entity.Shortlink

	rows, err := findShortlinksStmt.QueryContext(ctx, strings.Join(linkUIDs, ","))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return links, nil
		}
		return nil, errors.Wrapf(err, "[postgres] select")
	}
	defer rows.Close()

	for rows.Next() {
		link := new(entity.Shortlink)
		var corrID sql.NullString
		err := rows.Scan(&link.UID, &link.UserUID, &link.Short, &link.Long, &corrID)
		if err != nil {
			return nil, errors.Wrapf(err, "[postgres] scan")
		}
		link.CorrelationID = corrID.String
		links = append(links, link)
	}

	err = rows.Err()
	if err != nil {
		return nil, errors.Wrapf(err, "[postgres] rows next")
	}

	return links, nil
}

func (r *PostgresRepo) GetShortlinks(ctx context.Context, userUID string) ([]*entity.Shortlink, error) {
	var links []*entity.Shortlink

	rows, err := getShortlinksByUserStmt.QueryContext(ctx, userUID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return links, nil
		}
		return nil, errors.Wrapf(err, "[postgres] select")
	}
	defer rows.Close()

	for rows.Next() {
		link := new(entity.Shortlink)
		var corrID sql.NullString
		err := rows.Scan(&link.UID, &link.UserUID, &link.Short, &link.Long, &corrID)
		if err != nil {
			return nil, errors.Wrapf(err, "[postgres] scan")
		}
		link.CorrelationID = corrID.String
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
		return errors.Wrapf(err, "[postgres] close backup")
	}

	err = r.closeStatements(ctx)
	if err != nil {
		return errors.Wrapf(err, "[postgres] close statements")
	}

	err = r.db.Close()
	if err != nil {
		return errors.Wrapf(err, "[postgres] close db")
	}

	return nil
}

func (r *PostgresRepo) initStatements(ctx context.Context) error {
	var err error

	saveShortlinkStmt, err = r.db.PrepareContext(ctx, "INSERT INTO shortlinks(link_uid, user_uid, short, long, correlation_id) VALUES($1, $2, $3, $4, $5)")
	if err != nil {
		return errors.Wrapf(err, "[postgres] prepare saveShortlinkStmt")
	}
	findShortlinkStmt, err = r.db.PrepareContext(ctx, "SELECT link_uid, user_uid, short, long, correlation_id FROM shortlinks WHERE link_uid = $1")
	if err != nil {
		return errors.Wrapf(err, "[postgres] prepare findShortlinkStmt")
	}
	findShortlinksStmt, err = r.db.PrepareContext(ctx, "SELECT link_uid, user_uid, short, long, correlation_id FROM shortlinks WHERE link_uid IN ($1)")
	if err != nil {
		return errors.Wrapf(err, "[postgres] prepare findShortlinksStmt")
	}
	findShortlinkByUserStmt, err = r.db.PrepareContext(ctx, "SELECT link_uid, user_uid, short, long, correlation_id FROM shortlinks WHERE link_uid = $1 AND user_uid = $2")
	if err != nil {
		return errors.Wrapf(err, "[postgres] prepare findShortlinkByUserStmt")
	}
	getShortlinksByUserStmt, err = r.db.PrepareContext(ctx, "SELECT link_uid, user_uid, short, long, correlation_id FROM shortlinks WHERE user_uid = $1")
	if err != nil {
		return errors.Wrapf(err, "[postgres] prepare getShortlinksByUserStmt")
	}

	return nil
}

func (r *PostgresRepo) closeStatements(ctx context.Context) error {
	if err := saveShortlinkStmt.Close(); err != nil {
		return errors.Wrapf(err, "[postgres] close saveShortlinkStmt")
	}
	if err := findShortlinkStmt.Close(); err != nil {
		return errors.Wrapf(err, "[postgres] close findShortlinkStmt")
	}
	if err := findShortlinksStmt.Close(); err != nil {
		return errors.Wrapf(err, "[postgres] close findShortlinksStmt")
	}
	if err := findShortlinkByUserStmt.Close(); err != nil {
		return errors.Wrapf(err, "[postgres] close findShortlinkByUserStmt")
	}
	if err := getShortlinksByUserStmt.Close(); err != nil {
		return errors.Wrapf(err, "[postgres] close getShortlinksByUserStmt")
	}
	return nil
}
