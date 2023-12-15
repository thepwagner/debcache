package repo

import (
	"context"
	"log/slog"
	"strings"

	"github.com/go-chi/chi/v5/middleware"
)

// Cached wraps a Repo with an LRU cache.
type Cached struct {
	src     Repo
	storage CacheStorage
}

type CacheStorage interface {
	ReleaseGet(context.Context, string) ([]byte, bool)
	ReleaseAdd(context.Context, string, []byte)

	ByHashGet(context.Context, string) ([]byte, bool)
	ByHashAdd(context.Context, string, []byte)

	PoolGet(context.Context, string) ([]byte, bool)
	PoolAdd(context.Context, string, []byte)
}

var _ Repo = (*Cached)(nil)

func NewCached(src Repo, storage CacheStorage) *Cached {
	return &Cached{
		src:     src,
		storage: storage,
	}
}

func (c Cached) InRelease(ctx context.Context, dist string) ([]byte, error) {
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

func (c Cached) ByHash(ctx context.Context, dist string, component string, arch string, digest string) ([]byte, error) {
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

func (c Cached) Pool(ctx context.Context, component string, pkg string, filename string) ([]byte, error) {
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
