package signature

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"log/slog"

	"github.com/go-openapi/runtime"
	"github.com/sigstore/cosign/v2/pkg/cosign"
	"github.com/sigstore/fulcio/pkg/certificate"
	rekor "github.com/sigstore/rekor/pkg/client"
	"github.com/sigstore/rekor/pkg/generated/client"
	"github.com/sigstore/rekor/pkg/generated/client/entries"
	"github.com/sigstore/rekor/pkg/generated/client/index"
	"github.com/sigstore/rekor/pkg/generated/models"
)

type CosignConfig struct {
	CertificateOIDCIssuer     string `yaml:"certificate-oidc-issuer"`
	CertificateIdentityRegExp string `yaml:"certificate-identity-regexp"`
}

// RekorVerifier verifies a deb using the Rekor transparency log.
type RekorVerifier struct {
	client *client.Rekor
	pubs   *cosign.TrustedTransparencyLogPubKeys
}

func NewCosignVerifier(ctx context.Context) (*RekorVerifier, error) {
	client, err := rekor.GetRekorClient("https://rekor.sigstore.dev/")
	if err != nil {
		return nil, err
	}

	pubs, err := cosign.GetRekorPubs(ctx)
	if err != nil {
		return nil, err
	}

	return &RekorVerifier{
		client: client,
		pubs:   pubs,
	}, nil
}

func (v *RekorVerifier) Verify(ctx context.Context, deb []byte) (bool, error) {
	digest := fmt.Sprintf("sha256:%x", sha256.Sum256(deb))
	log := slog.With(slog.String("digest", digest))

	entries, err := v.findEntry(ctx, digest)
	if err != nil {
		return false, fmt.Errorf("searching index: %w", err)
	} else if len(entries) == 0 {
		log.Debug("no entry found")
		return false, nil
	}
	log.Debug("entry found", slog.Any("entries", entries))

	for _, entry := range entries {
		ce, err := v.verifyEntry(ctx, entry)
		if err != nil {
			return false, fmt.Errorf("verifying entry: %w", err)
		} else if ce == nil {
			continue
		}
		log.Debug("entry verified", slog.String("workflow", ce.Issuer))
	}

	return false, fmt.Errorf("not implemented")
}

func (v RekorVerifier) findEntry(ctx context.Context, digest string) ([]string, error) {
	res, err := v.client.Index.SearchIndex(&index.SearchIndexParams{
		Context: ctx,
		Query: &models.SearchIndex{
			Hash: digest,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("searching index: %w", err)
	}

	return res.Payload, nil
}

func (v RekorVerifier) verifyEntry(ctx context.Context, entryUUID string) (*certificate.Extensions, error) {
	entryRes, err := v.client.Entries.GetLogEntryByUUID(&entries.GetLogEntryByUUIDParams{
		Context:   ctx,
		EntryUUID: entryUUID,
	})
	if err != nil {
		return nil, fmt.Errorf("fetching index: %w", err)
	}
	for _, payload := range entryRes.GetPayload() {
		if err := cosign.VerifyTLogEntryOffline(ctx, &payload, v.pubs); err != nil {
			return nil, fmt.Errorf("verifying entry: %w", err)
		}
		dec, err := base64.StdEncoding.DecodeString(payload.Body.(string))
		if err != nil {
			return nil, fmt.Errorf("decoding entry: %w", err)
		}
		pe, err := models.UnmarshalProposedEntry(bytes.NewReader(dec), runtime.JSONConsumer())
		if err != nil {
			return nil, fmt.Errorf("unmarshaling proposed entry: %w", err)
		}

		switch entry := pe.(type) {
		case *models.Intoto:
			d := entry.Spec.(map[string]interface{})
			pubKeyRaw, err := base64.StdEncoding.DecodeString(d["publicKey"].(string))
			if err != nil {
				return nil, fmt.Errorf("decoding public key: %w", err)
			}
			block, rest := pem.Decode(pubKeyRaw)
			if len(rest) > 0 {
				return nil, fmt.Errorf("extra data after PEM block")
			}
			cert, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("parsing public key: %w", err)
			}
			ext := certificate.Extensions{}
			for _, e := range cert.Extensions {
				switch {
				case e.Id.Equal(certificate.OIDIssuerV2):
					if err := certificate.ParseDERString(e.Value, &ext.Issuer); err != nil {
						return nil, fmt.Errorf("parsing issuer: %w", err)
					}
					// TODO: rest of https://github.com/znewman01/fulcio/blob/main/pkg/certificate/extensions.go
				}
			}

			return &ext, nil
		default:
			return nil, fmt.Errorf("unsupported entry type: %T", pe)
		}
	}
	return nil, nil
}
