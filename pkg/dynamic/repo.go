package dynamic

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/ProtonMail/go-crypto/openpgp/clearsign"
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
	signer *openpgp.Entity
	src    PackageSource
	maxAge time.Duration

	mu         sync.RWMutex
	renderTime time.Time
	rendered   *RenderedPackages
}

type RepoConfig struct {
	SigningConfig SigningConfig `yaml:",inline"`
	Files         LocalConfig   `yaml:"files"`
}

type RenderedPackages struct {
	inRelease []byte
	packages  map[repo.Component]map[repo.Architecture][]byte
	byHash    map[string][]byte
}

var _ repo.Repo = (*Repo)(nil)

func NewRepo(signer *openpgp.Entity, src PackageSource) *Repo {
	return &Repo{
		signer: signer,
		src:    src,
		maxAge: 5 * time.Minute,
	}
}

func RepoFromConfig(cfg RepoConfig) (*Repo, error) {
	entity, err := EntityFromConfig(cfg.SigningConfig)
	if err != nil {
		return nil, err
	}
	slog.Debug("key loaded", slog.String("fingerprint", fmt.Sprintf("%x", entity.PrimaryKey.Fingerprint)))

	var src PackageSource
	if cfg.Files.Directory != "" {
		src = &LocalSource{
			dir: cfg.Files.Directory,
		}
	}
	if src == nil {
		return nil, fmt.Errorf("no source configured")
	}

	return NewRepo(entity, src), nil
}

func (r *Repo) InRelease(ctx context.Context, dist repo.Distribution) ([]byte, error) {
	if err := r.render(ctx, dist); err != nil {
		return nil, err
	}
	return r.rendered.inRelease, nil
}

func (r *Repo) Packages(ctx context.Context, dist repo.Distribution, component repo.Component, arch repo.Architecture, compression repo.Compression) ([]byte, error) {
	if err := r.render(ctx, dist); err != nil {
		return nil, err
	}

	componentData := r.rendered.packages[component]
	if componentData == nil {
		return nil, nil
	}
	pkgRaw := componentData[arch]

	return compression.Compress(pkgRaw)
}

func (r *Repo) ByHash(ctx context.Context, dist repo.Distribution, _ repo.Component, _ repo.Architecture, digest string) ([]byte, error) {
	if err := r.render(ctx, dist); err != nil {
		return nil, err
	}
	return r.rendered.byHash[digest], nil
}

func (r *Repo) Pool(_ context.Context, _ repo.Component, _, filename string) ([]byte, error) {
	return r.src.Deb(context.Background(), filename)
}

func (r *Repo) SigningKeyPEM() ([]byte, error) {
	var buf bytes.Buffer
	w, err := armor.Encode(&buf, openpgp.PublicKeyType, nil)
	if err != nil {
		return nil, err
	}
	if err := r.signer.Serialize(w); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (r *Repo) render(ctx context.Context, dist repo.Distribution) error {
	// Fast read lock path:
	r.mu.RLock()
	age := time.Since(r.renderTime)
	r.mu.RUnlock()
	if age < r.maxAge {
		slog.Debug("skipping render", slog.Duration("age", age))
		return nil
	}

	// Slow write lock, avoid concurrent renders:
	r.mu.Lock()
	defer r.mu.Unlock()
	if age := time.Since(r.renderTime); age < r.maxAge {
		slog.Debug("skipping render", slog.Duration("age", age))
		return nil
	}

	pkgs, pkgTime, err := r.src.Packages(ctx)
	if err != nil {
		return err
	}
	// If packages have not changed since the last render, we can skip:
	if pkgTime.Before(r.renderTime) {
		slog.Debug("skipping render", slog.Time("pkgTime", pkgTime), slog.Time("renderTime", r.renderTime))
		r.renderTime = time.Now()
		return nil
	}

	slog.Debug("rendering packages", slog.Int("count", len(pkgs)))
	r.renderTime = time.Now()
	rendered, err := r.renderPackages(pkgs, pkgTime, dist)
	if err != nil {
		return err
	}
	r.rendered = rendered
	return nil
}

type inReleaseDigestEntry struct {
	Digest string
	Size   int64
	Path   string
}

var compressors = []repo.Compression{
	repo.CompressionNone,
	repo.CompressionGZIP,
	repo.CompressionXZ,
}

func (r *Repo) renderPackages(pkgs PackageList, pkgTime time.Time, dist repo.Distribution) (*RenderedPackages, error) {
	// We have three things to index:
	var components []string
	var architectures []string
	var digests []inReleaseDigestEntry

	ret := RenderedPackages{
		packages: map[repo.Component]map[repo.Architecture][]byte{},
		byHash:   map[string][]byte{},
	}

	archIndex := map[repo.Architecture]struct{}{}
	for name, component := range pkgs {
		components = append(components, string(name))
		renderedComponent := map[repo.Architecture][]byte{}

		for arch, packages := range component {
			archIndex[arch] = struct{}{}

			var pkgRaw bytes.Buffer
			if err := debian.WriteControlFile(&pkgRaw, packages...); err != nil {
				return nil, err
			}
			renderedComponent[arch] = pkgRaw.Bytes()

			for _, compressor := range compressors {
				compressed, err := compressor.Compress(pkgRaw.Bytes()) // compressed, err? what is the PSI?
				if err != nil {
					return nil, err
				}
				digest := fmt.Sprintf("%x", sha256.Sum256(compressed))
				digests = append(digests, inReleaseDigestEntry{
					Digest: digest,
					Size:   int64(len(compressed)),
					Path:   fmt.Sprintf("%s/binary-%s/Packages%s", name, arch, compressor.Extension()),
				})
				ret.byHash[digest] = compressed
			}
		}
		ret.packages[name] = renderedComponent
	}
	for arch := range archIndex {
		architectures = append(architectures, string(arch))
	}
	sort.Strings(components)
	sort.Strings(architectures)
	sort.Slice(digests, func(i, j int) bool {
		return digests[i].Path < digests[j].Path
	})

	// Use our collect data to render InRelease:
	release := debian.Paragraph{
		"Origin":          "Debian",
		"Label":           "Debian",
		"Architectures":   strings.Join(architectures, " "),
		"Components":      strings.Join(components, " "),
		"Date":            pkgTime.UTC().Format(time.RFC1123Z),
		"Acquire-By-Hash": "yes",
		"Description":     "Debian",
		"Codename":        dist.String(),
	}
	var sha256 strings.Builder
	for _, digest := range digests {
		sha256.WriteString(fmt.Sprintf(" %s  %d %s\n", digest.Digest, digest.Size, digest.Path))
	}
	release["SHA256"] = sha256.String()

	// Sign the release:
	var inRelease bytes.Buffer
	enc, err := clearsign.Encode(&inRelease, r.signer.PrivateKey, nil)
	if err != nil {
		return nil, err
	}
	if err := debian.WriteControlFile(enc, release); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	if _, err = fmt.Fprintln(&inRelease); err != nil {
		return nil, err
	}

	ret.inRelease = inRelease.Bytes()
	return &ret, nil
}
