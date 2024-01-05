package repo

import (
	"context"
	"log/slog"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/thepwagner/debcache/pkg/cache"
)

// Cache wraps a Repo with a cache.
type Cache struct {
	Source  Repo
	Storage cache.Storage
}

var _ Repo = (*Cache)(nil)

const (
	releases     = cache.Namespace("releases")
	packages     = cache.Namespace("packages")
	byHash       = cache.Namespace("by-hash")
	pool         = cache.Namespace("pool")
	translations = cache.Namespace("translations")
)

func NewCache(src Repo, storage cache.Storage) *Cache {
	return &Cache{
		Source:  src,
		Storage: storage,
	}
}

func (c Cache) InRelease(ctx context.Context, dist Distribution) ([]byte, error) {
	key := releases.Key(dist.String())
	v, ok := c.Storage.Get(ctx, key)
	slog.Debug("cached InRelease",
		slog.String("request_id", middleware.GetReqID(ctx)),
		slog.Any("dist", dist),
		slog.Bool("cache_hit", ok),
	)
	if ok {
		return v, nil
	}

	v, err := c.Source.InRelease(ctx, dist)
	if err != nil {
		return nil, err
	}
	c.Storage.Add(ctx, key, v)
	return v, nil
}

func (c Cache) Packages(ctx context.Context, dist Distribution, component Component, arch Architecture, compression Compression) ([]byte, error) {
	key := packages.Key(dist.String(), component.String(), arch.String(), compression.String())
	v, ok := c.Storage.Get(ctx, key)
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

	v, err := c.Source.Packages(ctx, dist, component, arch, compression)
	if err != nil {
		return nil, err
	}
	c.Storage.Add(ctx, key, v)
	return v, nil
}
func (c Cache) Translations(ctx context.Context, dist Distribution, component Component, lang Language, compression Compression) ([]byte, error) {
	key := translations.Key(dist.String(), component.String(), lang.String(), compression.String())
	v, ok := c.Storage.Get(ctx, key)
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

	v, err := c.Source.Translations(ctx, dist, component, lang, compression)
	if err != nil {
		return nil, err
	}
	c.Storage.Add(ctx, key, v)
	return v, nil
}

func (c Cache) ByHash(ctx context.Context, dist Distribution, component Component, arch Architecture, digest string) ([]byte, error) {
	key := byHash.Key(dist.String(), component.String(), arch.String(), digest)
	v, ok := c.Storage.Get(ctx, key)
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

	v, err := c.Source.ByHash(ctx, dist, component, arch, digest)
	if err != nil {
		return nil, err
	}
	c.Storage.Add(ctx, key, v)
	return v, nil
}

func (c Cache) Pool(ctx context.Context, component Component, pkg, filename string) ([]byte, error) {
	key := pool.Key(component.String(), pkg, filename)
	v, ok := c.Storage.Get(ctx, key)
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

	v, err := c.Source.Pool(ctx, component, pkg, filename)
	if err != nil {
		return nil, err
	}
	c.Storage.Add(ctx, key, v)
	return v, nil
}

func (c Cache) SigningKeyPEM() ([]byte, error) {
	return c.Source.SigningKeyPEM()
}
