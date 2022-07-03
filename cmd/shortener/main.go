package main

import (
	"errors"
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/virp/go-shortener/internal/app/handlers"
	"github.com/virp/go-shortener/internal/app/storage"
)

type config struct {
	serverAddress        string
	baseURL              string
	fileStoragePath      string
	isFileStoragePathSet bool
}

func main() {
	cfg, err := getConfig()
	if err != nil {
		log.Fatal(err)
	}

	s, err := getStorage(cfg)
	if err != nil {
		log.Fatal(err)
	}

	h := handlers.Handlers{
		Storage: s,
		BaseURL: cfg.baseURL,
		Secret:  "secretappkey",
	}
	r := handlers.NewRouter(h)

	log.Fatal(http.ListenAndServe(cfg.serverAddress, r))
}

func getStorage(cfg config) (storage.URLStorage, error) {
	if cfg.fileStoragePath != "" {
		return storage.NewFileStorage(cfg.fileStoragePath)
	} else {
		return storage.NewMemoryStorage()
	}
}

func getConfig() (config, error) {
	// Default config
	cfg := config{
		serverAddress:   ":8080",
		baseURL:         "http://localhost:8080",
		fileStoragePath: "",
	}

	// Override config by flags
	cfg = getFlagConfig(cfg)

	// Override config by env
	cfg = getEnvConfig(cfg)

	// Check required config fields
	if cfg.serverAddress == "" {
		return config{}, errors.New("config: server address not configured")
	}
	if cfg.baseURL == "" {
		return config{}, errors.New("config: base URL not configured")
	}

	return cfg, nil
}

func getFlagConfig(cfg config) config {
	sa := flag.String("a", cfg.serverAddress, "Server Address")
	bu := flag.String("b", cfg.baseURL, "Base URL")
	fsp := flag.String("f", cfg.fileStoragePath, "File Storage Path")
	flag.Parse()

	cfg.serverAddress = *sa
	cfg.baseURL = *bu
	cfg.fileStoragePath = *fsp

	return cfg
}

func getEnvConfig(cfg config) config {
	if sa, ok := os.LookupEnv("SERVER_ADDRESS"); ok {
		cfg.serverAddress = sa
	}
	if bu, ok := os.LookupEnv("BASE_URL"); ok {
		cfg.baseURL = bu
	}
	if fsp, ok := os.LookupEnv("FILE_STORAGE_PATH"); ok {
		cfg.fileStoragePath = fsp
	}

	return cfg
}
