package server

import (
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/thepwagner/debcache/pkg/dynamic"
	"github.com/thepwagner/debcache/pkg/repo"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Addr  string                `yaml:"addr"`
	Repos map[string]RepoConfig `yaml:"repos"`
}

type RepoConfig struct {
	Upstream repo.UpstreamConfig `yaml:"upstream"`
	Dynamic  dynamic.RepoConfig  `yaml:"dynamic"`
	Cache    repo.CacheConfig    `yaml:"cache"`
}

func loadConfig() (*Config, error) {
	var cfg Config

	f, err := os.Open("debcache.yml")
	if err == nil {
		defer f.Close()
		if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
			return nil, fmt.Errorf("error decoding config: %w", err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("error opening config: %w", err)
	} else {
		slog.Info("no config file found, using defaults")
	}

	if cfg.Addr == "" {
		cfg.Addr = ":8080"
	}
	if len(cfg.Repos) == 0 {
		cfg.Repos = map[string]RepoConfig{
			"debian": {
				Upstream: repo.UpstreamConfig{
					URL: "https://deb.debian.org/debian",
				},
			},
		}
	}

	return &cfg, nil
}

func BuildRepo(name string, cfg RepoConfig) (repo.Repo, error) {
	slog.Debug("building repo", slog.String("repo", name))

	var base repo.Repo
	var err error
	if cfg.Upstream.URL != "" {
		base, err = repo.UpstreamFromConfig(cfg.Upstream)
	} else if cfg.Dynamic.SigningKey != "" || cfg.Dynamic.SigningKeyPath != "" {
		base, err = dynamic.RepoFromConfig(cfg.Dynamic)
	}
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
