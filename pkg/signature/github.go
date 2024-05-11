package signature

import (
	"context"
	"crypto/sha256"
	"crypto/x509"
	"fmt"
	"log/slog"

	"github.com/google/go-github/v62/github"
	"github.com/sigstore/sigstore-go/pkg/bundle"
	"github.com/sigstore/sigstore-go/pkg/root"
	"github.com/sigstore/sigstore-go/pkg/tuf"
	"github.com/sigstore/sigstore-go/pkg/verify"
)

type GitHubVerifier struct {
	gh       *github.Client
	id       *CertificateVerifier
	verifier *verify.SignedEntityVerifier
	nwo      string
}

func NewGitHubVerifier(gh *github.Client, repo string, identity FulcioIdentity) (*GitHubVerifier, error) {
	verifier, err := newPublicGoodVerifier()
	if err != nil {
		return nil, err
	}
	id, err := NewCertificateVerifier(identity)
	if err != nil {
		return nil, err
	}
	return &GitHubVerifier{
		gh:       gh,
		id:       id,
		verifier: verifier,
		nwo:      repo,
	}, nil
}

func (gh *GitHubVerifier) Verify(ctx context.Context, version string, deb []byte) (bool, error) {
	hash := sha256.Sum256(deb)
	digest := hash[:]

	attestations, err := gh.getAttestations(ctx, digest)
	if err != nil {
		return false, fmt.Errorf("failed to get attestations: %w", err)
	}

	pol := verify.NewPolicy(verify.WithArtifactDigest("sha256", digest), verify.WithoutIdentitiesUnsafe())

	for _, a := range attestations {
		cert, err := x509.ParseCertificate(a.Bundle.GetVerificationMaterial().GetCertificate().GetRawBytes())
		if err != nil {
			return false, fmt.Errorf("parsing public key: %w", err)
		}
		if ok, err := gh.id.Verify(version, cert); err != nil {
			return false, fmt.Errorf("verifying extensions: %w", err)
		} else if !ok {
			continue
		}

		if _, err := gh.verifier.Verify(a.Bundle, pol); err != nil {
			slog.Info("failed to verify", slog.Any("error", err))
			continue
		}
		return true, nil
	}

	return false, nil
}

func (gh *GitHubVerifier) getAttestations(ctx context.Context, digest []byte) ([]Attestation, error) {
	// Is fetching attestations from the organization ever required?
	req, err := gh.gh.NewRequest("GET", fmt.Sprintf("repos/%s/attestations/sha256:%x", gh.nwo, digest), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	var attestation AttestationReply
	resp, err := gh.gh.Do(ctx, req, &attestation)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	return attestation.Attestations, nil
}

type Attestation struct {
	Bundle *bundle.ProtobufBundle `json:"bundle"`
}

type AttestationReply struct {
	Attestations []Attestation `json:"attestations"`
}

func newPublicGoodVerifier() (*verify.SignedEntityVerifier, error) {
	client, err := tuf.New(tuf.DefaultOptions())
	if err != nil {
		return nil, fmt.Errorf("failed to create TUF client: %w", err)
	}
	trustedRoot, err := root.GetTrustedRoot(client)
	if err != nil {
		return nil, fmt.Errorf("failed to get trusted root: %w", err)
	}

	sv, err := verify.NewSignedEntityVerifier(trustedRoot, verify.WithSignedCertificateTimestamps(1), verify.WithTransparencyLog(1), verify.WithObserverTimestamps(1))
	if err != nil {
		return nil, fmt.Errorf("failed to create Public Good verifier: %w", err)
	}

	return sv, nil
}
