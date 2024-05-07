package signature_test

import (
	"context"
	"os"
	"testing"

	"github.com/google/go-github/v61/github"
	"github.com/stretchr/testify/require"
	"github.com/thepwagner/debcache/pkg/signature"
)

func TestGitHubVerifier_Verify(t *testing.T) {
	t.Parallel()
	tok := os.Getenv("TEST_GITHUB_TOKEN")
	if tok == "" {
		t.Skip("no TEST_GITHUB_TOKEN")
	}

	gh := github.NewClient(nil).WithAuthToken(tok)

	verifier, err := signature.NewGitHubVerifier(gh, "thepwagner/ghcr-reaper", signature.FulcioIdentity{
		GitHubWorkflowTrigger: "push",
	})
	require.NoError(t, err)

	b, err := os.ReadFile("/Users/pwagner/tmp/ghcr-reaper_0.0.1_linux_amd64.deb")
	require.NoError(t, err)
	ok, err := verifier.Verify(context.Background(), "0.0.1", b)
	require.NoError(t, err)
	require.True(t, ok)
}
