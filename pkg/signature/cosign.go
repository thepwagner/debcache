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
	rekor "github.com/sigstore/rekor/pkg/client"
	"github.com/sigstore/rekor/pkg/generated/client"
	"github.com/sigstore/rekor/pkg/generated/client/entries"
	"github.com/sigstore/rekor/pkg/generated/client/index"
	"github.com/sigstore/rekor/pkg/generated/models"
)

// RekorVerifier verifies a deb using the Rekor transparency log.
type RekorVerifier struct {
	client *client.Rekor
	pubs   *cosign.TrustedTransparencyLogPubKeys
	id     *CertificateVerifier
}

var _ Verifier = (*RekorVerifier)(nil)

func NewRekorVerifier(ctx context.Context, identity FulcioIdentity) (*RekorVerifier, error) {
	client, err := rekor.GetRekorClient("https://rekor.sigstore.dev/")
	if err != nil {
		return nil, err
	}
	pubs, err := cosign.GetRekorPubs(ctx)
	if err != nil {
		return nil, err
	}
	id, err := NewCertificateVerifier(identity)
	if err != nil {
		return nil, err
	}
	return &RekorVerifier{
		client: client,
		pubs:   pubs,
		id:     id,
	}, nil
}

func (v *RekorVerifier) Verify(ctx context.Context, version string, deb []byte) (bool, error) {
	digest := fmt.Sprintf("sha256:%x", sha256.Sum256(deb))
	log := slog.With(slog.String("digest", digest))
	entries, err := v.findEntry(ctx, digest)
	if err != nil {
		return false, fmt.Errorf("searching index: %w", err)
	} else if len(entries) == 0 {
		log.Debug("no entry found")
		return false, nil
	}
	log.Debug("entries found", slog.Any("entries", entries))

	for _, entry := range entries {
		if ok, err := v.verifyEntry(ctx, version, entry); err != nil {
			return false, fmt.Errorf("verifying entry: %w", err)
		} else if ok {
			log.Debug("entry verified")
			return true, nil
		}
	}

	log.Debug("no entries could be verified", slog.Any("entries", entries))
	return false, nil
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

func (v RekorVerifier) verifyEntry(ctx context.Context, version string, entryUUID string) (bool, error) {
	entryRes, err := v.client.Entries.GetLogEntryByUUID(&entries.GetLogEntryByUUIDParams{
		Context:   ctx,
		EntryUUID: entryUUID,
	})
	if err != nil {
		return false, fmt.Errorf("fetching index: %w", err)
	}
	for _, payload := range entryRes.GetPayload() {
		if err := cosign.VerifyTLogEntryOffline(ctx, &payload, v.pubs); err != nil {
			return false, fmt.Errorf("verifying entry: %w", err)
		}
		dec, err := base64.StdEncoding.DecodeString(payload.Body.(string))
		if err != nil {
			return false, fmt.Errorf("decoding entry: %w", err)
		}
		pe, err := models.UnmarshalProposedEntry(bytes.NewReader(dec), runtime.JSONConsumer())
		if err != nil {
			return false, fmt.Errorf("unmarshaling proposed entry: %w", err)
		}

		var pubKeyString string
		switch entry := pe.(type) {
		case *models.Intoto:
			d := entry.Spec.(map[string]any)
			pubKeyString = d["publicKey"].(string)

		case *models.Hashedrekord:
			d := entry.Spec.(map[string]any)
			sig := d["signature"].(map[string]any)
			pk := sig["publicKey"].(map[string]any)
			pubKeyString = pk["content"].(string)

		case *models.DSSE:
			d := entry.Spec.(map[string]any)
			sigs := d["signatures"].([]any)
			firstSig := sigs[0].(map[string]any)
			pubKeyString = firstSig["verifier"].(string)

		default:
			return false, fmt.Errorf("unsupported entry type: %T", pe)
		}

		pubKeyRaw, err := base64.StdEncoding.DecodeString(pubKeyString)
		if err != nil {
			return false, fmt.Errorf("decoding public key: %w", err)
		}
		block, rest := pem.Decode(pubKeyRaw)
		if len(rest) > 0 {
			return false, fmt.Errorf("extra data after PEM block")
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return false, fmt.Errorf("parsing public key: %w", err)
		}

		if ok, err := v.id.Verify(version, cert); err != nil {
			return false, fmt.Errorf("parsing extensions: %w", err)
		} else if ok {
			return true, nil
		}
	}
	return false, nil
}
