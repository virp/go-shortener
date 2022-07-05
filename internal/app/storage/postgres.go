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
		"insert into urls (url, user_id) values ($1, $2) returning id",
		url.LongURL,
		url.UserID,
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
		"select id, url, user_id from urls where id = $1",
		id,
	)
	if r == nil {
		return ShortURL{}, ErrNotFound
	}
	var url ShortURL
	if err := r.Scan(&url.ID, &url.LongURL, &url.UserID); err != nil {
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
		"select id, url, user_id from urls where user_id = $1",
		userID,
	)
	if err != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var url ShortURL
		err = rows.Scan(&url.ID, &url.LongURL, &url.UserID)
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
