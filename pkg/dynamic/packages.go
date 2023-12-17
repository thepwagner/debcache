package dynamic

import (
	"github.com/thepwagner/debcache/pkg/debian"
	"github.com/thepwagner/debcache/pkg/repo"
)

// PackageList are packages indexed by Component and Architecture.
type PackageList map[repo.Component]map[repo.Architecture][]debian.Paragraph

func (pl PackageList) Add(component repo.Component, architecture repo.Architecture, p debian.Paragraph) {
	if pl[component] == nil {
		pl[component] = map[repo.Architecture][]debian.Paragraph{}
	}
	pl[component][architecture] = append(pl[component][architecture], p)
}

type RenderedPackages struct {
	inRelease []byte
	packages  map[repo.Component]map[repo.Architecture][]byte
	byHash    map[string][]byte
}
