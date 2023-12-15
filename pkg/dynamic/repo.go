package dynamic

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/clearsign"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
	"github.com/thepwagner/debcache/pkg/debian"
	"github.com/thepwagner/debcache/pkg/repo"
)

// Repo is dynamically generated from a PackageSource.
type Repo struct {
	pk *packet.PrivateKey

	Release       debian.Paragraph
	Components    []string
	Architectures []string
	Source        PackageSource
}

type PackageSource interface {
	Packages(ctx context.Context, component, architecture string) ([]debian.Paragraph, time.Time, error)
}

var _ repo.Repo = (*Repo)(nil)

func NewRepo() (*Repo, error) {
	keyIn, err := os.Open("tmp/key.asc")
	if err != nil {
		return nil, err
	}
	defer keyIn.Close()
	key, err := openpgp.ReadArmoredKeyRing(keyIn)
	if err != nil {
		return nil, fmt.Errorf("decoding key: %w", err)
	}

	return &Repo{
		pk:     key[0].PrivateKey,
		Source: &FileSource{"tmp/pool/"},
	}, nil
}

func (r *Repo) InRelease(ctx context.Context, dist string) ([]byte, error) {

	// TODO: organize into package lists, calculate digests
	pkgs, latest, err := r.Source.Packages(ctx, "main", "amd64")
	if err != nil {
		return nil, err
	}
	slog.Info("dynamic InRelease", "latest", latest, "packages", pkgs)

	var buf bytes.Buffer
	enc, err := clearsign.Encode(&buf, r.pk, nil)
	if err != nil {
		return nil, err
	}

	release := debian.Paragraph{
		"Origin":   "Debian",
		"Label":    "Debian",
		"Suite":    dist,
		"Codename": dist,
		"Version":  "12.4",
		"Date":     latest.UTC().Format(time.RFC1123Z),
		// "Acquire-By-Hash": "yes", <-- hash tag squad guals
		"Architectures": "amd64",
		"Components":    "main",
		"Description":   "Debian",
	}
	// FIXME: repo requires "a" digest to not be insecure, slap one in
	release["SHA256"] = " d6c9c82f4e61b4662f9ba16b9ebb379c57b4943f8b7813091d1f637325ddfb79  1484322 contrib/Contents-all\n"

	if err := debian.WriteControlFile(enc, release); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	if _, err = fmt.Fprintln(&buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (r *Repo) ByHash(ctx context.Context, dist, component, arch, digest string) ([]byte, error) {
	return nil, nil
}

func (r *Repo) Pool(ctx context.Context, component, pkg, filename string) ([]byte, error) {
	return nil, nil
}
