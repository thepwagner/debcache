package cache_test

import (
	"testing"
	"time"

	"github.com/thepwagner/debcache/pkg/cache"
)

func TestFileStorage(t *testing.T) {
	t.Parallel()

	testCache(t, func() cache.Storage {
		return cache.NewFileStorage(t.TempDir(), time.Minute)
	})
}
