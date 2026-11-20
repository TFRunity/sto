package vehicle

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestPhotoStorageRemovesFileWhenPersistFails(t *testing.T) {
	dir := t.TempDir()
	store := photoStorage{dir: dir}
	dbErr := errors.New("write key failed")
	err := store.save([]byte("photo contents"), func(key string) error {
		data, err := os.ReadFile(filepath.Join(dir, key))
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != "photo contents" {
			t.Fatalf("unexpected file contents: %q", data)
		}
		return dbErr
	})
	if !errors.Is(err, dbErr) {
		t.Fatalf("save error = %v, want %v", err, dbErr)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("%d unreferenced files in photo directory", len(entries))
	}
}
