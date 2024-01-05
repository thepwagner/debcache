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
	Path  string
	ttl   time.Duration
	nsTTL map[Namespace]time.Duration
}

type FileConfig struct {
	Path string        `yaml:"path"`
	TTL  time.Duration `yaml:"ttl"`
}

func NewFileStorage(cfg FileConfig) *FileStorage {
	var ttl time.Duration
	if cfg.TTL == 0 {
		ttl = time.Hour
	} else {
		ttl = cfg.TTL
	}
	return &FileStorage{
		Path:  cfg.Path,
		ttl:   ttl,
		nsTTL: map[Namespace]time.Duration{},
	}
}

var _ Storage = (*FileStorage)(nil)

func (f *FileStorage) Get(_ context.Context, key Key) ([]byte, bool) {
	p := filepath.Join(f.Path, string(key))

	ttl, ok := f.nsTTL[key.Namespace()]
	if !ok {
		ttl = f.ttl
	}
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
			slog.Error("cache.FileStorage.get error", slog.String("error", err.Error()))
		}
		return nil, false
	}
	return b, true
}

func (f *FileStorage) Add(_ context.Context, key Key, value []byte) {
	p := filepath.Join(f.Path, string(key))
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		slog.Error("FileCacheStorage.add mkdir error", slog.String("error", err.Error()))
		return
	}

	if err := os.WriteFile(p, value, 0644); err != nil {
		slog.Error("FileCacheStorage.add write error", slog.String("error", err.Error()))
	}
}

func (f *FileStorage) NamespaceTTL(namepace Namespace, ttl time.Duration) {
	f.nsTTL[namepace] = ttl
}
