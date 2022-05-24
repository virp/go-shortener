package storage

import (
	"strconv"
	"sync/atomic"
)

type inMemoryStorage struct {
	urls   map[string]ShortURL
	lastID int32
}

func NewInMemoryStorage() URLStorage {
	return &inMemoryStorage{
		urls:   make(map[string]ShortURL),
		lastID: 0,
	}
}

func (s *inMemoryStorage) Create(url ShortURL) (ShortURL, error) {
	newID := atomic.AddInt32(&s.lastID, 1)
	if url.ID == "" {
		url.ID = strconv.Itoa(int(newID))
	}

	if _, ok := s.urls[url.ID]; ok {
		return ShortURL{}, ErrAlreadyExist
	}

	s.urls[url.ID] = url

	return url, nil
}

func (s *inMemoryStorage) GetById(id string) (ShortURL, error) {
	url, ok := s.urls[id]
	if !ok {
		return ShortURL{}, ErrNotFound
	}

	return url, nil
}
