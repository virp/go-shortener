package handlers

import (
	"compress/flate"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jmoiron/sqlx"
	"github.com/virp/go-shortener/internal/app/storage"
)

type Handlers struct {
	Storage storage.URLStorage
	BaseURL string
	Secret  string
	DB      *sqlx.DB
}

type apiStoreRequest struct {
	URL string `json:"url"`
}

type apiStoreResponse struct {
	Result string `json:"result"`
}

type apiStoreBatchRequest struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

type apiStoreBatchResponse struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
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
	r.Post("/api/shorten/batch", h.APIStoreURLBatch)
	r.Get("/api/user/urls", h.APIGetUserURLs)
	r.Delete("/api/user/urls", h.APIDeleteUserURLs)

	r.Get("/ping", h.CheckDB)

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
	statusCode := http.StatusCreated
	shortURL, err = h.Storage.Create(r.Context(), shortURL)
	if err != nil {
		if errors.Is(err, storage.ErrAlreadyExist) {
			statusCode = http.StatusConflict
		} else {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}

	generatedShortURL := fmt.Sprintf("%s/%s", h.BaseURL, shortURL.ID)

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(statusCode)
	_, _ = w.Write([]byte(generatedShortURL))
}

func (h Handlers) GetURL(w http.ResponseWriter, r *http.Request) {
	shortID := chi.URLParam(r, "id")
	shortURL, err := h.Storage.GetByID(r.Context(), shortID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if shortURL.IsDeleted {
		w.WriteHeader(http.StatusGone)
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
	statusCode := http.StatusCreated
	shortURL, err = h.Storage.Create(r.Context(), shortURL)
	if err != nil {
		if errors.Is(err, storage.ErrAlreadyExist) {
			statusCode = http.StatusConflict
		} else {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
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
	w.WriteHeader(statusCode)
	_, _ = w.Write(resBody)
}

func (h Handlers) APIStoreURLBatch(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	defer func() { _ = r.Body.Close() }()

	var reqData []apiStoreBatchRequest
	err = json.Unmarshal(body, &reqData)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	userID := getUserIDFromRequest(r)

	var urls []storage.ShortURL
	for _, rd := range reqData {
		u, err := url.ParseRequestURI(rd.OriginalURL)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		urlShort := storage.ShortURL{
			LongURL:       u.String(),
			CorrelationID: rd.CorrelationID,
			UserID:        userID,
		}
		urls = append(urls, urlShort)
	}

	urls, err = h.Storage.CreateBatch(r.Context(), urls)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	var resData []apiStoreBatchResponse
	for _, urlShort := range urls {
		rd := apiStoreBatchResponse{
			CorrelationID: urlShort.CorrelationID,
			ShortURL:      fmt.Sprintf("%s/%s", h.BaseURL, urlShort.ID),
		}
		resData = append(resData, rd)
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

	urls := h.Storage.FindByUserID(r.Context(), userID)
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

func (h Handlers) APIDeleteUserURLs(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	defer func() { _ = r.Body.Close() }()

	userID := getUserIDFromRequest(r)
	var ids []string
	err = json.Unmarshal(body, &ids)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	go func() {
		_ = h.Storage.DeleteBatch(r.Context(), userID, ids)
	}()

	w.WriteHeader(http.StatusAccepted)
}

func (h Handlers) CheckDB(w http.ResponseWriter, r *http.Request) {
	if err := h.DB.PingContext(r.Context()); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func getUserIDFromRequest(r *http.Request) string {
	var userID string
	if ctxValue := r.Context().Value(userKey); ctxValue != nil {
		userID = ctxValue.(string)
	}

	return userID
}
