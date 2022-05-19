package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
)

var urls map[int]string

func main() {
	urls = make(map[int]string)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			if r.URL.Path != "/" {
				http.NotFound(w, r)
				return
			}
			defer func() {
				_ = r.Body.Close()
			}()
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}
			id := addURL(string(body))
			shortURL := fmt.Sprintf("http://localhost:8080/%s", id)
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(shortURL))
		case http.MethodGet:
			idParam := strings.TrimPrefix(r.URL.Path, "/")
			id, err := strconv.Atoi(idParam)
			if err != nil {
				http.Error(w, "Wrong url ID", http.StatusBadRequest)
				return
			}
			long, err := getURL(id)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Location", long)
			w.WriteHeader(http.StatusTemporaryRedirect)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	})

	if err := http.ListenAndServe("0.0.0.0:8080", nil); err != nil {
		log.Fatalln(err)
	}
}

func addURL(long string) string {
	nextID := len(urls) + 1
	urls[nextID] = long
	return strconv.Itoa(nextID)
}

func getURL(id int) (string, error) {
	long, ok := urls[id]
	if !ok {
		return "", errors.New("not found")
	}
	return long, nil
}
