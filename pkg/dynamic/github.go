package dynamic

import (
	"bufio"
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"hash"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"regexp"
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
	repos         map[string]*releaseRepo
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

type releaseRepo struct {
	ChecksumFile string
	verifier     signature.Verifier
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

	repos := make(map[string]*releaseRepo, len(config.Repositories))
	for repoName, repoConfig := range config.Repositories {
		log := slog.With(slog.String("github_repository", repoName))

		release := releaseRepo{
			ChecksumFile: repoConfig.ChecksumFile,
		}
		if repoConfig.Signer != nil {
			verifier, err := signature.NewRekorVerifier(ctx, *repoConfig.Signer)
			if err != nil {
				return nil, fmt.Errorf("failed to create rekor verifier: %w", err)
			}
			log.Debug("using rekor verifier")
			release.verifier = verifier
		} else {
			log.Debug("verification disabled")
			release.verifier = signature.AlwaysPass()
		}
		repos[repoName] = &release
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

var lineRe = regexp.MustCompile(`^([a-f0-9]+)\s+(\S+)$`)

func (gh *GitHubReleasesSource) Packages(ctx context.Context) (PackageList, time.Time, error) {
	ret := PackageList{}
	var latest time.Time
	for ghRepo, releaseRepo := range gh.repos {
		repoName := strings.SplitN(ghRepo, "/", 2)
		verifier := releaseRepo.verifier

		slog.Debug("listing releases", slog.String("repo_owner", repoName[0]), slog.String("repo_name", repoName[1]))
		releases, _, err := gh.github.Repositories.ListReleases(ctx, repoName[0], repoName[1], &github.ListOptions{PerPage: 5})
		if err != nil {
			return nil, time.Time{}, fmt.Errorf("failed to list releases for %q: %w", ghRepo, err)
		}
		for _, release := range releases {
			slog.Debug("inspecting release", slog.String("tag", release.GetTagName()), slog.Int("asset_count", len(release.Assets)))

			// If we're processing checksums, grab that file first:
			checksums, digester, err := gh.getCheckSums(ctx, ghRepo, *releaseRepo, release)
			if err != nil {
				return nil, time.Time{}, fmt.Errorf("getting checksums: %w", err)
			}

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

				// Download and verify the .deb file:
				b, err := gh.get(ctx, ghRepo, fn, ass.GetBrowserDownloadURL())
				if err != nil {
					return nil, time.Time{}, fmt.Errorf("getting asset: %w", err)
				}
				if len(checksums) == 0 {
					// If we're not processing checksums, attempt to verify the .deb directly
					if ok, err := verifier.Verify(ctx, release.GetTagName(), b); err != nil {
						return nil, time.Time{}, fmt.Errorf("verifying asset: %w", err)
					} else if !ok {
						log.Warn("deb failed verification")
						continue
					}
				} else {
					// Validate the checksum - don't worry about signature verification (the checksum file may have been signed, but we don't care here)
					// It is OK that we might calculate sha256(b) twice.
					expected, ok := checksums[fn]
					if !ok {
						return nil, time.Time{}, fmt.Errorf("deb not found in checksum file: %s", fn)
					}

					hash := digester()
					if _, err := hash.Write(b); err != nil {
						return nil, time.Time{}, fmt.Errorf("calculating checksum: %w", err)
					}
					if actual := fmt.Sprintf("%x", hash.Sum(nil)); actual != expected {
						return nil, time.Time{}, fmt.Errorf("checksum mismatch on %s: expected %s, got %s", fn, expected, actual)
					}
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

func (gh *GitHubReleasesSource) get(ctx context.Context, ghRepo, fn, url string) ([]byte, error) {
	key := assets.Key(strings.ReplaceAll(ghRepo, "/", "_") + "_" + fn)
	if b, ok := gh.cache.Get(ctx, key); ok {
		return b, nil
	}
	res, err := gh.http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("getting asset: %w", err)
	}
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading asset: %w", err)
	}
	gh.cache.Add(ctx, key, b)
	return b, nil
}

func (gh *GitHubReleasesSource) getCheckSums(ctx context.Context, ghRepo string, repoCfg releaseRepo, release *github.RepositoryRelease) (map[string]string, func() hash.Hash, error) {
	if repoCfg.ChecksumFile == "" {
		return nil, nil, nil
	}

	checksumFile := strings.ReplaceAll(repoCfg.ChecksumFile, "{{VERSION}}", release.GetTagName())
	slog.Debug("looking for checksum file", slog.String("filename", checksumFile))

	expectedChecksums := make(map[string]string)
	for _, ass := range release.Assets {
		fn := (*ass).GetName()
		if fn != checksumFile {
			continue
		}

		// Download and verify the signature of the checksum file:
		b, err := gh.get(ctx, ghRepo, fn, ass.GetBrowserDownloadURL())
		if err != nil {
			return nil, nil, fmt.Errorf("downloading checksum file: %w", err)
		}
		if ok, err := repoCfg.verifier.Verify(ctx, release.GetTagName(), b); err != nil {
			return nil, nil, fmt.Errorf("verifying asset: %w", err)
		} else if !ok {
			slog.Warn("deb failed verification")
			continue
		}

		// Parse the file into a map of filenames to encoded hash:
		var digestLen int
		scanner := bufio.NewScanner(bytes.NewReader(b))
		for scanner.Scan() {
			m := lineRe.FindAllStringSubmatch(scanner.Text(), 1)
			if len(m) == 0 {
				continue
			}

			if dl := len(m[0][1]); digestLen == 0 {
				digestLen = dl
			} else if dl != digestLen {
				return nil, nil, fmt.Errorf("invalid checksum file - lengths change from %d to %d", dl, digestLen)
			}
			expectedChecksums[m[0][2]] = m[0][1]
		}
		if err := scanner.Err(); err != nil {
			return nil, nil, fmt.Errorf("reading checksums file: %w", err)
		}

		// Use the detected digest length to guess the digest algorithm:
		var digester func() hash.Hash
		switch digestLen {
		case sha256.Size * 2:
			digester = sha256.New
		case sha512.Size * 2:
			digester = sha512.New
		default:
			return nil, nil, fmt.Errorf("unknown digest length: %d", digestLen)
		}

		return expectedChecksums, digester, nil
	}

	return nil, nil, fmt.Errorf("checksum file %q not found", checksumFile)
}
