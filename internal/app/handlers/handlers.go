package handlers

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/virp/go-shortener/internal/app/storage"
)

type Handlers struct {
	Storage  storage.URLStorage
	BaseHost string
}

func NewRouter(h Handlers) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/" {
			h.StoreURL(w, r)
			return
		}

		if r.Method == http.MethodGet {
			_, err := strconv.Atoi(r.URL.Path[1:])
			if err != nil {
				http.NotFound(w, r)
				return
			}
			h.GetURL(w, r)
			return
		}

		http.NotFound(w, r)
	})

	return mux
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
	shortID := strings.TrimPrefix(r.URL.Path, "/")
	shortURL, err := h.Storage.GetById(shortID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Location", shortURL.LongURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}
