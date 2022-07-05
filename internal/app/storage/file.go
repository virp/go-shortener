package storage

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"sync"
)

type file struct {
	urls   map[string]ShortURL
	lastID int
	mu     *sync.RWMutex
	f      *os.File
	w      *bufio.Writer
}

func NewFileStorage(filename string) (URLStorage, error) {
	f, err := os.OpenFile(filename, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0777)
	if err != nil {
		return nil, err
	}

	urls := make(map[string]ShortURL)
	var lastID int
	s := bufio.NewScanner(f)
	for s.Scan() {
		var url ShortURL
		err := json.Unmarshal(s.Bytes(), &url)
		if err != nil {
			return nil, err
		}
		urls[url.ID] = url

		if id, err := strconv.Atoi(url.ID); err == nil && id > lastID {
			lastID = id
		}
	}
	if err = s.Err(); err != nil {
		return nil, err
	}

	w := bufio.NewWriter(f)

	return &file{
		urls:   urls,
		lastID: lastID,
		mu:     new(sync.RWMutex),
		f:      f,
		w:      w,
	}, nil
}

func (s *file) Create(url ShortURL) (ShortURL, error) {
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

	data, err := json.Marshal(url)
	if err != nil {
		return ShortURL{}, err
	}
	if _, err := s.w.Write(data); err != nil {
		return ShortURL{}, err
	}
	if err = s.w.WriteByte('\n'); err != nil {
		return ShortURL{}, err
	}
	if err = s.w.Flush(); err != nil {
		return ShortURL{}, err
	}

	return url, nil
}

func (s *file) GetByID(id string) (ShortURL, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	url, ok := s.urls[id]
	if !ok {
		return ShortURL{}, ErrNotFound
	}

	return url, nil
}

func (s *file) FindByUserID(userID string) []ShortURL {
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

func (s *file) CreateBatch(urls []ShortURL) ([]ShortURL, error) {
	createdUrls := make([]ShortURL, 0, len(urls))
	for _, u := range urls {
		cu, err := s.Create(u)
		if err != nil {
			return nil, fmt.Errorf("create url: %w", err)
		}
		createdUrls = append(createdUrls, cu)
	}

	return createdUrls, nil
}
