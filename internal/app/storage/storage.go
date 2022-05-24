package storage

import (
	"errors"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrAlreadyExist = errors.New("url ID already exist")
)

type UrlStorage interface {
	Create(ShortURL) (ShortURL, error)
	GetById(string) (ShortURL, error)
}
