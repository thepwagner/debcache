package repo

import (
	"context"
	"strings"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
)

// Cached wraps a Repo with an LRU cache.
type Cached struct {
	src Repo

	release *expirable.LRU[string, []byte]
	byHash  *expirable.LRU[string, []byte]
	pool    *expirable.LRU[string, []byte]
}

var _ Repo = (*Cached)(nil)

func NewCached(src Repo) *Cached {
	release := expirable.NewLRU[string, []byte](16, nil, 1*time.Hour)
	byHash := expirable.NewLRU[string, []byte](16, nil, 1*time.Hour)
	pool := expirable.NewLRU[string, []byte](128, nil, 24*time.Hour)

	return &Cached{
		src:     src,
		release: release,
		byHash:  byHash,
		pool:    pool,
	}
}

func (c Cached) InRelease(ctx context.Context, dist string) ([]byte, error) {
	if v, ok := c.release.Get(dist); ok {
		return v, nil
	}

	v, err := c.src.InRelease(ctx, dist)
	if err != nil {
		return nil, err
	}

	c.release.Add(dist, v)
	return v, nil
}

func (c Cached) ByHash(ctx context.Context, dist string, component string, arch string, digest string) ([]byte, error) {
	key := strings.Join([]string{dist, component, arch, digest}, " ")
	if v, ok := c.byHash.Get(key); ok {
		return v, nil
	}

	v, err := c.src.ByHash(ctx, dist, component, arch, digest)
	if err != nil {
		return nil, err
	}

	c.byHash.Add(key, v)
	return v, nil
}

func (c Cached) Pool(ctx context.Context, component string, pkg string, filename string) ([]byte, error) {
	key := strings.Join([]string{component, pkg, filename}, " ")
	if v, ok := c.pool.Get(key); ok {
		return v, nil
	}

	v, err := c.src.Pool(ctx, component, pkg, filename)
	if err != nil {
		return nil, err
	}

	c.pool.Add(key, v)
	return v, nil
}
