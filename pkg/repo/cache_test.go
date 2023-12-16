package repo_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/thepwagner/debcache/pkg/repo"
)

func TestCached_InRelease(t *testing.T) {
	t.Parallel()
	srv := countingServer(t, "/dists/test/InRelease")
	cached := repo.NewCache(repo.NewUpstream(srv), repo.NewLRUCache())

	ctx := context.Background()
	for i := 0; i < 3; i++ {
		b, err := cached.InRelease(ctx, "test")
		require.NoError(t, err)
		require.Equal(t, []byte("1"), b)
	}
}

func TestCached_Packages(t *testing.T) {
	t.Parallel()
	srv := countingServer(t, "/dists/test/component/binary-arch/Packages")
	cached := repo.NewCache(repo.NewUpstream(srv), repo.NewLRUCache())

	ctx := context.Background()
	for i := 0; i < 3; i++ {
		b, err := cached.Packages(ctx, "test", "component", "arch", repo.CompressionNone)
		require.NoError(t, err)
		require.Equal(t, []byte("1"), b)
	}
}

func TestCached_ByHash(t *testing.T) {
	t.Parallel()
	srv := countingServer(t, "/dists/test/component/binary-arch/by-hash/SHA256/abc123")
	cached := repo.NewCache(repo.NewUpstream(srv), repo.NewLRUCache())

	ctx := context.Background()
	for i := 0; i < 3; i++ {
		b, err := cached.ByHash(ctx, "test", "component", "arch", "abc123")
		require.NoError(t, err)
		require.Equal(t, []byte("1"), b)
	}
}

func TestCached_Pool(t *testing.T) {
	t.Parallel()
	srv := countingServer(t, "/pool/component/p/pkg/pkg_1.0_amd64.deb")
	cached := repo.NewCache(repo.NewUpstream(srv), repo.NewLRUCache())

	ctx := context.Background()
	for i := 0; i < 3; i++ {
		b, err := cached.Pool(ctx, "component", "pkg", "pkg_1.0_amd64.deb")
		require.NoError(t, err)
		require.Equal(t, []byte("1"), b)
	}
}
