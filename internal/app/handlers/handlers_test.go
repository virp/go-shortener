package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
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
				shortURL, err := tt.handlers.Storage.GetByID(context.Background(), tt.want.shortID)
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
		shortID  string
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
			shortID: "1",
			want: want{
				statusCode:     http.StatusTemporaryRedirect,
				locationHeader: "https://example.com/very/long/url/for/shortener",
			},
		},
		{
			name:     "should return 404 for non existed url",
			handlers: getHandlers([]storage.ShortURL{}),
			shortID:  "1",
			want: want{
				statusCode:     http.StatusNotFound,
				locationHeader: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "https://example.com/"+tt.shortID, nil)
			rCtx := chi.NewRouteContext()
			rCtx.URLParams.Add("id", tt.shortID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rCtx))
			w := httptest.NewRecorder()

			tt.handlers.GetURL(w, req)
			res := w.Result()
			err := res.Body.Close()
			require.NoError(t, err)

			assert.Equal(t, tt.want.statusCode, res.StatusCode)
			assert.Equal(t, tt.want.locationHeader, res.Header.Get("Location"))
		})
	}
}

func TestHandlers_APIStoreURL(t *testing.T) {
	type want struct {
		statusCode  int
		response    string
		shortID     string
		contentType string
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
				statusCode:  http.StatusCreated,
				response:    `{"result":"https://example.com/1"}`,
				shortID:     "1",
				contentType: "application/json",
			},
		},
		{
			name:     "should return bad request without provided url",
			handlers: getHandlers([]storage.ShortURL{}),
			longURL:  "",
			want: want{
				statusCode:  http.StatusBadRequest,
				response:    http.StatusText(http.StatusBadRequest),
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:     "should return bad request with not valid url",
			handlers: getHandlers([]storage.ShortURL{}),
			longURL:  "not a url",
			want: want{
				statusCode:  http.StatusBadRequest,
				response:    http.StatusText(http.StatusBadRequest),
				contentType: "text/plain; charset=utf-8",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqData := apiStoreRequest{URL: tt.longURL}
			reqBody, err := json.Marshal(reqData)
			require.NoError(t, err)
			buf := bytes.NewBuffer(reqBody)
			req := httptest.NewRequest(http.MethodPost, "https://example.com", buf)
			w := httptest.NewRecorder()

			tt.handlers.APIStoreURL(w, req)
			res := w.Result()

			resBody, err := ioutil.ReadAll(res.Body)
			require.NoError(t, err)
			err = res.Body.Close()
			require.NoError(t, err)

			assert.Equal(t, tt.want.statusCode, res.StatusCode)
			assert.Equal(t, tt.want.response, strings.TrimRight(string(resBody), "\n"))

			if tt.want.shortID != "" {
				shortURL, err := tt.handlers.Storage.GetByID(context.Background(), tt.want.shortID)
				assert.NoError(t, err)
				assert.Equal(t, tt.longURL, shortURL.LongURL)
			}

			assert.Equal(t, tt.want.contentType, res.Header.Get("Content-Type"))
		})
	}
}

func getHandlers(urls []storage.ShortURL) Handlers {
	s, err := storage.NewMemoryStorage()
	if err != nil {
		panic(err)
	}

	for _, url := range urls {
		_, _ = s.Create(context.Background(), storage.ShortURL{ID: url.ID, LongURL: url.LongURL})
	}

	h := Handlers{
		Storage: s,
		BaseURL: "https://example.com",
	}

	return h
}
