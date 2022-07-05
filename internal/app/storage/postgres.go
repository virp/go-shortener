package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type postgres struct {
	db *sql.DB
}

func NewPostgresStorage(db *sql.DB) (URLStorage, error) {
	return &postgres{
		db: db,
	}, nil
}

func (s *postgres) Create(url ShortURL) (ShortURL, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := s.db.QueryRowContext(
		ctx,
		"insert into urls (url, user_id, correlation_id) values ($1, $2, $3) returning id",
		url.LongURL,
		url.UserID,
		url.CorrelationID,
	).Scan(&url.ID)
	if err != nil {
		return ShortURL{}, fmt.Errorf("insert url to DB: %w", err)
	}

	return url, nil
}

func (s *postgres) GetByID(id string) (ShortURL, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	r := s.db.QueryRowContext(
		ctx,
		"select id, url, user_id, correlation_id from urls where id = $1",
		id,
	)
	if r == nil {
		return ShortURL{}, ErrNotFound
	}
	var url ShortURL
	if err := r.Scan(&url.ID, &url.LongURL, &url.UserID, &url.CorrelationID); err != nil {
		return ShortURL{}, fmt.Errorf("scan row: %w", err)
	}

	return url, nil
}

func (s *postgres) FindByUserID(userID string) []ShortURL {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var urls []ShortURL

	rows, err := s.db.QueryContext(
		ctx,
		"select id, url, user_id, correlation_id from urls where user_id = $1",
		userID,
	)
	if err != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var url ShortURL
		err = rows.Scan(&url.ID, &url.LongURL, &url.UserID, &url.CorrelationID)
		if err != nil {
			return nil
		}
		urls = append(urls, url)
	}

	err = rows.Err()
	if err != nil {
		return nil
	}

	return urls
}

func (s *postgres) CreateBatch(urls []ShortURL) ([]ShortURL, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("create tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(
		ctx,
		"insert into urls (url, user_id, correlation_id) values ($1, $2, $3) returning id",
	)
	if err != nil {
		return nil, fmt.Errorf("prepare stmt: %w", err)
	}
	defer func() { _ = stmt.Close() }()

	createdUrls := make([]ShortURL, 0, len(urls))
	for _, u := range urls {
		err := stmt.QueryRowContext(
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
