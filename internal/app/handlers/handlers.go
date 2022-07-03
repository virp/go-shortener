package handlers

import (
	"compress/flate"
	"encoding/json"
	"fmt"
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
	Secret  string
}

type apiStoreRequest struct {
	URL string `json:"url"`
}

type apiStoreResponse struct {
	Result string `json:"result"`
}

type apiUserURL struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

func NewRouter(h Handlers) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Compress(flate.BestCompression, "text/plain", "application/json"))
	r.Use(DecompressRequest)
	r.Use(IdentifyUser(h.Secret))

	r.Post("/", h.StoreURL)
	r.Get("/{id}", h.GetURL)

	r.Post("/api/shorten", h.APIStoreURL)
	r.Get("/api/user/urls", h.APIGetUserURLs)

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

	userID := getUserIDFromRequest(r)

	shortURL := storage.ShortURL{
		LongURL: u.String(),
		UserID:  userID,
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

	userID := getUserIDFromRequest(r)

	shortURL := storage.ShortURL{
		LongURL: u.String(),
		UserID:  userID,
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

func (h Handlers) APIGetUserURLs(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromRequest(r)
	if userID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	urls := h.Storage.FindByUserID(userID)
	if len(urls) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	response := make([]apiUserURL, len(urls))
	for i, shortURL := range urls {
		apiURL := apiUserURL{
			ShortURL:    fmt.Sprintf("%s/%s", h.BaseURL, shortURL.ID),
			OriginalURL: shortURL.LongURL,
		}
		response[i] = apiURL
	}

	resBody, err := json.Marshal(response)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(resBody)
}

func getUserIDFromRequest(r *http.Request) string {
	var userID string
	if ctxValue := r.Context().Value(userKey); ctxValue != nil {
		userID = ctxValue.(string)
	}

	return userID
}
