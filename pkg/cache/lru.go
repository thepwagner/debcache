package cache

import (
	"context"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
)

type LRUStorage struct {
	data *expirable.LRU[Key, []byte]
}

func NewLRUStorage(size int, ttl time.Duration) *LRUStorage {
	return &LRUStorage{
		data: expirable.NewLRU[Key, []byte](size, nil, ttl),
	}
}

var _ Storage = (*LRUStorage)(nil)

func (l *LRUStorage) Get(ctx context.Context, key Key) ([]byte, bool) {
	return l.data.Get(key)
}

func (l *LRUStorage) Add(ctx context.Context, key Key, value []byte) {
	l.data.Add(key, value)
}
