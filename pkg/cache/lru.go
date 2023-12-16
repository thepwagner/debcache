package cache

import (
	"context"
	"sync"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
)

type LRUStorage struct {
	size       int
	defaultTTL time.Duration

	mu   sync.RWMutex
	data map[Namespace]*expirable.LRU[Key, []byte]
}

func NewLRUStorage(nsSize int, ttl time.Duration) *LRUStorage {
	return &LRUStorage{
		size:       nsSize,
		defaultTTL: ttl,
		data:       map[Namespace]*expirable.LRU[Key, []byte]{},
	}
}

var _ Storage = (*LRUStorage)(nil)

func (l *LRUStorage) Get(_ context.Context, key Key) ([]byte, bool) {
	return l.dataMap(key).Get(key)
}

func (l *LRUStorage) Add(_ context.Context, key Key, value []byte) {
	l.dataMap(key).Add(key, value)
}

func (l *LRUStorage) NamespaceTTL(namespace Namespace, ttl time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if ttl == 0 {
		delete(l.data, namespace)
		return
	}
	l.data[namespace] = expirable.NewLRU[Key, []byte](l.size, nil, ttl)
}

func (l *LRUStorage) dataMap(key Key) *expirable.LRU[Key, []byte] {
	ns := key.Namespace()

	l.mu.RLock()
	m, ok := l.data[ns]
	l.mu.RUnlock()
	if ok {
		return m
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	if m, ok = l.data[ns]; !ok {
		m = expirable.NewLRU[Key, []byte](l.size, nil, l.defaultTTL)
		l.data[ns] = m
	}

	return m
}
