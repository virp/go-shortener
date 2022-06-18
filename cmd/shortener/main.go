package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/virp/go-shortener/internal/app/handlers"
	"github.com/virp/go-shortener/internal/app/storage"
)

var (
	serverAddress   *string
	baseURL         *string
	fileStoragePath *string
)

func init() {
	sa, ok := os.LookupEnv("SERVER_ADDRESS")
	if !ok {
		sa = ":8080"
	}
	serverAddress = flag.String("a", sa, "Server Address")

	bu, ok := os.LookupEnv("BASE_URL")
	if !ok {
		bu = "http://localhost:8080"
	}
	baseURL = flag.String("b", bu, "Base URL")

	fileStoragePath = flag.String("f", os.Getenv("FILE_STORAGE_PATH"), "File Storage Path")
}

func main() {
	flag.Parse()

	s, err := getStorage()
	if err != nil {
		log.Fatal(err)
	}

	h := handlers.Handlers{
		Storage: s,
		BaseURL: *baseURL,
	}
	r := handlers.NewRouter(h)

	log.Fatal(http.ListenAndServe(*serverAddress, r))
}

func getStorage() (storage.URLStorage, error) {
	if *fileStoragePath != "" {
		return storage.NewFileStorage(*fileStoragePath)
	} else {
		return storage.NewMemoryStorage()
	}
}
