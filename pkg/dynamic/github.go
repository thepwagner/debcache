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
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/go-github/v61/github"
	"github.com/thepwagner/debcache/pkg/cache"
	"github.com/thepwagner/debcache/pkg/debian"
	"github.com/thepwagner/debcache/pkg/repo"
	"github.com/thepwagner/debcache/pkg/signature"
)

type GitHubReleasesSource struct {
	github *github.Client

	cache         cache.Storage
	architectures map[repo.Architecture]struct{}
	repos         map[string]*releaseRepo
}

var _ PackageSource = (*GitHubReleasesSource)(nil)

type GitHubReleasesConfig struct {
	Token string `yaml:"token"`
	// Repositories is a map of GitHub repository `owner/name`s to configuration.
	Repositories  map[string]GitHubReleasesRepoConfig `yaml:"repositories"`
	Architectures []repo.Architecture                 `yaml:"architectures"`
	// Cache will be used for storing downloaded assets.
	Cache cache.FileConfig `yaml:"cache"`
}

type GitHubReleasesRepoConfig struct {
	Signer       *signature.FulcioIdentity `yaml:"rekor-signer"`
	ChecksumFile string                    `yaml:"checksums"`
}

type releaseRepo struct {
	ChecksumFile string
	verifier     signature.Verifier
}

func NewGitHubReleasesSource(ctx context.Context, config GitHubReleasesConfig) (*GitHubReleasesSource, error) {
	client := github.NewClient(&http.Client{})
	if config.Token != "" {
		var tok string
		if strings.HasPrefix(config.Token, "env.") {
			tok = os.Getenv(strings.TrimPrefix(config.Token, "env."))
		} else {
			tok = config.Token
		}
		client = client.WithAuthToken(tok)
	}

	arches := make(map[repo.Architecture]struct{}, len(config.Architectures))
	for _, arch := range config.Architectures {
		arches[arch] = struct{}{}

		// Hack for aquasec/trivy file naming...
		switch arch {
		case "amd64":
			arches["Linux-64bit"] = struct{}{}
		case "arm64":
			arches["Linux-ARM64"] = struct{}{}
		}
	}
	if len(arches) == 0 {
		arches["amd64"] = struct{}{}
		arches["Linux-64bit"] = struct{}{}
	}

	var storage cache.Storage
	if config.Cache.Path == "" {
		storage = cache.NewLRUStorage(cache.LRUConfig{})
		slog.Warn("github cache disabled, don't use this in production")
	} else {
		storage = cache.NewFileStorage(config.Cache)
		slog.Debug("github cache set up", slog.String("path", config.Cache.Path))
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

	slog.Debug("github releases repo", slog.Int("repo_count", len(repos)), slog.Any("arches", arches))
	return &GitHubReleasesSource{
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
			if release.GetDraft() || release.GetPrerelease() {
				continue
			}

			// If we're processing checksums, grab that file first:
			checksums, digester, err := gh.getCheckSums(ctx, repoName[0], repoName[1], *releaseRepo, release)
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
				log := slog.With(slog.String("fn", fn))

				// Focus only on the architectures of interest:
				debArch := repo.Architecture(fn[strings.LastIndex(fn, "_")+1 : len(fn)-4])
				if _, ok := gh.architectures[debArch]; !ok {
					log.Debug("release has other arch deb")
					continue
				}
				log.Debug("release has deb asset")

				// Download and verify the .deb file:
				b, err := gh.get(ctx, repoName[0], repoName[1], ass.GetID())
				if err != nil {
					return nil, time.Time{}, fmt.Errorf("getting asset: %w", err)
				}
				log.Debug("asset download complete", slog.Int("bytes", len(b)))

				if len(checksums) == 0 {
					// If we're not processing checksums, attempt to verify the .deb directly
					if ok, err := verifier.Verify(ctx, release.GetTagName(), b); err != nil {
						return nil, time.Time{}, fmt.Errorf("verifying asset: %w", err)
					} else if !ok {
						log.Warn("deb failed verification")
						continue
					}
					log.Debug("file passed signature verification")
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
					log.Debug("checksum verified", slog.String("expected", expected))
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

				pkg["Filename"] = "pool/main/p/pkg/" + fmt.Sprintf("%s_%s_%d.deb", repoName[0], repoName[1], ass.GetID())
				pkg["Size"] = fmt.Sprintf("%d", len(b))
				pkg["MD5sum"] = fmt.Sprintf("%x", md5sum.Sum(nil))
				pkg["SHA256"] = fmt.Sprintf("%x", sha256sum.Sum(nil))

				if assetTime := ass.GetUpdatedAt().Time; assetTime.After(latest) {
					latest = assetTime
				}
				ret.Add("main", repo.Architecture(pkg["Architecture"]), pkg)
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
	filename = strings.TrimPrefix(filename, "main/p/pkg/")
	key := assets.Key(filename)
	b, ok := gh.cache.Get(ctx, key)
	slog.Debug("github serving deb", slog.Bool("ok", ok), slog.Any("cache_key", key))
	if ok {
		return b, nil
	}
	return nil, nil
}

func (gh *GitHubReleasesSource) get(ctx context.Context, owner, repo string, assetID int64) ([]byte, error) {
	key := assets.Key(fmt.Sprintf("%s_%s_%d.deb", owner, repo, assetID))
	if b, ok := gh.cache.Get(ctx, key); ok {
		return b, nil
	}

	body, _, err := gh.github.Repositories.DownloadReleaseAsset(ctx, owner, repo, assetID, http.DefaultClient)
	if err != nil {
		return nil, fmt.Errorf("getting asset %d: %w", assetID, err)
	}
	defer body.Close()

	b, err := io.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("reading asset: %w", err)
	}
	slog.Debug("fetched asset", slog.String("repo_owner", owner), slog.String("repo_name", repo), slog.Int64("asset_id", assetID), slog.Int("bytes", len(b)))

	slog.Debug("github adding deb to cache", slog.Any("cache_key", key))
	gh.cache.Add(ctx, key, b)
	return b, nil
}

func (gh *GitHubReleasesSource) getCheckSums(ctx context.Context, owner, repo string, repoCfg releaseRepo, release *github.RepositoryRelease) (map[string]string, func() hash.Hash, error) {
	if repoCfg.ChecksumFile == "" {
		return nil, nil, nil
	}

	checksumFile := strings.ReplaceAll(repoCfg.ChecksumFile, "{{VERSION}}", release.GetTagName())
	checksumFile = strings.ReplaceAll(checksumFile, "{{VERSION_WITHOUT_V}}", strings.TrimPrefix(release.GetTagName(), "v"))
	slog.Debug("looking for checksum file", slog.String("filename", checksumFile))

	expectedChecksums := make(map[string]string)
	for _, ass := range release.Assets {
		fn := (*ass).GetName()
		if fn != checksumFile {
			continue
		}
		slog.Debug("found checksum asset", slog.String("repo_owner", owner), slog.String("repo_name", repo), slog.Int("asset_id", int(ass.GetID())))

		// Download and verify the signature of the checksum file:
		b, err := gh.get(ctx, owner, repo, ass.GetID())
		if err != nil {
			return nil, nil, fmt.Errorf("downloading checksum file: %w", err)
		}
		if ok, err := repoCfg.verifier.Verify(ctx, release.GetTagName(), b); err != nil {
			return nil, nil, fmt.Errorf("verifying asset: %w", err)
		} else if !ok {
			slog.Warn("deb failed verification")
			continue
		}
		slog.Debug("checksum file passed signature verification")

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
