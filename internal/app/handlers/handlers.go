package handlers

import (
	"compress/flate"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/virp/go-shortener/internal/app/storage"
)

type Handlers struct {
	Storage storage.URLStorage
	BaseURL string
}

type apiStoreRequest struct {
	URL string `json:"url"`
}

type apiStoreResponse struct {
	Result string `json:"result"`
}

func NewRouter(h Handlers) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Compress(flate.BestCompression, "text/plain", "application/json"))
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Content-Encoding") != "gzip" {
				next.ServeHTTP(w, r)
				return
			}

			gz, err := gzip.NewReader(r.Body)
			if err != nil {
				_, _ = io.WriteString(w, err.Error())
				return
			}
			r.Body = gz
			next.ServeHTTP(w, r)
		})
	})

	r.Post("/", h.StoreURL)
	r.Get("/{id}", h.GetURL)

	r.Post("/api/shorten", h.APIStoreURL)

	return r
}

func (h Handlers) StoreURL(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	defer func() {
		_ = r.Body.Close()
	}()

	u, err := url.ParseRequestURI(string(body))
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	shortURL := storage.ShortURL{
		LongURL: u.String(),
	}
	shortURL, err = h.Storage.Create(shortURL)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	generatedShortURL := fmt.Sprintf("%s/%s", h.BaseURL, shortURL.ID)

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write([]byte(generatedShortURL))
}

func (h Handlers) GetURL(w http.ResponseWriter, r *http.Request) {
	shortID := chi.URLParam(r, "id")
	shortURL, err := h.Storage.GetByID(shortID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Location", shortURL.LongURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func (h Handlers) APIStoreURL(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	defer func() {
		_ = r.Body.Close()
	}()

	var reqData apiStoreRequest
	err = json.Unmarshal(body, &reqData)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	u, err := url.ParseRequestURI(reqData.URL)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	shortURL := storage.ShortURL{
		LongURL: u.String(),
	}
	shortURL, err = h.Storage.Create(shortURL)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	generatedShortURL := fmt.Sprintf("%s/%s", h.BaseURL, shortURL.ID)

	resData := apiStoreResponse{
		Result: generatedShortURL,
	}

	resBody, err := json.Marshal(resData)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write(resBody)
}
