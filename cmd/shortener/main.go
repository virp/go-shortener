package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/virp/go-shortener/internal/app/handlers"
	"github.com/virp/go-shortener/internal/app/storage"
)

const defaultDatabaseQueryTimeout = "3s"

type config struct {
	serverAddress        string
	baseURL              string
	fileStoragePath      string
	databaseDSN          string
	databaseQueryTimeout time.Duration
}

func main() {
	cfg, err := getConfig()
	if err != nil {
		log.Fatal(err)
	}

	var database *sqlx.DB

	if cfg.databaseDSN != "" {
		db, err := sqlx.Open("pgx", cfg.databaseDSN)
		if err != nil {
			log.Fatal(err)
		}
		database = db
	}

	defer func() {
		if database != nil {
			_ = database.Close()
		}
	}()

	s, err := getStorage(cfg, database)
	if err != nil {
		log.Fatal(err)
	}

	h := handlers.Handlers{
		Storage: s,
		BaseURL: cfg.baseURL,
		Secret:  "secretappkey",
		DB:      database,
	}
	r := handlers.NewRouter(h)

	log.Fatal(http.ListenAndServe(cfg.serverAddress, r))
}

func getStorage(cfg config, db *sqlx.DB) (storage.URLStorage, error) {
	if cfg.databaseDSN != "" {
		if err := checkDBTables(db, cfg.databaseQueryTimeout); err != nil {
			return nil, fmt.Errorf("check db tables: %w", err)
		}
		return storage.NewPostgresStorage(db, cfg.databaseQueryTimeout)
	}
	if cfg.fileStoragePath != "" {
		return storage.NewFileStorage(cfg.fileStoragePath)
	}
	return storage.NewMemoryStorage()
}

func getConfig() (config, error) {
	dqt, err := time.ParseDuration(defaultDatabaseQueryTimeout)
	if err != nil {
		return config{}, fmt.Errorf("parse database query time duration: %w", err)
	}

	// Default config
	cfg := config{
		serverAddress:        ":8080",
		baseURL:              "http://localhost:8080",
		fileStoragePath:      "",
		databaseDSN:          "",
		databaseQueryTimeout: dqt,
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
	flag.StringVar(&cfg.serverAddress, "a", cfg.serverAddress, "Server Address")
	flag.StringVar(&cfg.baseURL, "b", cfg.baseURL, "Base URL")
	flag.StringVar(&cfg.fileStoragePath, "f", cfg.fileStoragePath, "File Storage Path")
	flag.StringVar(&cfg.databaseDSN, "d", cfg.databaseDSN, "Database DSN")
	flag.Parse()

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
	if dsn, ok := os.LookupEnv("DATABASE_DSN"); ok {
		cfg.databaseDSN = dsn
	}

	return cfg
}

const createUrlsTableQuery = `create table if not exists urls
(
    id             serial primary key,
    url            text not null unique,
    user_id        uuid default null,
    correlation_id text default null
)`

func checkDBTables(db *sqlx.DB, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	_, err := db.ExecContext(
		ctx,
		createUrlsTableQuery,
	)
	if err != nil {
		return fmt.Errorf("creating urls table: %w", err)
	}

	return nil
}
