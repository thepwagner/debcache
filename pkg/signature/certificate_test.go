package signature_test

import (
	"crypto/x509"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thepwagner/debcache/pkg/signature"
)

func TestCertificateVerifier_Verify(t *testing.T) {
	t.Parallel()

	const testCert = `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`
	certID := signature.FulcioIdentity{
		Issuer:                              "https://token.actions.githubusercontent.com",
		SubjectAltName:                      "https://github.com/thepwagner-org/actions/.github/workflows/golang-release-attest.yaml@refs/heads/github-attestations",
		GitHubWorkflowTrigger:               "push",
		GitHubWorkflowSha:                   "2c82d327043cd45d9d5ac935159a368679402de2",
		GitHubWorkflowName:                  "Release",
		GitHubWorkflowRepository:            "thepwagner/ghcr-reaper",
		GitHubWorkflowRef:                   "refs/tags/{{VERSION}}",
		BuildSignerURI:                      "https://github.com/thepwagner-org/actions/.github/workflows/golang-release-attest.yaml@refs/heads/github-attestations",
		BuildSignerDigest:                   "997382cc2748637d733269d54b93a0d50ce1f0da",
		RunnerEnvironment:                   "github-hosted",
		SourceRepositoryURI:                 "https://github.com/thepwagner/ghcr-reaper",
		SourceRepositoryDigest:              "2c82d327043cd45d9d5ac935159a368679402de2",
		SourceRepositoryRef:                 "refs/tags/{{VERSION}}",
		SourceRepositoryIdentifier:          "514619908",
		SourceRepositoryOwnerURI:            "https://github.com/thepwagner",
		SourceRepositoryOwnerIdentifier:     "1559510",
		BuildConfigURI:                      "https://github.com/thepwagner/ghcr-reaper/.github/workflows/release.yaml@refs/tags/{{VERSION}}",
		BuildConfigDigest:                   "2c82d327043cd45d9d5ac935159a368679402de2",
		BuildTrigger:                        "push",
		RunInvocationURI:                    "https://github.com/thepwagner/ghcr-reaper/actions/runs/8946281267/attempts/1",
		SourceRepositoryVisibilityAtSigning: "public",
	}

	b, err := base64.StdEncoding.DecodeString(testCert)
	require.NoError(t, err)
	cert, err := x509.ParseCertificate(b)
	require.NoError(t, err)

	cases := map[string]struct {
		mutator func(*signature.FulcioIdentity)
		ok      bool
	}{
		"all match":                       {ok: true},
		"issuer mismatch":                 {mutator: func(id *signature.FulcioIdentity) { id.Issuer = "meow" }},
		"san mismatch":                    {mutator: func(id *signature.FulcioIdentity) { id.SubjectAltName = "meow" }},
		"trigger mismatch":                {mutator: func(id *signature.FulcioIdentity) { id.GitHubWorkflowTrigger = "meow" }},
		"sha mismatch":                    {mutator: func(id *signature.FulcioIdentity) { id.GitHubWorkflowSha = "meow" }},
		"name mismatch":                   {mutator: func(id *signature.FulcioIdentity) { id.GitHubWorkflowName = "meow" }},
		"repo mismatch":                   {mutator: func(id *signature.FulcioIdentity) { id.GitHubWorkflowRepository = "meow" }},
		"ref mismatch":                    {mutator: func(id *signature.FulcioIdentity) { id.GitHubWorkflowRef = "meow" }},
		"signer uri mismatch":             {mutator: func(id *signature.FulcioIdentity) { id.BuildSignerURI = "meow" }},
		"signer digest mismatch":          {mutator: func(id *signature.FulcioIdentity) { id.BuildSignerDigest = "meow" }},
		"runner env mismatch":             {mutator: func(id *signature.FulcioIdentity) { id.RunnerEnvironment = "meow" }},
		"source repo uri mismatch":        {mutator: func(id *signature.FulcioIdentity) { id.SourceRepositoryURI = "meow" }},
		"source repo digest mismatch":     {mutator: func(id *signature.FulcioIdentity) { id.SourceRepositoryDigest = "meow" }},
		"source repo ref mismatch":        {mutator: func(id *signature.FulcioIdentity) { id.SourceRepositoryRef = "meow" }},
		"source repo id mismatch":         {mutator: func(id *signature.FulcioIdentity) { id.SourceRepositoryIdentifier = "meow" }},
		"source repo owner uri mismatch":  {mutator: func(id *signature.FulcioIdentity) { id.SourceRepositoryOwnerURI = "meow" }},
		"source repo owner id mismatch":   {mutator: func(id *signature.FulcioIdentity) { id.SourceRepositoryOwnerIdentifier = "meow" }},
		"build config uri mismatch":       {mutator: func(id *signature.FulcioIdentity) { id.BuildConfigURI = "meow" }},
		"build config digest mismatch":    {mutator: func(id *signature.FulcioIdentity) { id.BuildConfigDigest = "meow" }},
		"build trigger mismatch":          {mutator: func(id *signature.FulcioIdentity) { id.BuildTrigger = "meow" }},
		"run invocation uri mismatch":     {mutator: func(id *signature.FulcioIdentity) { id.RunInvocationURI = "meow" }},
		"source repo visibility mismatch": {mutator: func(id *signature.FulcioIdentity) { id.SourceRepositoryVisibilityAtSigning = "meow" }},

		"issuer pattern match":    {ok: true, mutator: func(id *signature.FulcioIdentity) { id.IssuerPattern = ".*" }},
		"issuer pattern mismatch": {mutator: func(id *signature.FulcioIdentity) { id.IssuerPattern = "ftp://meow.com" }},
	}
	for label, tc := range cases {
		t.Run(label, func(t *testing.T) {
			t.Parallel()

			tCert := certID
			if tc.mutator != nil {
				tc.mutator(&tCert)
			}
			verifier, err := signature.NewCertificateVerifier(tCert)
			require.NoError(t, err)

			ok, err := verifier.Verify("v0.0.1", cert)
			require.NoError(t, err)
			assert.Equal(t, tc.ok, ok)
		})
	}
}
