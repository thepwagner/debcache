package server_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thepwagner/debcache/pkg/cache"
	"github.com/thepwagner/debcache/pkg/repo"
	"github.com/thepwagner/debcache/pkg/server"
	"gopkg.in/yaml.v3"
)

func TestConfig(t *testing.T) {
	t.Parallel()
	var cfg server.Config
	err := yaml.NewDecoder(strings.NewReader(`---
repos:
  debian:
    type: file-cache
    path: ./tmp/debian
    source:
      type: upstream
      url: http://deb.debian.org/debian
`)).Decode(&cfg)
	require.NoError(t, err)

	assert.Len(t, cfg.Repos, 1)
	require.Contains(t, cfg.Repos, "debian")

	debian, err := server.BuildRepo(context.Background(), "debian", cfg.Repos["debian"])
	require.NoError(t, err)

	fileCache, ok := debian.(*repo.Cache)
	require.True(t, ok)
	store, ok := fileCache.Storage.(*cache.FileStorage)
	require.True(t, ok)
	assert.Equal(t, "./tmp/debian", store.Path)

	upstream, ok := fileCache.Source.(*repo.Upstream)
	require.True(t, ok)
	assert.Equal(t, "", upstream.URL.String())
}
