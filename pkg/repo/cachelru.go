package repo

import (
	"context"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
)

// LRUCache stores cache to an in-memory LRU. Zoom.
type LRUCache struct {
	release  *expirable.LRU[string, []byte]
	packages *expirable.LRU[string, []byte]
	byHash   *expirable.LRU[string, []byte]
	pool     *expirable.LRU[string, []byte]
}

var _ CacheStorage = (*LRUCache)(nil)

func NewLRUCache() LRUCache {
	return LRUCache{
		release:  expirable.NewLRU[string, []byte](16, nil, time.Hour),
		packages: expirable.NewLRU[string, []byte](16, nil, time.Hour),
		byHash:   expirable.NewLRU[string, []byte](16, nil, 24*time.Hour),
		pool:     expirable.NewLRU[string, []byte](128, nil, 24*time.Hour),
	}
}

func (c LRUCache) ReleaseGet(_ context.Context, key string) ([]byte, bool) {
	return c.release.Get(key)
}

func (c LRUCache) ReleaseAdd(_ context.Context, key string, value []byte) {
	c.release.Add(key, value)
}

func (c LRUCache) PackagesGet(_ context.Context, key string) ([]byte, bool) {
	return c.packages.Get(key)
}

func (c LRUCache) PackagesAdd(_ context.Context, key string, value []byte) {
	c.packages.Add(key, value)
}

func (c LRUCache) ByHashGet(_ context.Context, key string) ([]byte, bool) {
	return c.byHash.Get(key)
}

func (c LRUCache) ByHashAdd(_ context.Context, key string, value []byte) {
	c.byHash.Add(key, value)
}

func (c LRUCache) PoolGet(_ context.Context, key string) ([]byte, bool) {
	return c.pool.Get(key)
}

func (c LRUCache) PoolAdd(_ context.Context, key string, value []byte) {
	c.pool.Add(key, value)
}
