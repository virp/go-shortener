package storage

import (
	"context"
	"errors"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrAlreadyExist = errors.New("url already exist")
)

type URLStorage interface {
	Create(context.Context, ShortURL) (ShortURL, error)
	GetByID(context.Context, string) (ShortURL, error)
	FindByUserID(context.Context, string) []ShortURL
	CreateBatch(context.Context, []ShortURL) ([]ShortURL, error)
}
