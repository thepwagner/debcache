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

// FileCache stores cache to the filesystem.
type FileCache struct {
	dir string
}

var _ CacheStorage = (*FileCache)(nil)

func NewFileCache(dir string) FileCache {
	return FileCache{dir: dir}
}

func (c FileCache) ReleaseGet(_ context.Context, key string) ([]byte, bool) {
	return c.get("release", key, time.Hour)
}

func (c FileCache) ReleaseAdd(_ context.Context, key string, value []byte) {
	c.add("release", key, value)
}

func (c FileCache) PackagesGet(_ context.Context, key string) ([]byte, bool) {
	return c.get("packages", key, time.Hour)
}

func (c FileCache) PackagesAdd(_ context.Context, key string, value []byte) {
	c.add("packages", key, value)
}

func (c FileCache) ByHashGet(_ context.Context, key string) ([]byte, bool) {
	return c.get("byhash", key, 0)
}

func (c FileCache) ByHashAdd(_ context.Context, key string, value []byte) {
	c.add("byhash", key, value)
}

func (c FileCache) PoolGet(_ context.Context, key string) ([]byte, bool) {
	return c.get("pool", key, 0)
}

func (c FileCache) PoolAdd(_ context.Context, key string, value []byte) {
	c.add("pool", key, value)
}

func (c FileCache) get(subDir, key string, ttl time.Duration) ([]byte, bool) {
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

func (c FileCache) add(subDir, key string, value []byte) {
	p := filepath.Join(c.dir, subDir, key)
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		slog.Error("FileCacheStorage.add mkdir error", slog.String("error", err.Error()))
		return
	}

	if err := os.WriteFile(p, value, 0644); err != nil {
		slog.Error("FileCacheStorage.add write error", slog.String("error", err.Error()))
	}
}
