package storage

import (
	"bufio"
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFile_Create(t *testing.T) {
	filename, err := getTmpFilename()
	require.NoError(t, err)
	defer func() {
		err := removeTmpFile(filename)
		require.NoError(t, err)
	}()

	s, err := NewFileStorage(filename)
	require.NoError(t, err)

	longURL := "https://example.com/very/long/url/for/shortener"
	url, err := s.Create(context.Background(), ShortURL{LongURL: longURL})
	require.NoError(t, err)
	assert.Equal(t, "1", url.ID)

	s, err = NewFileStorage(filename)
	require.NoError(t, err)
	url, err = s.GetByID(context.Background(), "1")
	require.NoError(t, err)
	assert.Equal(t, longURL, url.LongURL)
}

func TestFile_GetByID(t *testing.T) {
	filename, err := getTmpFilename()
	require.NoError(t, err)
	defer func() {
		err := removeTmpFile(filename)
		require.NoError(t, err)
	}()

	f, err := os.OpenFile(filename, os.O_WRONLY, 0777)
	require.NoError(t, err)
	w := bufio.NewWriter(f)
	_, err = w.WriteString(`{"ID":"custom","LongURL":"https://example.com/very/long/url/for/shortener"}`)
	require.NoError(t, err)
	err = w.Flush()
	require.NoError(t, err)
	err = f.Close()
	require.NoError(t, err)

	s, err := NewFileStorage(filename)
	require.NoError(t, err)

	url, err := s.GetByID(context.Background(), "custom")
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/very/long/url/for/shortener", url.LongURL)
}

func getTmpFilename() (string, error) {
	f, err := os.CreateTemp("/tmp", "file_storage_test_")
	if err != nil {
		return "", err
	}
	defer func() {
		_ = f.Close()
	}()

	filename := f.Name()

	return filename, nil
}

func removeTmpFile(filename string) error {
	return os.Remove(filename)
}
