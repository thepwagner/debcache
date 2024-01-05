package cache_test

import (
	"testing"

	"github.com/thepwagner/debcache/pkg/cache"
)

func TestFileStorage(t *testing.T) {
	t.Parallel()

	testCache(t, func() cache.Storage {
		return cache.NewFileStorage(cache.FileConfig{Path: t.TempDir()})
	})
}
