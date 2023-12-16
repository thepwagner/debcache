package cache_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thepwagner/debcache/pkg/cache"
)

func testCache(t *testing.T, storage cache.Storage) {
	t.Helper()

	value := []byte("testValue")
	ctx := context.Background()

	t.Run("key not found", func(t *testing.T) {
		t.Parallel()
		_, ok := storage.Get(ctx, cache.Key("keyNotFound"))
		assert.False(t, ok)
	})

	t.Run("store", func(t *testing.T) {
		t.Parallel()
		storage.Add(ctx, cache.Key("keyWrittenNeverRead"), value)
	})

	t.Run("key found", func(t *testing.T) {
		t.Parallel()

		key := cache.Namespace("foo").Key("bar")
		storage.Add(ctx, key, value)
		storedValue, ok := storage.Get(ctx, key)
		assert.True(t, ok)
		assert.Equal(t, value, storedValue)
	})
}
