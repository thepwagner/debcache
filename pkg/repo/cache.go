package repo

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/thepwagner/debcache/pkg/cache"
)

// Cache wraps a Repo with a cache.
type Cache struct {
	src     Repo
	storage cache.Storage
}

var _ Repo = (*Cache)(nil)

const (
	releases     = cache.Namespace("releases")
	packages     = cache.Namespace("packages")
	byHash       = cache.Namespace("by-hash")
	pool         = cache.Namespace("pool")
	translations = cache.Namespace("translations")
)

func CacheFromConfig(src Repo, cfg cache.Config) (*Cache, error) {
	store, err := cache.StorageFromConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("preparing cache storage: %w", err)
	}

	// TODO: this assumes reads are clustered
	// What if clients are pulling for multiple architectures? We don't want the InReleases file to expire and reference a new amd64/Packages, while the arm64/Pacakges is still cached.
	// We need to flush `releases`, `packages` and `byHash` at the same time.
	store.NamespaceTTL(releases, 4*time.Hour)
	store.NamespaceTTL(packages, 4*time.Hour)
	store.NamespaceTTL(byHash, 4*time.Hour)

	return NewCache(src, store), nil
}

func NewCache(src Repo, storage cache.Storage) *Cache {
	return &Cache{
		src:     src,
		storage: storage,
	}
}

func (c Cache) InRelease(ctx context.Context, dist Distribution) ([]byte, error) {
	key := releases.Key(dist.String())
	v, ok := c.storage.Get(ctx, key)
	slog.Debug("cached InRelease",
		slog.String("request_id", middleware.GetReqID(ctx)),
		slog.Any("dist", dist),
		slog.Bool("cache_hit", ok),
	)
	if ok {
		return v, nil
	}

	v, err := c.src.InRelease(ctx, dist)
	if err != nil {
		return nil, err
	}
	c.storage.Add(ctx, key, v)
	return v, nil
}

func (c Cache) Packages(ctx context.Context, dist Distribution, component Component, arch Architecture, compression Compression) ([]byte, error) {
	key := packages.Key(dist.String(), component.String(), arch.String(), compression.String())
	v, ok := c.storage.Get(ctx, key)
	slog.Debug("cached Packages",
		slog.String("request_id", middleware.GetReqID(ctx)),
		slog.Any("dist", dist),
		slog.Any("component", component),
		slog.Any("arch", arch),
		slog.String("compression", string(compression)),
		slog.Bool("cache_hit", ok),
	)
	if ok {
		return v, nil
	}

	v, err := c.src.Packages(ctx, dist, component, arch, compression)
	if err != nil {
		return nil, err
	}
	c.storage.Add(ctx, key, v)
	return v, nil
}
func (c Cache) Translations(ctx context.Context, dist Distribution, component Component, lang Language, compression Compression) ([]byte, error) {
	key := translations.Key(dist.String(), component.String(), lang.String(), compression.String())
	v, ok := c.storage.Get(ctx, key)
	slog.Debug("cached Translations",
		slog.String("request_id", middleware.GetReqID(ctx)),
		slog.Any("dist", dist),
		slog.Any("component", component),
		slog.Any("lang", lang),
		slog.String("compression", string(compression)),
		slog.Bool("cache_hit", ok),
	)
	if ok {
		return v, nil
	}

	v, err := c.src.Translations(ctx, dist, component, lang, compression)
	if err != nil {
		return nil, err
	}
	c.storage.Add(ctx, key, v)
	return v, nil
}

func (c Cache) ByHash(ctx context.Context, dist Distribution, component Component, arch Architecture, digest string) ([]byte, error) {
	key := byHash.Key(dist.String(), component.String(), arch.String(), digest)
	v, ok := c.storage.Get(ctx, key)
	slog.Debug("cached ByHash",
		slog.String("request_id", middleware.GetReqID(ctx)),
		slog.Any("dist", dist),
		slog.Any("component", component),
		slog.Any("arch", arch),
		slog.String("digest", digest),
		slog.Bool("cache_hit", ok),
	)
	if ok {
		return v, nil
	}

	v, err := c.src.ByHash(ctx, dist, component, arch, digest)
	if err != nil {
		return nil, err
	}
	c.storage.Add(ctx, key, v)
	return v, nil
}

func (c Cache) Pool(ctx context.Context, component Component, pkg, filename string) ([]byte, error) {
	key := pool.Key(component.String(), pkg, filename)
	v, ok := c.storage.Get(ctx, key)
	slog.Debug("cached Pool",
		slog.String("request_id", middleware.GetReqID(ctx)),
		slog.Any("component", component),
		slog.String("pkg", pkg),
		slog.String("filename", filename),
		slog.Bool("cache_hit", ok),
	)
	if ok {
		return v, nil
	}

	v, err := c.src.Pool(ctx, component, pkg, filename)
	if err != nil {
		return nil, err
	}
	c.storage.Add(ctx, key, v)
	return v, nil
}

func (c Cache) SigningKeyPEM() ([]byte, error) {
	return c.src.SigningKeyPEM()
}
