package storage

import (
	"errors"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrAlreadyExist = errors.New("url ID already exist")
)

type URLStorage interface {
	Create(ShortURL) (ShortURL, error)
	GetByID(string) (ShortURL, error)
	FindByUserID(string) []ShortURL
}
