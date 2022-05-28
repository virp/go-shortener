package handlers

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
	"github.com/virp/go-shortener/internal/app/storage"
)

type Handlers struct {
	Storage  storage.URLStorage
	BaseHost string
}

func NewRouter(h Handlers) *chi.Mux {
	r := chi.NewRouter()
	r.Post("/", h.StoreURL)
	r.Get("/{id}", h.GetURL)

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

	generatedShortURL := fmt.Sprintf("%s/%s", h.BaseHost, shortURL.ID)

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
