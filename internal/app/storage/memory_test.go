package storage

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemory_Create(t *testing.T) {
	tests := []struct {
		name     string
		storage  *memory
		url      ShortURL
		expectID string
	}{
		{
			name: "add first url",
			storage: &memory{
				urls: map[string]ShortURL{},
				mu:   new(sync.RWMutex),
			},
			url:      ShortURL{LongURL: "https://example.com/very/long/url/for/shortener"},
			expectID: "1",
		},
		{
			name: "add second url",
			storage: &memory{
				urls: map[string]ShortURL{
					"1": {
						ID:      "1",
						LongURL: "https://example.com/added/long/url",
					},
				},
				lastID: 1,
				mu:     new(sync.RWMutex),
			},
			url:      ShortURL{LongURL: "https://example.com/very/long/url/for/shortener"},
			expectID: "2",
		},
		{
			name: "add url with custom ID",
			storage: &memory{
				urls: map[string]ShortURL{},
				mu:   new(sync.RWMutex),
			},
			url: ShortURL{
				ID:      "custom",
				LongURL: "https://example.com/very/long/url/for/shortener",
			},
			expectID: "custom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, err := tt.storage.Create(context.Background(), tt.url)
			require.NoError(t, err)
			assert.Equal(t, tt.expectID, url.ID)
			assert.Contains(t, tt.storage.urls, url.ID)
		})
	}
}

func TestMemory_GetByID(t *testing.T) {
	tests := []struct {
		name          string
		storage       URLStorage
		ID            string
		expectedURL   string
		expectedError error
	}{
		{
			name: "get existed url",
			storage: &memory{
				urls: map[string]ShortURL{
					"1": {
						ID:      "1",
						LongURL: "https://example.com/existed/long/url",
					},
				},
				mu: new(sync.RWMutex),
			},
			ID:            "1",
			expectedURL:   "https://example.com/existed/long/url",
			expectedError: nil,
		},
		{
			name: "get non existed url",
			storage: &memory{
				urls: map[string]ShortURL{},
				mu:   new(sync.RWMutex),
			},
			ID:            "42",
			expectedURL:   "",
			expectedError: ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, err := tt.storage.GetByID(context.Background(), tt.ID)
			if tt.expectedError != nil {
				assert.ErrorIs(t, err, tt.expectedError)
				return
			}
			assert.Equal(t, tt.ID, url.ID)
			assert.Equal(t, tt.expectedURL, url.LongURL)
		})
	}
}
