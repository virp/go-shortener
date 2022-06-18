package main

import (
	"log"
	"net/http"
	"os"

	"github.com/virp/go-shortener/internal/app/handlers"
	"github.com/virp/go-shortener/internal/app/storage"
)

func main() {
	serverAddress, ok := os.LookupEnv("SERVER_ADDRESS")
	if !ok {
		serverAddress = ":8080"
	}

	baseURL, ok := os.LookupEnv("BASE_URL")
	if !ok {
		baseURL = "http://localhost:8080"
	}

	s, err := getStorage()
	if err != nil {
		log.Fatal(err)
	}

	h := handlers.Handlers{
		Storage: s,
		BaseURL: baseURL,
	}
	r := handlers.NewRouter(h)

	log.Fatal(http.ListenAndServe(serverAddress, r))
}

func getStorage() (storage.URLStorage, error) {
	fileStoragePath, ok := os.LookupEnv("FILE_STORAGE_PATH")
	if ok {
		return storage.NewFileStorage(fileStoragePath)
	} else {
		return storage.NewMemoryStorage()
	}
}
