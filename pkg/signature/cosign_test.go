package signature_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thepwagner/debcache/pkg/signature"
)

func init() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))
}

func TestRekorVerifier(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		dontSkip bool
		identity signature.FulcioIdentity
		asset    string
		version  string
	}{
		// signs the checksum file (which includes .deb digests), not individual debs:
		"anchore/grype": {
			asset:   "https://github.com/anchore/grype/releases/download/v0.73.5/grype_0.73.5_checksums.txt",
			version: "v0.73.5",
			identity: signature.FulcioIdentity{
				GitHubWorkflowRef:     "refs/heads/main",
				GitHubWorkflowTrigger: "workflow_dispatch",

				SourceRepositoryOwnerURI: "https://github.com/anchore",
				BuildConfigURI:           "https://github.com/anchore/grype/.github/workflows/release.yaml@refs/heads/main",
				BuildTrigger:             "workflow_dispatch",
			},
		},

		// signs the .deb directly:
		"getsops/sops": {
			asset:   "https://github.com/getsops/sops/releases/download/v3.8.1/sops_3.8.1_amd64.deb",
			version: "v3.8.1",
			identity: signature.FulcioIdentity{
				GitHubWorkflowRef:        "refs/tags/{{VERSION}}",
				GitHubWorkflowTrigger:    "push",
				SourceRepositoryOwnerURI: "https://github.com/getsops",
				BuildConfigURI:           "https://github.com/getsops/sops/.github/workflows/release.yml@refs/tags/{{VERSION}}",
				BuildTriggerPattern:      `(push|pull_request)`,
			},
		},

		// signs the checksum file (which includes .deb digests), not individual debs:
		"goreleaser/goreleaser": {
			asset:   "https://github.com/goreleaser/goreleaser/releases/download/v1.23.0/checksums.txt",
			version: "v1.23.0",
			identity: signature.FulcioIdentity{
				GitHubWorkflowRef: "refs/tags/{{VERSION}}",
			},
		},

		// signs the checksum file (which includes .deb digests), not individual debs:
		"opentofu/opentofu": {
			asset:   "https://github.com/opentofu/opentofu/releases/download/v1.6.0-rc1/tofu_1.6.0-rc1_SHA256SUMS",
			version: "v1.6.0-rc1",
			identity: signature.FulcioIdentity{
				GitHubWorkflowRef: "refs/heads/v1.6",
				BuildTrigger:      "workflow_dispatch",
			},
		},

		// signs the .deb directly:
		"sigstore/cosign": {
			asset:   "https://github.com/sigstore/cosign/releases/download/v2.2.2/cosign_2.2.2_amd64.deb",
			version: "v2.2.2",
			identity: signature.FulcioIdentity{
				Issuer:         "https://accounts.google.com",
				SubjectAltName: "keyless@projectsigstore.iam.gserviceaccount.com",
			},
		},
	}

	for repo, tc := range cases {
		tc := tc
		t.Run(repo, func(t *testing.T) {
			t.Parallel()
			if !tc.dontSkip {
				t.Skip("skipped - set dontSkip in code to run")
			}

			ctx := context.Background()
			v, err := signature.NewRekorVerifier(ctx, tc.identity)
			require.NoError(t, err)

			resp, err := http.Get(tc.asset)
			require.NoError(t, err)
			defer resp.Body.Close()
			deb, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			ok, err := v.Verify(ctx, tc.version, deb)
			require.NoError(t, err)
			assert.True(t, ok)
		})
	}
}
