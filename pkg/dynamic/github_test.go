package dynamic_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thepwagner/debcache/pkg/dynamic"
)

func TestGitHubReleasesSource(t *testing.T) {
	t.Skip("be nice to github")
	t.Parallel()

	ctx := context.Background()
	gh, err := dynamic.NewGitHubReleasesSource(ctx, dynamic.GitHubReleasesConfig{
		Repositories: map[string]dynamic.GitHubReleasesRepoConfig{
			"sigstore/cosign": {},
		},
	})
	require.NoError(t, err)

	pkgs, ts, err := gh.Packages(ctx)
	require.NoError(t, err)
	assert.False(t, ts.IsZero())
	assert.Len(t, pkgs, 1)
	assert.Len(t, pkgs["main"], 2)
	assert.Len(t, pkgs["main"]["amd64"], 1)
	assert.Len(t, pkgs["main"]["arm64"], 1)

	pkg := pkgs["main"]["amd64"][0]
	assert.Equal(t, "cosign", pkg["Package"])
	assert.NotEqual(t, "", pkg["SHA256"])
}
