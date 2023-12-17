package dynamic

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/clearsign"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
	"github.com/thepwagner/debcache/pkg/debian"
	"github.com/thepwagner/debcache/pkg/repo"
)

// PackageSource provides package data for the Repo.
type PackageSource interface {
	Packages(ctx context.Context) (PackageList, error)
	Deb(ctx context.Context, filename string) ([]byte, error)
}

// Repo is dynamically generated from a PackageSource.
type Repo struct {
	key *packet.PrivateKey

	src PackageSource
}

type RepoConfig struct {
	SigningKey     string `yaml:"signingKey"`
	SigningKeyPath string `yaml:"signingKeyPath"`
}

var _ repo.Repo = (*Repo)(nil)

func NewRepo(key *packet.PrivateKey, src PackageSource) *Repo {
	return &Repo{
		key: key,
		src: src,
	}
}

func RepoFromConfig(cfg RepoConfig) (*Repo, error) {
	var key *packet.PrivateKey
	var err error
	if cfg.SigningKey != "" {
		slog.Debug("reading key from config")
		key, err = KeyFromReader(strings.NewReader(cfg.SigningKey))
	} else if cfg.SigningKeyPath != "" {
		slog.Debug("reading key from file", slog.String("path", cfg.SigningKeyPath))
		var f io.ReadCloser
		f, err = os.Open(cfg.SigningKeyPath)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		key, err = KeyFromReader(f)
	}
	if err != nil {
		return nil, err
	}

	// FIXME: more config here plz
	src := &FileSource{
		dir: "tmp/debs/",
	}

	return NewRepo(key, src), nil
}

func KeyFromReader(in io.Reader) (*packet.PrivateKey, error) {
	keyRing, err := openpgp.ReadArmoredKeyRing(in)
	if err != nil {
		return nil, fmt.Errorf("decoding key: %w", err)
	}
	return keyRing[0].PrivateKey, nil
}

func (r *Repo) InRelease(ctx context.Context, dist string) ([]byte, error) {
	var buf bytes.Buffer
	enc, err := clearsign.Encode(&buf, r.key, nil)
	if err != nil {
		return nil, err
	}

	pkgs, err := r.src.Packages(ctx)
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
	pkgs, err := r.src.Packages(ctx)
	if err != nil {
		return nil, err
	}
	slog.Info("dynamic Packages")
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
	return r.src.Deb(context.Background(), filename)
}
