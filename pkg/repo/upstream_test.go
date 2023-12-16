package repo_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thepwagner/debcache/pkg/repo"
)

var upstreamPayload = []byte("meow")

func TestUpstream_InRelease(t *testing.T) {
	u := repo.NewUpstream(assertingServer(t, "/dists/test/InRelease"))

	res, err := u.InRelease(context.Background(), "test")
	require.NoError(t, err)
	require.Equal(t, upstreamPayload, res)
}

func TestUpstream_ByHash(t *testing.T) {
	u := repo.NewUpstream(assertingServer(t, "/dists/test/component/binary-arch/by-hash/SHA256/abc123"))

	res, err := u.ByHash(context.Background(), "test", "component", "arch", "abc123")
	require.NoError(t, err)
	require.Equal(t, upstreamPayload, res)
}

func TestUpstream_Pool(t *testing.T) {
	u := repo.NewUpstream(assertingServer(t, "/pool/component/p/pkg/pkg_1.0_amd64.deb"))

	res, err := u.Pool(context.Background(), "component", "pkg", "pkg_1.0_amd64.deb")
	require.NoError(t, err)
	require.Equal(t, upstreamPayload, res)
}

func assertingServer(tb testing.TB, path string) url.URL {
	tb.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(tb, path, r.URL.Path)
		_, _ = w.Write(upstreamPayload)
	}))
	tb.Cleanup(srv.Close)
	u, _ := url.Parse(srv.URL)
	return *u
}
