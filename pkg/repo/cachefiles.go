package repo

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

// FileCacheStorage stores cache to the filesystem.
type FileCacheStorage struct {
	dir string
}

var _ CacheStorage = (*FileCacheStorage)(nil)

func NewFileCacheStorage(dir string) FileCacheStorage {
	return FileCacheStorage{dir: dir}
}

func (c FileCacheStorage) ReleaseGet(_ context.Context, key string) ([]byte, bool) {
	return c.get("release", key, time.Hour)
}

func (c FileCacheStorage) ReleaseAdd(_ context.Context, key string, value []byte) {
	c.add("release", key, value)
}

func (c FileCacheStorage) ByHashGet(_ context.Context, key string) ([]byte, bool) {
	return c.get("byhash", key, 0)
}

func (c FileCacheStorage) ByHashAdd(_ context.Context, key string, value []byte) {
	c.add("byhash", key, value)
}

func (c FileCacheStorage) PoolGet(_ context.Context, key string) ([]byte, bool) {
	return c.get("pool", key, 0)
}

func (c FileCacheStorage) PoolAdd(_ context.Context, key string, value []byte) {
	c.add("pool", key, value)
}

func (c FileCacheStorage) get(subDir, key string, ttl time.Duration) ([]byte, bool) {
	p := filepath.Join(c.dir, subDir, key)
	if ttl > 0 {
		// Check the file's mtime and ignore if expired:
		stat, err := os.Stat(p)
		if errors.Is(err, fs.ErrNotExist) {
			return nil, false
		} else if err == nil && time.Since(stat.ModTime()) > ttl {
			return nil, false
		}
	}

	b, err := os.ReadFile(p)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			slog.Error("FileCacheStorage.get error", slog.String("error", err.Error()))
		}
		return nil, false
	}
	return b, true
}

func (c FileCacheStorage) add(subDir, key string, value []byte) {
	p := filepath.Join(c.dir, subDir, key)
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		slog.Error("FileCacheStorage.add mkdir error", slog.String("error", err.Error()))
		return
	}

	if err := os.WriteFile(p, value, 0644); err != nil {
		slog.Error("FileCacheStorage.add write error", slog.String("error", err.Error()))
	}
}
