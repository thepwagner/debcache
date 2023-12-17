package dynamic

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/thepwagner/debcache/pkg/debian"
	"github.com/thepwagner/debcache/pkg/repo"
)

// Component is a Debian component (e.g. "main", "contrib", "non-free").
type Component string

// Architecture is a Debian architecture (e.g. "amd64", "arm64").
type Architecture string

// PackageList are packages indexed by Component and Architecture.
type PackageList map[Component]map[Architecture][]debian.Paragraph

func (pl PackageList) Add(component Component, architecture Architecture, p debian.Paragraph) {
	if pl[component] == nil {
		pl[component] = map[Architecture][]debian.Paragraph{}
	}
	pl[component][architecture] = append(pl[component][architecture], p)
}

type RenderedPackageList struct {
	Digest string
	Size   int64
	Path   string
}

func Render(pl PackageList, compressions ...repo.Compression) ([]RenderedPackageList, error) {
	var ret []RenderedPackageList
	for component, componentData := range pl {
		for architecture, packages := range componentData {
			var pkgRaw bytes.Buffer
			if err := debian.WriteControlFile(&pkgRaw, packages...); err != nil {
				return nil, err
			}

			for _, compressor := range compressions {
				data, err := compressor.Compress(pkgRaw.Bytes())
				if err != nil {
					return nil, err
				}
				digest := fmt.Sprintf("%x", sha256.Sum256(data))
				ret = append(ret, RenderedPackageList{
					Digest: digest,
					Size:   int64(len(data)),
					Path:   fmt.Sprintf("%s/binary-%s/Packages%s", component, architecture, compressor.Extension()),
				})
			}
		}
	}
	return ret, nil
}

func (pl PackageList) Release() (debian.Paragraph, error) {
	var components []string
	archIndex := map[Architecture]struct{}{}
	for name, component := range pl {
		components = append(components, string(name))
		for arch := range component {
			archIndex[arch] = struct{}{}
		}
	}
	var architectures []string
	for arch := range archIndex {
		architectures = append(architectures, string(arch))
	}

	release := debian.Paragraph{
		"Origin":        "Debian",
		"Label":         "Debian",
		"Architectures": strings.Join(architectures, " "),
		"Components":    strings.Join(components, " "),
		// "Suite":    dist,
		// "Codename": dist,
		"Version": "12.4",
		"Date":    time.Now().UTC().Format(time.RFC1123Z),
		// "Acquire-By-Hash": "yes", <-- hash tag squad guals
		"Description": "Debian",
	}

	type digestEntry struct {
		digest string
		size   int64
		path   string
	}
	var digests []digestEntry
	for component, componentData := range pl {
		for architecture, packages := range componentData {
			var pkgRaw bytes.Buffer
			if err := debian.WriteControlFile(&pkgRaw, packages...); err != nil {
				return nil, err
			}

			for _, compression := range []repo.Compression{repo.CompressionNone, repo.CompressionGZIP, repo.CompressionXZ} {
				data, err := compression.Compress(pkgRaw.Bytes())
				if err != nil {
					return nil, err
				}
				digest := fmt.Sprintf("%x", sha256.Sum256(data))
				digests = append(digests, digestEntry{
					digest: digest,
					size:   int64(len(data)),
					path:   fmt.Sprintf("%s/binary-%s/Packages%s", component, architecture, compression.Extension()),
				})
			}
		}
	}

	sort.Slice(digests, func(i, j int) bool {
		return digests[i].path < digests[j].path
	})

	var sha256 strings.Builder
	for _, digest := range digests {
		sha256.WriteString(fmt.Sprintf(" %s  %d %s\n", digest.digest, digest.size, digest.path))
	}
	release["SHA256"] = sha256.String()

	return release, nil
}
