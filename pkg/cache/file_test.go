package cache_test

import (
	"testing"
	"time"

	"github.com/thepwagner/debcache/pkg/cache"
)

func TestFileStorage(t *testing.T) {
	t.Parallel()

	lru := cache.NewFileStorage(t.TempDir(), time.Minute)
	testCache(t, lru)
}
