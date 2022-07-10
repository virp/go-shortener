package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

type postgres struct {
	db      *sqlx.DB
	timeout time.Duration
}

func NewPostgresStorage(db *sqlx.DB, timeout time.Duration) (URLStorage, error) {
	return &postgres{
		db:      db,
		timeout: timeout,
	}, nil
}

func (s *postgres) Create(ctx context.Context, url ShortURL) (ShortURL, error) {
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	rows, err := s.db.NamedQueryContext(
		ctx,
		"insert into urls (url, user_id, correlation_id) values (:url, :user_id, :correlation_id) on conflict on constraint urls_url_key do nothing returning id",
		&url,
	)
	if err != nil {
		return ShortURL{}, fmt.Errorf("insert url to DB: %w", err)
	}
	if rows.Next() {
		err = rows.Scan(&url.ID)
		if err != nil {
			return ShortURL{}, fmt.Errorf("get inserted url id: %w", err)
		}
		return url, nil
	}

	err = s.db.GetContext(
		ctx,
		&url,
		"select id, url, user_id, correlation_id from urls where url = $1 limit 1",
		url.LongURL,
	)
	if err != nil {
		return ShortURL{}, fmt.Errorf("get duplicated url: %w", err)
	}

	return url, ErrAlreadyExist
}

func (s *postgres) GetByID(ctx context.Context, id string) (ShortURL, error) {
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	var url ShortURL
	err := s.db.GetContext(ctx, &url, "select id, url, user_id, correlation_id from urls where id = $1", id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ShortURL{}, ErrNotFound
		}
		return ShortURL{}, fmt.Errorf("get url: %w", err)
	}

	return url, nil
}

func (s *postgres) FindByUserID(ctx context.Context, userID string) []ShortURL {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	var urls []ShortURL

	err := s.db.SelectContext(
		ctx,
		&urls,
		"select id, url, user_id, correlation_id from urls where user_id = $1",
		userID,
	)
	if err != nil {
		return nil
	}

	return urls
}

func (s *postgres) CreateBatch(ctx context.Context, urls []ShortURL) ([]ShortURL, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("create tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PreparexContext(
		ctx,
		"insert into urls (url, user_id, correlation_id) values ($1, $2, $3) returning id",
	)
	if err != nil {
		return nil, fmt.Errorf("prepare stmt: %w", err)
	}
	defer func() { _ = stmt.Close() }()

	createdUrls := make([]ShortURL, 0, len(urls))
	for _, u := range urls {
		err := stmt.QueryRowxContext(
			ctx,
			u.LongURL,
			u.UserID,
			u.CorrelationID,
		).Scan(&u.ID)
		if err != nil {
			return nil, fmt.Errorf("insert url to DB: %w", err)
		}
		createdUrls = append(createdUrls, u)
	}

	err = tx.Commit()
	if err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return createdUrls, nil
}
