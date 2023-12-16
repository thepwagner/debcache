package server

import (
	"fmt"
	"log/slog"

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
	Upstream repo.UpstreamConfig `yaml:"upstream"`
	Cache    repo.CacheConfig    `yaml:"cache"`
}

func BuildRepo(name string, cfg RepoConfig) (repo.Repo, error) {
	slog.Debug("building repo", slog.String("repo", name))

	var base repo.Repo
	var err error
	if cfg.Upstream.URL != "" {
		base, err = repo.UpstreamFromConfig(cfg.Upstream)
	} // TODO: dynamic repos go here
	if err != nil {
		return nil, fmt.Errorf("error building base repo: %w", err)
	} else if base == nil {
		return nil, fmt.Errorf("no repository configured")
	}

	if cfg.Cache.URL == "" {
		return base, nil
	}
	return repo.CacheFromConfig(base, cfg.Cache)
}
