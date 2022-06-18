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

	s := storage.NewMemoryStorage()
	h := handlers.Handlers{
		Storage: s,
		BaseURL: baseURL,
	}
	r := handlers.NewRouter(h)

	log.Fatal(http.ListenAndServe(serverAddress, r))
}
