package main

import (
	"log"
	"net/http"

	"github.com/virp/go-shortener/internal/app/handlers"
	"github.com/virp/go-shortener/internal/app/storage"
)

func main() {
	s := storage.NewMemoryStorage()
	h := handlers.Handlers{
		Storage:  s,
		BaseHost: "http://localhost:8080",
	}
	r := handlers.NewRouter(h)

	log.Fatal(http.ListenAndServe(":8080", r))
}
