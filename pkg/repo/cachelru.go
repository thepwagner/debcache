package repo

import (
	"context"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
)

// LRUCacheStorage stores cache to an in-memory LRU. Zoom.
type LRUCacheStorage struct {
	release *expirable.LRU[string, []byte]
	byHash  *expirable.LRU[string, []byte]
	pool    *expirable.LRU[string, []byte]
}

var _ CacheStorage = (*LRUCacheStorage)(nil)

func NewLRUCacheStorage() LRUCacheStorage {
	return LRUCacheStorage{
		release: expirable.NewLRU[string, []byte](16, nil, time.Hour),
		byHash:  expirable.NewLRU[string, []byte](16, nil, time.Hour),
		pool:    expirable.NewLRU[string, []byte](128, nil, 24*time.Hour),
	}
}

func (c LRUCacheStorage) ReleaseGet(_ context.Context, key string) ([]byte, bool) {
	return c.release.Get(key)
}

func (c LRUCacheStorage) ReleaseAdd(_ context.Context, key string, value []byte) {
	c.release.Add(key, value)
}

func (c LRUCacheStorage) ByHashGet(_ context.Context, key string) ([]byte, bool) {
	return c.byHash.Get(key)
}

func (c LRUCacheStorage) ByHashAdd(_ context.Context, key string, value []byte) {
	c.byHash.Add(key, value)
}

func (c LRUCacheStorage) PoolGet(_ context.Context, key string) ([]byte, bool) {
	return c.pool.Get(key)
}

func (c LRUCacheStorage) PoolAdd(_ context.Context, key string, value []byte) {
	c.pool.Add(key, value)
}
