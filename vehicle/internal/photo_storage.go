package vehicle

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/google/uuid"
)

type photoStorage struct {
	dir string
	mu  sync.Mutex
}

var (
	errPhotoCleanup  = errors.New("remove unreferenced vehicle photo")
	errPhotoNotFound = errors.New("vehicle photo not found")
)

// save сначала записывает JPEG и только потом сохраняет ключ в PostgreSQL.
// Уборщик должен учитывать весь этот промежуток: до persist новый файл
// в каталоге уже есть, а среди ключей в базе его ещё нет.
func (s *photoStorage) save(jpeg []byte, persist func(string) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := uuid.NewString() + ".jpg"
	path := filepath.Join(s.dir, key)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("create vehicle photo file: %w", err)
	}
	_, writeErr := file.Write(jpeg)
	if err := errors.Join(writeErr, file.Close()); err != nil {
		return removeFailedPhoto(path, fmt.Errorf("write vehicle photo: %w", err))
	}

	if err := persist(key); err != nil {
		return removeFailedPhoto(path, err)
	}
	return nil
}

func removeFailedPhoto(path string, cause error) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("%w: %w (original failure: %w)", errPhotoCleanup, err, cause)
	}
	return cause
}

func isPhotoKey(key string) bool {
	name, ok := strings.CutSuffix(key, ".jpg")
	if !ok {
		return false
	}
	id, err := uuid.Parse(name)
	return err == nil && id.String() == name
}

func (s *photoStorage) open(key string) (*os.File, int64, error) {
	if !isPhotoKey(key) {
		return nil, 0, errors.New("invalid stored vehicle photo key")
	}
	file, err := os.Open(filepath.Join(s.dir, key))
	if errors.Is(err, os.ErrNotExist) {
		return nil, 0, errPhotoNotFound
	}
	if err != nil {
		return nil, 0, fmt.Errorf("open vehicle photo: %w", err)
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, 0, fmt.Errorf("stat vehicle photo: %w", err)
	}
	if !info.Mode().IsRegular() {
		_ = file.Close()
		return nil, 0, errors.New("stored vehicle photo is not a regular file")
	}
	return file, info.Size(), nil
}
