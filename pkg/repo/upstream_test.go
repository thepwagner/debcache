package repo_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thepwagner/debcache/pkg/repo"
)

func TestUpstream_InRelease(t *testing.T) {
	t.Parallel()
	srv := countingServer(t, "/dists/test/InRelease")
	u := repo.NewUpstream(srv)

	res, err := u.InRelease(context.Background(), "test")
	require.NoError(t, err)
	require.Equal(t, []byte("1"), res)
}

func TestUpstream_Packages(t *testing.T) {
	t.Parallel()
	srv := countingServer(t, "/dists/test/component/binary-arch/Packages")
	u := repo.NewUpstream(srv)

	res, err := u.Packages(context.Background(), "test", "component", "arch", repo.CompressionNone)
	require.NoError(t, err)
	require.Equal(t, []byte("1"), res)
}

func TestUpstream_Translations(t *testing.T) {
	t.Parallel()
	srv := countingServer(t, "/dists/test/component/i18n/Translation-de.bz2")
	u := repo.NewUpstream(srv)

	res, err := u.Translations(context.Background(), "test", "component", "de", repo.CompressionBZIP)
	require.NoError(t, err)
	require.Equal(t, []byte("1"), res) // das ist gut
}

func TestUpstream_ByHash(t *testing.T) {
	t.Parallel()
	srv := countingServer(t, "/dists/test/component/binary-arch/by-hash/SHA256/abc123")
	u := repo.NewUpstream(srv)

	res, err := u.ByHash(context.Background(), "test", "component", "arch", "abc123")
	require.NoError(t, err)
	require.Equal(t, []byte("1"), res)
}

func TestUpstream_Pool(t *testing.T) {
	t.Parallel()
	srv := countingServer(t, "/pool/component/p/pkg/pkg_1.0_amd64.deb")
	u := repo.NewUpstream(srv)

	res, err := u.Pool(context.Background(), "component/p/pkg/pkg_1.0_amd64.deb")
	require.NoError(t, err)
	require.Equal(t, []byte("1"), res)
}

func countingServer(tb testing.TB, path string) url.URL {
	tb.Helper()

	var counter int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(tb, path, r.URL.Path)
		v := atomic.AddInt64(&counter, 1)
		_, _ = fmt.Fprintf(w, "%d", v)
	}))
	tb.Cleanup(srv.Close)
	u, _ := url.Parse(srv.URL)
	return *u
}
