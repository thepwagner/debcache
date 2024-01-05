package cache_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/thepwagner/debcache/pkg/cache"
)

func TestLRUStorage(t *testing.T) {
	t.Parallel()
	testCache(t, func() cache.Storage {
		return cache.NewLRUStorage(cache.LRUConfig{TTL: time.Minute})
	})
}

func TestLRUStorage_Eviction(t *testing.T) {
	t.Parallel()

	lru := cache.NewLRUStorage(cache.LRUConfig{Size: 5})
	ctx := context.Background()
	value := []byte("testValue")

	for i := 0; i < 10; i++ {
		lru.Add(ctx, cache.Key(fmt.Sprintf("%d", i)), value)
	}

	for i := 0; i < 5; i++ {
		v, ok := lru.Get(ctx, cache.Key(fmt.Sprintf("%d", i)))
		assert.False(t, ok)
		assert.Nil(t, v)
	}
	for i := 5; i < 10; i++ {
		v, ok := lru.Get(ctx, cache.Key(fmt.Sprintf("%d", i)))
		assert.True(t, ok)
		assert.Equal(t, value, v)
	}
}

func TestLRUStorage_Expiry(t *testing.T) {
	t.Parallel()

	lru := cache.NewLRUStorage(cache.LRUConfig{TTL: 10 * time.Millisecond})
	ctx := context.Background()
	key := cache.Key("testKey")
	value := []byte("testValue")

	lru.Add(ctx, key, value)
	storedValue, ok := lru.Get(ctx, key)
	assert.True(t, ok)
	assert.Equal(t, value, storedValue)

	time.Sleep(20 * time.Millisecond) // wait for the value to expire
	storedValue, ok = lru.Get(ctx, key)
	assert.False(t, ok)
	assert.Nil(t, storedValue)
}
