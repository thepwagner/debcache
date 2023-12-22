package signature_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thepwagner/debcache/pkg/signature"
)

func TestCosignVerifier(t *testing.T) {
	t.Parallel()
	t.Skip("uses network")
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))

	ctx := context.Background()
	v, err := signature.NewRekorVerifier(ctx, signature.FulcioIdentity{
		GitHubWorkflowRef:        "refs/tags/{{VERSION}}",
		GitHubWorkflowTrigger:    "push",
		SourceRepositoryOwnerURI: "https://github.com/getsops",
		BuildConfigURI:           "https://github.com/getsops/sops/.github/workflows/release.yml@refs/tags/{{VERSION}}",
		BuildTriggerPattern:      `(push|pull_request)`,
	})
	require.NoError(t, err)

	deb, err := os.ReadFile("../../tmp/github/github-release-assets:::getsops_sops_sops_3.8.1_amd64.deb")
	require.NoError(t, err)

	ok, err := v.Verify(ctx, "v3.8.1", deb)
	require.NoError(t, err)
	assert.True(t, ok)
}

// test cases:

// https://github.com/sigstore/cosign/releases/tag/v2.2.2 signs the .deb directly
// https://github.com/aquasecurity/trivy/releases/tag/v0.48.1 signs the .deb directly

// https://github.com/anchore/grype/releases/tag/v0.73.4 signs the checksums
// https://github.com/getsops/sops/releases/tag/v3.8.1 signs the checksums file  (which doesn't include the .deb, but that is another problem)
// https://github.com/goreleaser/goreleaser/releases/tag/v1.22.1 signs the checksums file
// https://github.com/opentofu/opentofu/releases/tag/v1.6.0-rc1 signs the checksums file, custom name?

// https://github.com/Shopify/hansel/releases/tag/v0.0.10 does not publish a deb

// https://github.com/Shopify/ejson/releases/tag/v1.4.1 does not sign
