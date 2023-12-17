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

// PackageSource provides package data for the Repo.
type PackageSource interface {
	Packages(ctx context.Context) (PackageList, time.Time, error)
	Deb(ctx context.Context, filename string) ([]byte, error)
}

// Repo is dynamically generated from a PackageSource.
type Repo struct {
	pk *packet.PrivateKey

	Source PackageSource
}

type RepoConfig struct {
	SigningKey string
}

var _ repo.Repo = (*Repo)(nil)

func NewRepo() (*Repo, error) {
	// FIXME: from config
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

	pkgs, _, err := r.Source.Packages(ctx)
	if err != nil {
		return nil, err
	}
	release, err := pkgs.Release()
	if err != nil {
		return nil, err
	}
	release["Codename"] = dist
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

func (r *Repo) Packages(ctx context.Context, _, component, arch string, compression repo.Compression) ([]byte, error) {
	pkgs, latest, err := r.Source.Packages(ctx)
	if err != nil {
		return nil, err
	}
	slog.Info("dynamic Packages", "latest", latest)
	componentData := pkgs[Component(component)]
	if componentData == nil {
		return nil, nil
	}
	packageList := componentData[Architecture(arch)]

	var pkgRaw bytes.Buffer
	if err := debian.WriteControlFile(&pkgRaw, packageList...); err != nil {
		return nil, err
	}

	return compression.Compress(pkgRaw.Bytes())
}

//nolint:revive
func (r *Repo) ByHash(ctx context.Context, dist, component, arch, digest string) ([]byte, error) {
	return nil, nil
}

func (r *Repo) Pool(_ context.Context, _, _, filename string) ([]byte, error) {
	return r.Source.Deb(context.Background(), filename)
}
