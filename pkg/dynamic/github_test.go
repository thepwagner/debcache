package dynamic_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thepwagner/debcache/pkg/dynamic"
	"github.com/thepwagner/debcache/pkg/repo"
	"github.com/thepwagner/debcache/pkg/signature"
)

func TestGitHubReleasesSource(t *testing.T) {
	t.Parallel()
	t.Skip("be nice to github")

	cases := map[string]struct {
		config dynamic.GitHubReleasesRepoConfig
	}{
		"no verification": {},
		"deb verification": {
			config: dynamic.GitHubReleasesRepoConfig{
				Signer: &signature.FulcioIdentity{
					Issuer:         "https://accounts.google.com",
					SubjectAltName: "keyless@projectsigstore.iam.gserviceaccount.com",
				},
			},
		},
		"checksum verification": {
			config: dynamic.GitHubReleasesRepoConfig{
				ChecksumFile: "cosign_checksums.txt",
				Signer: &signature.FulcioIdentity{
					Issuer:         "https://accounts.google.com",
					SubjectAltName: "keyless@projectsigstore.iam.gserviceaccount.com",
				},
			},
		},
	}

	for label, tc := range cases {
		tc := tc
		t.Run(label, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			gh, err := dynamic.NewGitHubReleasesSource(ctx, dynamic.GitHubReleasesConfig{
				Architectures: []repo.Architecture{"amd64", "arm64"},
				Repositories: map[string]dynamic.GitHubReleasesRepoConfig{
					"sigstore/cosign": tc.config,
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
		})
	}
}
