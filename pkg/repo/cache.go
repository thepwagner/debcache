package repo

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5/middleware"
)

// Cache wraps a Repo with an LRU cache.
type Cache struct {
	src     Repo
	storage CacheStorage
}

type CacheConfig struct {
	URL string `yaml:"url"`
}

type CacheStorage interface {
	ReleaseGet(context.Context, string) ([]byte, bool)
	ReleaseAdd(context.Context, string, []byte)

	PackagesGet(context.Context, string) ([]byte, bool)
	PackagesAdd(context.Context, string, []byte)

	ByHashGet(context.Context, string) ([]byte, bool)
	ByHashAdd(context.Context, string, []byte)

	PoolGet(context.Context, string) ([]byte, bool)
	PoolAdd(context.Context, string, []byte)
}

var _ Repo = (*Cache)(nil)

func CacheFromConfig(src Repo, cfg CacheConfig) (*Cache, error) {
	u, err := url.Parse(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("error parsing cache URL: %w", err)
	}

	var store CacheStorage
	switch u.Scheme {
	case "file":
		p := filepath.Join(u.Hostname(), u.Path)
		store = NewFileCache(p)
		slog.Debug("decorating in file cache", slog.String("path", p))
	default:
		return nil, fmt.Errorf("unsupported cache scheme %q", u.Scheme)
	}

	return NewCache(src, store), nil
}

func NewCache(src Repo, storage CacheStorage) *Cache {
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
	if v, ok := c.storage.ReleaseGet(ctx, dist); ok {
		slog.Info("InRelease cache hit", logAttrs...)
		return v, nil
	}
	slog.Info("InRelease cache miss", logAttrs...)

	v, err := c.src.InRelease(ctx, dist)
	if err != nil {
		return nil, err
	}

	c.storage.ReleaseAdd(ctx, dist, v)
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
	key := strings.Join([]string{dist, component, arch, string(compression)}, " ")
	if v, ok := c.storage.PackagesGet(ctx, key); ok {
		slog.Info("Packages cache hit", logAttrs...)
		return v, nil
	}
	slog.Info("Packages cache miss", logAttrs...)

	v, err := c.src.Packages(ctx, dist, component, arch, compression)
	if err != nil {
		return nil, err
	}

	c.storage.PackagesAdd(ctx, key, v)
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
	key := strings.Join([]string{dist, component, arch, digest}, " ")
	if v, ok := c.storage.ByHashGet(ctx, key); ok {
		slog.Info("ByHash cache hit", logAttrs...)
		return v, nil
	}
	slog.Info("ByHash cache miss", logAttrs...)

	v, err := c.src.ByHash(ctx, dist, component, arch, digest)
	if err != nil {
		return nil, err
	}

	c.storage.ByHashAdd(ctx, key, v)
	return v, nil
}

func (c Cache) Pool(ctx context.Context, component string, pkg string, filename string) ([]byte, error) {
	logAttrs := []any{
		slog.String("request_id", middleware.GetReqID(ctx)),
		slog.String("component", component),
		slog.String("pkg", pkg),
		slog.String("filename", filename),
	}
	key := strings.Join([]string{component, pkg, filename}, " ")
	if v, ok := c.storage.PoolGet(ctx, key); ok {
		slog.Info("Pool cache hit", logAttrs...)
		return v, nil
	}
	slog.Info("Pool cache miss", logAttrs...)

	v, err := c.src.Pool(ctx, component, pkg, filename)
	if err != nil {
		return nil, err
	}

	c.storage.PoolAdd(ctx, key, v)
	return v, nil
}
