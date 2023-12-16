package cache

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

type FileStorage struct {
	dir string
	ttl time.Duration
}

func NewFileStorage(dir string, ttl time.Duration) *FileStorage {
	return &FileStorage{
		dir: dir,
		ttl: ttl,
	}
}

var _ Storage = (*FileStorage)(nil)

func (f *FileStorage) Get(ctx context.Context, key Key) ([]byte, bool) {
	p := filepath.Join(f.dir, string(key))
	if f.ttl > 0 {
		// Check the file's mtime and ignore if expired:
		stat, err := os.Stat(p)
		if errors.Is(err, fs.ErrNotExist) {
			return nil, false
		} else if err == nil && time.Since(stat.ModTime()) > f.ttl {
			return nil, false
		}
	}

	b, err := os.ReadFile(p)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			slog.Error("cache.FileStorage.get error", slog.String("error", err.Error()))
		}
		return nil, false
	}
	return b, true
}

func (f *FileStorage) Add(ctx context.Context, key Key, value []byte) {
	p := filepath.Join(f.dir, string(key))
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		slog.Error("FileCacheStorage.add mkdir error", slog.String("error", err.Error()))
		return
	}

	if err := os.WriteFile(p, value, 0644); err != nil {
		slog.Error("FileCacheStorage.add write error", slog.String("error", err.Error()))
	}
}
