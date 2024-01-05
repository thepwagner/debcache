package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/thepwagner/debcache/pkg/cache"
	"github.com/thepwagner/debcache/pkg/dynamic"
	"github.com/thepwagner/debcache/pkg/repo"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Addr  string                `yaml:"addr"`
	Repos map[string]RepoConfig `yaml:"repos"`
}

type RepoConfig struct {
	Type   string         `yaml:"type"`
	Config map[string]any `yaml:",inline"`
}

func loadConfig() (*Config, error) {
	var cfg Config

	var cfgPath string
	if p, ok := os.LookupEnv("DEBCACHE_CONFIG"); ok {
		cfgPath = p
	} else {
		cfgPath = "debcache.yml"
	}
	f, err := os.Open(cfgPath)
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
				Type: "upstream",
				Config: map[string]any{
					"url": "https://deb.debian.org/debian",
				},
			},
		}
	}

	return &cfg, nil
}

func BuildRepo(ctx context.Context, name string, cfg RepoConfig) (repo.Repo, error) {
	slog.Debug("building repo", slog.String("repo", name), slog.String("type", cfg.Type), slog.Any("config", cfg.Config))

	switch cfg.Type {
	case "dynamic":
		dynCfg, err := decodeSource[dynamic.RepoConfig](cfg.Config)
		if err != nil {
			return nil, fmt.Errorf("error decoding dynamic config: %w", err)
		}
		return dynamic.RepoFromConfig(ctx, *dynCfg)

	case "file-cache":
		src, err := newCacheSource(ctx, fmt.Sprintf("file-cache.%s", name), cfg.Config["source"])
		if err != nil {
			return nil, fmt.Errorf("error building file-cache source: %w", err)
		}
		cacheCfg, err := decodeSource[cache.FileConfig](cfg.Config)
		if err != nil {
			return nil, fmt.Errorf("error decoding file-cache config: %w", err)
		}
		return repo.NewCache(src, cache.NewFileStorage(*cacheCfg)), nil

	case "memory-cache":
		src, err := newCacheSource(ctx, fmt.Sprintf("memory-cache.%s", name), cfg.Config["source"])
		if err != nil {
			return nil, fmt.Errorf("error building memory-cache source: %w", err)
		}
		cacheCfg, err := decodeSource[cache.LRUConfig](cfg.Config)
		if err != nil {
			return nil, fmt.Errorf("error decoding memory-cache config: %w", err)
		}
		return repo.NewCache(src, cache.NewLRUStorage(*cacheCfg)), nil

	case "upstream":
		cacheCfg, err := decodeSource[repo.UpstreamConfig](cfg.Config)
		if err != nil {
			return nil, fmt.Errorf("error decoding upstream config: %w", err)
		}
		return repo.UpstreamFromConfig(*cacheCfg)
	}

	return nil, fmt.Errorf("unknown repo type %q", cfg.Type)
}

func decodeSource[T any](src any) (*T, error) {
	// mapstructure doesn't work here: so cycle through YAML
	var buf bytes.Buffer
	if err := yaml.NewEncoder(&buf).Encode(src); err != nil {
		return nil, fmt.Errorf("error encoding cache source config: %w", err)
	}
	var cfg T
	if err := yaml.NewDecoder(&buf).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("error decoding cache source config: %w", err)
	}
	return &cfg, nil
}

func newCacheSource(ctx context.Context, name string, src any) (repo.Repo, error) {
	srcCfg, err := decodeSource[RepoConfig](src)
	if err != nil {
		return nil, fmt.Errorf("error building file-cache source: %w", err)
	}
	return BuildRepo(ctx, name, *srcCfg)
}
