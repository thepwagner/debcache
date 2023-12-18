package cache_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/thepwagner/debcache/pkg/cache"
)

func testCache(t *testing.T, storage func() cache.Storage) {
	t.Helper()

	value := []byte("testValue")
	ctx := context.Background()

	t.Run("key not found", func(t *testing.T) {
		t.Parallel()
		_, ok := storage().Get(ctx, cache.Key("keyNotFound"))
		assert.False(t, ok)
	})

	t.Run("store", func(t *testing.T) {
		t.Parallel()
		storage().Add(ctx, cache.Key("keyWrittenNeverRead"), value)
	})

	t.Run("key found", func(t *testing.T) {
		t.Parallel()

		key := cache.Namespace("foo").Key("bar")
		stor := storage()
		stor.Add(ctx, key, value)
		storedValue, ok := stor.Get(ctx, key)
		assert.True(t, ok)
		assert.Equal(t, value, storedValue)
	})

	t.Run("namespace expiry", func(t *testing.T) {
		t.Parallel()
		stor := storage()

		fastNS := cache.Namespace("fast")
		slowNS := cache.Namespace("slow")
		stor.NamespaceTTL(fastNS, 10*time.Millisecond)
		stor.NamespaceTTL(slowNS, time.Minute)

		stor.Add(ctx, fastNS.Key("foo"), value)
		stor.Add(ctx, slowNS.Key("foo"), value)

		time.Sleep(50 * time.Millisecond)

		_, ok := stor.Get(ctx, fastNS.Key("foo"))
		assert.False(t, ok)
		_, ok = stor.Get(ctx, slowNS.Key("foo"))
		assert.True(t, ok)
	})
}
