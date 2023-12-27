package dynamic

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha256"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/go-github/v57/github"
	"github.com/thepwagner/debcache/pkg/cache"
	"github.com/thepwagner/debcache/pkg/debian"
	"github.com/thepwagner/debcache/pkg/repo"
	"github.com/thepwagner/debcache/pkg/signature"
)

type GitHubReleasesSource struct {
	http   *http.Client
	github *github.Client

	cache         cache.Storage
	architectures map[repo.Architecture]struct{}
	repos         map[string]signature.Verifier
}

var _ PackageSource = (*GitHubReleasesSource)(nil)

type GitHubReleasesConfig struct {
	// Repositories is a map of GitHub repository `owner/name`s to configuration.
	Repositories  map[string]GitHubReleasesRepoConfig `yaml:"repositories"`
	Architectures []repo.Architecture                 `yaml:"architectures"`
	// Cache will be used for storing downloaded assets.
	Cache cache.Config `yaml:"cache"`
}

type GitHubReleasesRepoConfig struct {
	Signer       *signature.FulcioIdentity `yaml:"rekor-signer"`
	ChecksumFile string                    `yaml:"checksum_file"`
}

func NewGitHubReleasesSource(ctx context.Context, config GitHubReleasesConfig) (*GitHubReleasesSource, error) {
	client := github.NewClient(http.DefaultClient)

	arches := make(map[repo.Architecture]struct{}, len(config.Architectures))
	for _, arch := range config.Architectures {
		arches[arch] = struct{}{}
	}
	if len(arches) == 0 {
		arches["amd64"] = struct{}{}
	}

	storage, err := cache.StorageFromConfig(config.Cache)
	if err != nil {
		return nil, err
	}

	repos := make(map[string]signature.Verifier, len(config.Repositories))
	for repoName, repoConfig := range config.Repositories {
		log := slog.With(slog.String("github_repository", repoName))
		var verifier signature.Verifier
		if repoConfig.Signer != nil {
			verifier, err = signature.NewRekorVerifier(ctx, *repoConfig.Signer)
			if err != nil {
				return nil, fmt.Errorf("failed to create rekor verifier: %w", err)
			}
			log.Debug("using rekor verifier")
		} else {
			verifier = signature.AlwaysPass()
			log.Debug("verification disabled")
		}

		repos[repoName] = verifier
	}

	return &GitHubReleasesSource{
		http:          http.DefaultClient,
		github:        client,
		repos:         repos,
		cache:         storage,
		architectures: arches,
	}, nil
}

const assets = cache.Namespace("github-release-assets")

func (gh *GitHubReleasesSource) Packages(ctx context.Context) (PackageList, time.Time, error) {
	ret := PackageList{}
	var latest time.Time
	for ghRepo := range gh.repos {
		repoName := strings.SplitN(ghRepo, "/", 2)
		verifier := gh.repos[ghRepo]

		slog.Debug("listing releases", slog.String("repo_owner", repoName[0]), slog.String("repo_name", repoName[1]))
		releases, _, err := gh.github.Repositories.ListReleases(ctx, repoName[0], repoName[1], &github.ListOptions{PerPage: 5})
		if err != nil {
			return nil, time.Time{}, fmt.Errorf("failed to list releases for %q: %w", ghRepo, err)
		}
		for _, release := range releases {
			slog.Debug("inspecting release", slog.String("tag", release.GetTagName()), slog.Int("asset_count", len(release.Assets)))
			var hasDeb bool
			for _, ass := range release.Assets {
				// Skip non-.deb:
				fn := (*ass).GetName()
				if filepath.Ext(fn) != ".deb" {
					continue
				}

				// Focus only on the architectures of interest:
				debArch := repo.Architecture(fn[strings.LastIndex(fn, "_")+1 : len(fn)-4])
				if _, ok := gh.architectures[debArch]; !ok {
					continue
				}
				log := slog.With(slog.String("fn", fn))
				log.Debug("release has deb asset")

				key := assets.Key(strings.ReplaceAll(ghRepo, "/", "_") + "_" + fn)
				b, ok := gh.cache.Get(ctx, key)
				if !ok {
					res, err := gh.http.Get(ass.GetBrowserDownloadURL())
					if err != nil {
						return nil, time.Time{}, fmt.Errorf("getting asset: %w", err)
					}
					defer res.Body.Close()
					b, err = io.ReadAll(res.Body)
					if err != nil {
						return nil, time.Time{}, fmt.Errorf("reading asset: %w", err)
					}
					gh.cache.Add(ctx, key, b)
				}

				if ok, err := verifier.Verify(ctx, release.GetTagName(), b); err != nil {
					return nil, time.Time{}, fmt.Errorf("verifying asset: %w", err)
				} else if !ok {
					log.Warn("deb failed verification")
					continue
				}

				pkgData, err := debian.ParagraphFromDeb(bytes.NewReader(b))
				if err != nil {
					return nil, time.Time{}, fmt.Errorf("parsing asset: %w", err)
				} else if pkgData == nil {
					slog.Info("package not found in asset")
					continue
				}
				pkg := *pkgData

				md5sum := md5.New()
				sha256sum := sha256.New()
				if _, err := io.MultiWriter(md5sum, sha256sum).Write(b); err != nil {
					return nil, time.Time{}, fmt.Errorf("digesting asset: %w", err)
				}

				pkg["Filename"] = "pool/main/p/pkg/" + strings.ReplaceAll(ghRepo, "/", "_") + "_" + fn
				pkg["Size"] = fmt.Sprintf("%d", len(b))
				pkg["MD5sum"] = fmt.Sprintf("%x", md5sum.Sum(nil))
				pkg["SHA256"] = fmt.Sprintf("%x", sha256sum.Sum(nil))

				if assetTime := ass.GetUpdatedAt().Time; assetTime.After(latest) {
					latest = assetTime
				}
				ret.Add("main", debArch, pkg)
				hasDeb = true
			}

			if hasDeb {
				break
			}
		}
	}
	return ret, latest, nil
}

func (gh *GitHubReleasesSource) Deb(ctx context.Context, filename string) ([]byte, error) {
	key := assets.Key(filename)
	if b, ok := gh.cache.Get(ctx, key); ok {
		return b, nil
	}
	return nil, nil
}
