package vehicle

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
)

func TestCleanPhotosKeepsLiveFiles(t *testing.T) {
	store := photoStorage{dir: t.TempDir()}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	live := uuid.NewString() + ".jpg"
	obsolete := uuid.NewString() + ".jpg"
	for _, key := range []string{live, obsolete, "unrelated.jpg"} {
		if err := os.WriteFile(filepath.Join(store.dir, key), []byte("photo"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	dbErr := errors.New("database unavailable")
	if err := store.clean(t.Context(), func(context.Context) ([]string, error) {
		return nil, dbErr
	}, logger); !errors.Is(err, dbErr) {
		t.Fatalf("cleanup error = %v, want %v", err, dbErr)
	}
	if _, err := os.Stat(filepath.Join(store.dir, obsolete)); err != nil {
		t.Fatalf("database failure should preserve the file: %v", err)
	}

	if err := store.clean(t.Context(), func(context.Context) ([]string, error) {
		return []string{live}, nil
	}, logger); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{live, "unrelated.jpg"} {
		if _, err := os.Stat(filepath.Join(store.dir, key)); err != nil {
			t.Errorf("%s should remain: %v", key, err)
		}
	}
	if _, err := os.Stat(filepath.Join(store.dir, obsolete)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("obsolete file stat error = %v, want not found", err)
	}
}

func TestCleanPhotosDuringUpload(t *testing.T) {
	store := photoStorage{dir: t.TempDir()}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	fileWritten := make(chan string, 1)
	persist := make(chan struct{})
	uploadDone := make(chan error, 1)
	var savedKey string

	go func() {
		uploadDone <- store.save([]byte("photo"), func(key string) error {
			fileWritten <- key
			<-persist
			savedKey = key
			return nil
		})
	}()

	var key string
	select {
	case key = <-fileWritten:
	case err := <-uploadDone:
		t.Fatalf("upload stopped before persistence: %v", err)
	}
	locked := !store.mu.TryLock()
	if !locked {
		store.mu.Unlock()
	}

	cleanupDone := make(chan error, 1)
	go func() {
		cleanupDone <- store.clean(t.Context(), func(context.Context) ([]string, error) {
			if store.mu.TryLock() {
				store.mu.Unlock()
				return nil, errors.New("cleanup read keys without holding the upload lock")
			}
			return []string{savedKey}, nil
		}, logger)
	}()

	close(persist)
	if err := <-uploadDone; err != nil {
		t.Fatal(err)
	}
	if err := <-cleanupDone; err != nil {
		t.Fatal(err)
	}
	if !locked {
		t.Error("upload did not hold the cleanup lock while persisting the key")
	}
	if _, err := os.Stat(filepath.Join(store.dir, key)); err != nil {
		t.Fatalf("uploaded photo should remain: %v", err)
	}
}
