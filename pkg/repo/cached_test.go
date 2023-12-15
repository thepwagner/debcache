package repo_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thepwagner/debcache/pkg/repo"
)

func TestCached_InRelease(t *testing.T) {
	u, ctr := countingServer(t, "/dists/test/InRelease")

	cached := repo.NewCached(repo.NewUpstream(u), repo.NewLRUCacheStorage())
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		b, err := cached.InRelease(ctx, "test")
		require.NoError(t, err)
		require.Equal(t, upstreamPayload, b)

		// never increments because of cache
		assert.Equal(t, int64(1), *ctr)
	}
}

func TestCached_ByHash(t *testing.T) {
	u, ctr := countingServer(t, "/dists/test/component/binary-arch/by-hash/SHA256/abc123")

	cached := repo.NewCached(repo.NewUpstream(u), repo.NewLRUCacheStorage())
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		b, err := cached.ByHash(ctx, "test", "component", "arch", "abc123")
		require.NoError(t, err)
		require.Equal(t, upstreamPayload, b)
		assert.Equal(t, int64(1), *ctr)
	}
}

func TestCached_Pool(t *testing.T) {
	u, ctr := countingServer(t, "/pool/component/p/pkg/pkg_1.0_amd64.deb")

	cached := repo.NewCached(repo.NewUpstream(u), repo.NewLRUCacheStorage())
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		b, err := cached.Pool(ctx, "component", "pkg", "pkg_1.0_amd64.deb")
		require.NoError(t, err)
		require.Equal(t, upstreamPayload, b)
		assert.Equal(t, int64(1), *ctr)
	}
}

func countingServer(t testing.TB, path string) (url.URL, *int64) {
	t.Helper()

	var counter int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, path, r.URL.Path)
		atomic.AddInt64(&counter, 1)
		_, _ = w.Write(upstreamPayload)
	}))
	t.Cleanup(srv.Close)
	u, _ := url.Parse(srv.URL)
	return *u, &counter
}
