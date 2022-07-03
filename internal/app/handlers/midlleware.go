package handlers

import (
	"compress/gzip"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
)

func DecompressRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Encoding") != "gzip" {
			next.ServeHTTP(w, r)
			return
		}

		gz, err := gzip.NewReader(r.Body)
		if err != nil {
			_, _ = io.WriteString(w, err.Error())
			return
		}
		r.Body = gz
		next.ServeHTTP(w, r)
	})
}

type userCtxKey int

const userKey userCtxKey = 1

func IdentifyUser(secret string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c, err := r.Cookie("user")
			if err != nil && !errors.Is(err, http.ErrNoCookie) {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}

			aesGCM, err := getAES(secret)
			if err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}

			if c != nil {
				if user, err := getUser(c.Value, aesGCM); err == nil {
					ctx := context.WithValue(r.Context(), userKey, user)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}

			nonce, err := generateRandom(aesGCM.NonceSize())
			if err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}

			user := uuid.NewString()
			encUser := aesGCM.Seal(nil, nonce, []byte(user), nil)

			msg := append(encUser, nonce...)

			c = &http.Cookie{
				Name:  "user",
				Value: hex.EncodeToString(msg),
			}
			http.SetCookie(w, c)

			ctx := context.WithValue(r.Context(), userKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func generateRandom(size int) ([]byte, error) {
	b := make([]byte, size)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func getAES(secret string) (cipher.AEAD, error) {
	key := sha256.Sum256([]byte(secret))
	aesBlock, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, fmt.Errorf("aesblock: %w", err)
	}
	aesGCM, err := cipher.NewGCM(aesBlock)
	if err != nil {
		return nil, fmt.Errorf("aesgcm: %w", err)
	}

	return aesGCM, nil
}

func getUser(cookieValue string, aesGCM cipher.AEAD) (string, error) {
	encUser, err := hex.DecodeString(cookieValue)
	if err != nil {
		return "", fmt.Errorf("decode cookie value: %w", err)
	}
	nonce := encUser[len(encUser)-aesGCM.NonceSize():]
	decUser, err := aesGCM.Open(nil, nonce, encUser[:len(encUser)-aesGCM.NonceSize()], nil)
	if err != nil {
		return "", fmt.Errorf("decrypt user: %w", err)
	}

	return string(decUser), nil
}
