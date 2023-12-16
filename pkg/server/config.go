package server

import (
	"fmt"
	"net/url"

	"github.com/thepwagner/debcache/pkg/repo"
)

type Config struct {
	Server ServerConfig          `yaml:"server"`
	Repos  map[string]RepoConfig `yaml:"repos"`
}

type ServerConfig struct {
	Addr string `yaml:"addr"`
}

type RepoConfig struct {
	Upstream RepoUpstreamConfig `yaml:"upstream"`
	Cache    RepoCacheConfig    `yaml:"cache"`
}

type RepoUpstreamConfig struct {
	URL    string `yaml:"url"`
	Verify bool   `yaml:"verify"`
}

type RepoCacheConfig struct {
	URL string `yaml:"url"`
}

func BuildRepo(cfg RepoConfig) (repo.Repo, error) {
	var base repo.Repo
	if cfg.Upstream.URL != "" {
		u, err := url.Parse(cfg.Upstream.URL)
		if err != nil {
			return nil, fmt.Errorf("error parsing upstream URL: %w", err)
		}
		base = repo.NewUpstream(*u)
	}

	if base == nil {
		return nil, fmt.Errorf("no repository configured")
	}

	// TODO cache is a wrapper here?

	return base, nil
}
