package storage

import (
	"strconv"
	"sync"
)

type memory struct {
	urls   map[string]ShortURL
	lastID int
	mu     *sync.RWMutex
}

func NewMemoryStorage() URLStorage {
	return &memory{
		urls:   make(map[string]ShortURL),
		lastID: 0,
		mu:     new(sync.RWMutex),
	}
}

func (s *memory) Create(url ShortURL) (ShortURL, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.lastID = s.lastID + 1
	if url.ID == "" {
		url.ID = strconv.Itoa(s.lastID)
	}

	if _, ok := s.urls[url.ID]; ok {
		return ShortURL{}, ErrAlreadyExist
	}

	s.urls[url.ID] = url

	return url, nil
}

func (s *memory) GetByID(id string) (ShortURL, error) {
	//s.mu.RLock()
	//defer s.mu.RUnlock()

	url, ok := s.urls[id]
	if !ok {
		return ShortURL{}, ErrNotFound
	}

	return url, nil
}
