package repo

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/thepwagner/debcache/pkg/cache"
)

// Cache wraps a Repo with an LRU cache.
type Cache struct {
	src     Repo
	storage cache.Storage
}

type CacheConfig struct {
	URL string `yaml:"url"`
}

var _ Repo = (*Cache)(nil)

const (
	releases = cache.Namespace("releases")
	packages = cache.Namespace("packages")
	byHash   = cache.Namespace("by-hash")
	pool     = cache.Namespace("pool")
)

func CacheFromConfig(src Repo, cfg CacheConfig) (*Cache, error) {
	u, err := url.Parse(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("error parsing cache URL: %w", err)
	}

	var store cache.Storage
	switch u.Scheme {
	case "file":
		p := filepath.Join(u.Hostname(), u.Path)
		store = cache.NewFileStorage(p, time.Hour)
		slog.Debug("decorating in file cache", slog.String("path", p))
	default:
		return nil, fmt.Errorf("unsupported cache scheme %q", u.Scheme)
	}

	return NewCache(src, store), nil
}

func NewCache(src Repo, storage cache.Storage) *Cache {
	return &Cache{
		src:     src,
		storage: storage,
	}
}

func (c Cache) InRelease(ctx context.Context, dist string) ([]byte, error) {
	logAttrs := []any{
		slog.String("request_id", middleware.GetReqID(ctx)),
		slog.String("dist", dist),
	}
	key := releases.Key(dist)
	if v, ok := c.storage.Get(ctx, key); ok {
		slog.Info("InRelease cache hit", logAttrs...)
		return v, nil
	}
	slog.Info("InRelease cache miss", logAttrs...)

	v, err := c.src.InRelease(ctx, dist)
	if err != nil {
		return nil, err
	}

	c.storage.Add(ctx, key, v)
	return v, nil
}

func (c Cache) Packages(ctx context.Context, dist, component, arch string, compression Compression) ([]byte, error) {
	logAttrs := []any{
		slog.String("request_id", middleware.GetReqID(ctx)),
		slog.String("dist", dist),
		slog.String("component", component),
		slog.String("arch", arch),
		slog.String("compression", string(compression)),
	}
	key := packages.Key(strings.Join([]string{dist, component, arch, string(compression)}, " "))
	if v, ok := c.storage.Get(ctx, key); ok {
		slog.Info("Packages cache hit", logAttrs...)
		return v, nil
	}
	slog.Info("Packages cache miss", logAttrs...)

	v, err := c.src.Packages(ctx, dist, component, arch, compression)
	if err != nil {
		return nil, err
	}

	c.storage.Add(ctx, key, v)
	return v, nil
}

func (c Cache) ByHash(ctx context.Context, dist string, component string, arch string, digest string) ([]byte, error) {
	logAttrs := []any{
		slog.String("request_id", middleware.GetReqID(ctx)),
		slog.String("dist", dist),
		slog.String("component", component),
		slog.String("arch", arch),
		slog.String("digest", digest),
	}
	key := byHash.Key(strings.Join([]string{dist, component, arch, digest}, " "))
	if v, ok := c.storage.Get(ctx, key); ok {
		slog.Info("ByHash cache hit", logAttrs...)
		return v, nil
	}
	slog.Info("ByHash cache miss", logAttrs...)

	v, err := c.src.ByHash(ctx, dist, component, arch, digest)
	if err != nil {
		return nil, err
	}

	c.storage.Add(ctx, key, v)
	return v, nil
}

func (c Cache) Pool(ctx context.Context, component string, pkg string, filename string) ([]byte, error) {
	logAttrs := []any{
		slog.String("request_id", middleware.GetReqID(ctx)),
		slog.String("component", component),
		slog.String("pkg", pkg),
		slog.String("filename", filename),
	}
	key := pool.Key(strings.Join([]string{component, pkg, filename}, " "))
	if v, ok := c.storage.Get(ctx, key); ok {
		slog.Info("Pool cache hit", logAttrs...)
		return v, nil
	}
	slog.Info("Pool cache miss", logAttrs...)

	v, err := c.src.Pool(ctx, component, pkg, filename)
	if err != nil {
		return nil, err
	}

	c.storage.Add(ctx, key, v)
	return v, nil
}
