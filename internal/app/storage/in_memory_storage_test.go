package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInMemoryStorage_Create(t *testing.T) {
	tests := []struct {
		name     string
		storage  *inMemoryStorage
		url      ShortURL
		expectId string
	}{
		{
			name: "add first url",
			storage: &inMemoryStorage{
				urls: map[string]ShortURL{},
			},
			url:      ShortURL{LongURL: "https://example.com/very/long/url/for/shortener"},
			expectId: "1",
		},
		{
			name: "add second url",
			storage: &inMemoryStorage{
				urls: map[string]ShortURL{
					"1": {
						ID:      "1",
						LongURL: "https://example.com/added/long/url",
					},
				},
				lastID: 1,
			},
			url:      ShortURL{LongURL: "https://example.com/very/long/url/for/shortener"},
			expectId: "2",
		},
		{
			name: "add url with custom ID",
			storage: &inMemoryStorage{
				urls: map[string]ShortURL{},
			},
			url: ShortURL{
				ID:      "custom",
				LongURL: "https://example.com/very/long/url/for/shortener",
			},
			expectId: "custom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, err := tt.storage.Create(tt.url)
			require.NoError(t, err)
			assert.Equal(t, tt.expectId, url.ID)
			assert.Contains(t, tt.storage.urls, url.ID)
		})
	}
}

func TestInMemoryStorage_GetById(t *testing.T) {
	tests := []struct {
		name          string
		storage       URLStorage
		ID            string
		expectedURL   string
		expectedError error
	}{
		{
			name: "get existed url",
			storage: &inMemoryStorage{
				urls: map[string]ShortURL{
					"1": {
						ID:      "1",
						LongURL: "https://example.com/existed/long/url",
					},
				},
			},
			ID:            "1",
			expectedURL:   "https://example.com/existed/long/url",
			expectedError: nil,
		},
		{
			name: "get non existed url",
			storage: &inMemoryStorage{
				urls: map[string]ShortURL{},
			},
			ID:            "42",
			expectedURL:   "",
			expectedError: ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, err := tt.storage.GetByID(tt.ID)
			if tt.expectedError != nil {
				assert.ErrorIs(t, err, tt.expectedError)
				return
			}
			assert.Equal(t, tt.ID, url.ID)
			assert.Equal(t, tt.expectedURL, url.LongURL)
		})
	}
}
