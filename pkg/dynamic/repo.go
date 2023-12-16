package dynamic

import (
	"bytes"
	"context"
	"crypto/sha256"
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
	Deb(ctx context.Context, filename string) ([]byte, error)
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
		pk: key[0].PrivateKey,
		Source: &FileSource{
			dir: "tmp/debs/",
		},
	}, nil
}

func (r *Repo) InRelease(ctx context.Context, dist string) ([]byte, error) {
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
		"Date":     time.Now().UTC().Format(time.RFC1123Z),
		// "Acquire-By-Hash": "yes", <-- hash tag squad guals
		"Architectures": "amd64",
		"Components":    "main",
		"Description":   "Debian",
	}

	// FIXME: Hack for an initial end to end:

	packages, err := r.Packages(ctx, "bookworm", "main", "amd64", repo.CompressionNone)
	if err != nil {
		return nil, err
	}

	dig := sha256.New()
	dig.Write(packages)

	// FIXME: repo requires "a" digest to not be insecure, slap one in
	release["SHA256"] = fmt.Sprintf(" %x  %d main/binary-amd64/Packages\n", dig.Sum(nil), len(packages))

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

func (r *Repo) Packages(ctx context.Context, dist, component, arch string, compression repo.Compression) ([]byte, error) {
	pkgs, latest, err := r.Source.Packages(ctx, component, arch)
	if err != nil {
		return nil, err
	}
	slog.Info("dynamic Packages", "latest", latest)

	var buf bytes.Buffer
	if err := debian.WriteControlFile(&buf, pkgs...); err != nil {
		return nil, fmt.Errorf("writing Packages: %w", err)
	}
	return buf.Bytes(), nil
}

func (r *Repo) ByHash(ctx context.Context, dist, component, arch, digest string) ([]byte, error) {
	return nil, nil
}

func (r *Repo) Pool(_ context.Context, _, _, filename string) ([]byte, error) {
	return r.Source.Deb(context.Background(), filename)
}
