package storage

import (
	"context"
	"fmt"
	"strconv"
	"sync"
)

type memory struct {
	urls   map[string]ShortURL
	lastID int
	mu     *sync.RWMutex
}

func NewMemoryStorage() (URLStorage, error) {
	return &memory{
		urls:   make(map[string]ShortURL),
		lastID: 0,
		mu:     new(sync.RWMutex),
	}, nil
}

func (s *memory) Create(ctx context.Context, url ShortURL) (ShortURL, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.lastID = s.lastID + 1
	if url.ID == "" {
		url.ID = strconv.Itoa(s.lastID)
	}

	if u, ok := s.urls[url.ID]; ok {
		return u, ErrAlreadyExist
	}

	s.urls[url.ID] = url

	return url, nil
}

func (s *memory) GetByID(ctx context.Context, id string) (ShortURL, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	url, ok := s.urls[id]
	if !ok {
		return ShortURL{}, ErrNotFound
	}

	return url, nil
}

func (s *memory) FindByUserID(ctx context.Context, userID string) []ShortURL {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var urls []ShortURL

	for _, url := range s.urls {
		if url.UserID == userID {
			urls = append(urls, url)
		}
	}

	return urls
}

func (s *memory) CreateBatch(ctx context.Context, urls []ShortURL) ([]ShortURL, error) {
	createdUrls := make([]ShortURL, 0, len(urls))
	for _, u := range urls {
		cu, err := s.Create(ctx, u)
		if err != nil {
			return nil, fmt.Errorf("create url: %w", err)
		}
		createdUrls = append(createdUrls, cu)
	}

	return createdUrls, nil
}

func (s *memory) DeleteBatch(ctx context.Context, userID string, ids []string) error {
	return nil
}
