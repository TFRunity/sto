package vehicle

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

const photoCleanupInterval = 30 * 24 * time.Hour

// RunPhotoCleanup runs at startup and every 30 days while Vehicle is running.
func (h *Handler) RunPhotoCleanup(ctx context.Context, logger *slog.Logger) {
	ticker := time.NewTicker(photoCleanupInterval)
	defer ticker.Stop()
	for ctx.Err() == nil {
		if err := h.photos.clean(ctx, h.repo.queries.ListLivePhotoKeys, logger); err != nil && ctx.Err() == nil {
			logger.WarnContext(ctx, "photo cleanup incomplete", "err", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *photoStorage) clean(ctx context.Context, liveKeys func(context.Context) ([]string, error), logger *slog.Logger) error {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return fmt.Errorf("list photo files: %w", err)
	}

	// Take the database snapshot after listing the directory. Hold the upload
	// lock through deletion so a file cannot still be waiting for its key.
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return err
	}
	keys, err := liveKeys(ctx)
	if err != nil {
		return fmt.Errorf("list live photo keys: %w", err)
	}
	live := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		live[key] = struct{}{}
	}

	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return err
		}
		key := entry.Name()
		if _, keep := live[key]; keep || !entry.Type().IsRegular() || !isPhotoKey(key) {
			continue
		}
		if err := os.Remove(filepath.Join(s.dir, key)); err != nil && !errors.Is(err, os.ErrNotExist) {
			logger.WarnContext(ctx, "could not remove unused photo", "key", key, "err", err)
		}
	}
	return nil
}
