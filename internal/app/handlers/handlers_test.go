package handlers

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/virp/go-shortener/internal/app/storage"
)

func TestHandlers_StoreURL(t *testing.T) {
	type want struct {
		statusCode int
		response   string
		shortID    string
	}

	tests := []struct {
		name     string
		handlers Handlers
		longURL  string
		want     want
	}{
		{
			name:     "should return short link",
			handlers: getHandlers([]storage.ShortURL{}),
			longURL:  "https://example.com/very/long/url/for/shortener",
			want: want{
				statusCode: http.StatusCreated,
				response:   "https://example.com/1",
				shortID:    "1",
			},
		},
		{
			name:     "should return bad request without provided url",
			handlers: getHandlers([]storage.ShortURL{}),
			longURL:  "",
			want: want{
				statusCode: http.StatusBadRequest,
				response:   http.StatusText(http.StatusBadRequest),
			},
		},
		{
			name:     "should return bad request with not valid url",
			handlers: getHandlers([]storage.ShortURL{}),
			longURL:  "not a url",
			want: want{
				statusCode: http.StatusBadRequest,
				response:   http.StatusText(http.StatusBadRequest),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody := bytes.NewBufferString(tt.longURL)
			req := httptest.NewRequest(http.MethodPost, "https://example.com", reqBody)
			w := httptest.NewRecorder()

			tt.handlers.StoreURL(w, req)
			res := w.Result()

			resBody, err := ioutil.ReadAll(res.Body)
			require.NoError(t, err)
			err = res.Body.Close()
			require.NoError(t, err)

			assert.Equal(t, tt.want.statusCode, res.StatusCode)
			assert.Equal(t, tt.want.response, strings.TrimRight(string(resBody), "\n"))

			if tt.want.shortID != "" {
				shortURL, err := tt.handlers.Storage.GetByID(tt.want.shortID)
				assert.NoError(t, err)
				assert.Equal(t, tt.longURL, shortURL.LongURL)
			}
		})
	}
}

func TestHandlers_GetURL(t *testing.T) {
	type want struct {
		statusCode     int
		locationHeader string
	}

	tests := []struct {
		name     string
		handlers Handlers
		request  string
		want     want
	}{
		{
			name: "should return long url",
			handlers: getHandlers([]storage.ShortURL{
				{
					ID:      "1",
					LongURL: "https://example.com/very/long/url/for/shortener",
				},
			}),
			request: "https://example.com/1",
			want: want{
				statusCode:     http.StatusTemporaryRedirect,
				locationHeader: "https://example.com/very/long/url/for/shortener",
			},
		},
		{
			name:     "should return 404 for non existed url",
			handlers: getHandlers([]storage.ShortURL{}),
			request:  "https://example.com/1",
			want: want{
				statusCode:     http.StatusNotFound,
				locationHeader: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.request, nil)
			w := httptest.NewRecorder()

			tt.handlers.GetURL(w, req)
			res := w.Result()

			assert.Equal(t, tt.want.statusCode, res.StatusCode)
			assert.Equal(t, tt.want.locationHeader, res.Header.Get("Location"))
		})
	}
}

func getHandlers(urls []storage.ShortURL) Handlers {
	s := storage.NewInMemoryStorage()

	for _, url := range urls {
		_, _ = s.Create(storage.ShortURL{ID: url.ID, LongURL: url.LongURL})
	}

	h := Handlers{
		Storage:  s,
		BaseHost: "https://example.com",
	}

	return h
}
