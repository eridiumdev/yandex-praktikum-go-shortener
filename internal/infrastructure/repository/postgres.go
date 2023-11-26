package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/eridiumdev/yandex-praktikum-go-shortener/config"
	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/entity"
	"github.com/eridiumdev/yandex-praktikum-go-shortener/internal/infrastructure/storage"
	"github.com/eridiumdev/yandex-praktikum-go-shortener/pkg/logger"
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
	log    *logger.Logger
}

func NewPostgresRepo(ctx context.Context, cfg config.PostgreSQL, backup storage.Storage, log *logger.Logger) (*PostgresRepo, error) {
	db, err := sql.Open("pgx", cfg.ConnString)
	if err != nil {
		return nil, log.Wrap(err, "open")
	}

	r := &PostgresRepo{
		cfg:    cfg,
		db:     db,
		backup: backup,
		log:    log,
	}

	err = r.Ping(context.Background())
	if err != nil {
		return nil, log.Wrap(err, "ping")
	}

	// Migrations
	migrator, err := migrate.New("file://"+cfg.MigrationsPath, cfg.ConnString)
	if err != nil {
		return nil, log.Wrap(err, "init migrator")
	}

	err = migrator.Up()
	if err == nil {
		log.Info(ctx).Msg("Migrations: done")
	} else {
		if !errors.Is(err, migrate.ErrNoChange) {
			return nil, log.Wrap(err, "migrations")
		}
		log.Info(ctx).Msg("Migrations: no changes")
	}

	// Statements
	err = r.initStatements(ctx)
	if err != nil {
		return nil, log.Wrap(err, "init statements")
	}

	return r, nil
}

func (r *PostgresRepo) Ping(ctx context.Context) error {
	cancelCtx, cancel := context.WithTimeout(ctx, r.cfg.PingTimeout)
	defer cancel()

	err := r.db.PingContext(cancelCtx)
	if err != nil {
		return r.log.Wrap(err, "ping")
	}
	return nil
}

func (r *PostgresRepo) SaveShortlink(ctx context.Context, link *entity.Shortlink) (*entity.Shortlink, error) {
	result := new(entity.Shortlink)
	var conflict bool

	row := saveShortlinkStmt.QueryRowContext(ctx, link.UID, link.UserUID, link.Short, link.Long, link.CorrelationID)
	err := row.Scan(&result.UID, &result.UserUID, &result.Short, &result.Long, &result.CorrelationID, &conflict)
	if err != nil {
		return nil, r.log.Wrap(err, "insert")
	}
	if conflict {
		return result, ErrURLConflict
	}
	return result, nil
}

func (r *PostgresRepo) SaveShortlinks(ctx context.Context, links []*entity.Shortlink) ([]*entity.Shortlink, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, r.log.Wrap(err, "begin tx")
	}

	var result []*entity.Shortlink

	for _, link := range links {
		resultLink := new(entity.Shortlink)
		var conflict bool

		row := tx.StmtContext(ctx, saveShortlinkStmt).QueryRowContext(ctx, link.UID, link.UserUID, link.Short, link.Long, link.CorrelationID)
		err := row.Scan(&resultLink.UID, &resultLink.UserUID, &resultLink.Short, &resultLink.Long, &resultLink.CorrelationID, &conflict)

		if err != nil {
			rollbackErr := tx.Rollback()
			if rollbackErr != nil {
				log.Printf("rollback after insert: %s", err)
			}
			return nil, r.log.Wrap(err, "insert")
		}

		result = append(result, resultLink)
	}

	err = tx.Commit()
	if err != nil {
		return nil, r.log.Wrap(err, "commit after insert")
	}

	return result, nil
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

	err := row.Scan(&link.UID, &link.UserUID, &link.Short, &link.Long, &link.Deleted, &corrID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, r.log.Wrap(err, "scan")
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
		return nil, r.log.Wrap(err, "select")
	}
	defer rows.Close()

	for rows.Next() {
		link := new(entity.Shortlink)
		var corrID sql.NullString
		err := rows.Scan(&link.UID, &link.UserUID, &link.Short, &link.Long, &link.Deleted, &corrID)
		if err != nil {
			return nil, r.log.Wrap(err, "scan")
		}
		link.CorrelationID = corrID.String
		links = append(links, link)
	}

	err = rows.Err()
	if err != nil {
		return nil, r.log.Wrap(err, "rows next")
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
		return nil, r.log.Wrap(err, "select")
	}
	defer rows.Close()

	for rows.Next() {
		link := new(entity.Shortlink)
		var corrID sql.NullString
		err := rows.Scan(&link.UID, &link.UserUID, &link.Short, &link.Long, &corrID)
		if err != nil {
			return nil, r.log.Wrap(err, "scan")
		}
		link.CorrelationID = corrID.String
		links = append(links, link)
	}

	err = rows.Err()
	if err != nil {
		return nil, r.log.Wrap(err, "rows next")
	}

	return links, nil
}

func (r *PostgresRepo) DeleteShortlinks(ctx context.Context, userUID string, linkUIDs []string) error {
	query := "UPDATE shortlinks SET deleted = true WHERE user_uid = $1 AND link_uid IN ("
	args := make([]any, len(linkUIDs)+1)
	args[0] = userUID

	for i := 0; i < len(linkUIDs); i++ {
		if i > 0 {
			query += ", "
		}
		query += fmt.Sprintf("$%d", i+2)
		args[i+1] = linkUIDs[i]
	}
	query += ")"

	_, err := r.db.ExecContext(ctx, query, args...)

	if err != nil {
		return r.log.Wrap(err, "update shortlinks deleted flag")
	}
	return nil
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
		return r.log.Wrap(err, "close backup")
	}

	err = r.closeStatements(ctx)
	if err != nil {
		return r.log.Wrap(err, "close statements")
	}

	err = r.db.Close()
	if err != nil {
		return r.log.Wrap(err, "close db")
	}

	return nil
}

func (r *PostgresRepo) initStatements(ctx context.Context) error {
	var err error

	saveShortlinkStmt, err = r.db.PrepareContext(ctx,
		"WITH inserted AS (INSERT INTO shortlinks(link_uid, user_uid, short, long, correlation_id) VALUES($1, $2, $3, $4, $5)"+
			" ON CONFLICT (long) DO NOTHING RETURNING link_uid, user_uid, short, long, correlation_id)"+
			", existing AS (SELECT link_uid, user_uid, short, long, correlation_id FROM shortlinks WHERE long = $4)"+
			" SELECT *, true AS conflict FROM existing UNION SELECT *, false AS conflict FROM inserted")
	if err != nil {
		return r.log.Wrap(err, "prepare saveShortlinkStmt")
	}
	findShortlinkStmt, err = r.db.PrepareContext(ctx, "SELECT link_uid, user_uid, short, long, deleted, correlation_id FROM shortlinks WHERE link_uid = $1")
	if err != nil {
		return r.log.Wrap(err, "prepare findShortlinkStmt")
	}
	findShortlinksStmt, err = r.db.PrepareContext(ctx, "SELECT link_uid, user_uid, short, long, deleted, correlation_id FROM shortlinks WHERE link_uid IN ($1)")
	if err != nil {
		return r.log.Wrap(err, "prepare findShortlinksStmt")
	}
	findShortlinkByUserStmt, err = r.db.PrepareContext(ctx, "SELECT link_uid, user_uid, short, long, correlation_id FROM shortlinks WHERE link_uid = $1 AND user_uid = $2 AND deleted = false")
	if err != nil {
		return r.log.Wrap(err, "prepare findShortlinkByUserStmt")
	}
	getShortlinksByUserStmt, err = r.db.PrepareContext(ctx, "SELECT link_uid, user_uid, short, long, correlation_id FROM shortlinks WHERE user_uid = $1 AND deleted = false")
	if err != nil {
		return r.log.Wrap(err, "prepare getShortlinksByUserStmt")
	}

	return nil
}

func (r *PostgresRepo) closeStatements(ctx context.Context) error {
	if err := saveShortlinkStmt.Close(); err != nil {
		return r.log.Wrap(err, "close saveShortlinkStmt")
	}
	if err := findShortlinkStmt.Close(); err != nil {
		return r.log.Wrap(err, "close findShortlinkStmt")
	}
	if err := findShortlinksStmt.Close(); err != nil {
		return r.log.Wrap(err, "close findShortlinksStmt")
	}
	if err := findShortlinkByUserStmt.Close(); err != nil {
		return r.log.Wrap(err, "close findShortlinkByUserStmt")
	}
	if err := getShortlinksByUserStmt.Close(); err != nil {
		return r.log.Wrap(err, "close getShortlinksByUserStmt")
	}
	return nil
}
