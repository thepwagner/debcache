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

type FileCacheConfig struct {
	Path   string     `yaml:"path"`
	Source RepoConfig `yaml:"source"`
}

type MemoryCacheConfig struct {
	Source RepoConfig `yaml:"source"`
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

	// TODO: this turned into a big mess, and i am too tired to clean it up atm
	// i like the RepoConfig struct
	// i don't like how the cache.Source fields need to be declared locally to avoid circular imports
	// i don't like how much this switch looks like TypedConfig
	//    - maybe TypedConfig dies, it was great for model tests
	//    - maybe we just pass the map[string]any to the package? (probably an exception for cache sources)
	switch cfg.Type {
	case "dynamic":
		dynCfg, err := decodeSource[dynamic.RepoConfig](name, cfg.Config)
		if err != nil {
			return nil, fmt.Errorf("error decoding dynamic config: %w", err)
		}
		return dynamic.RepoFromConfig(ctx, *dynCfg)

	case "file-cache":
		src, err := newCacheSource(ctx, fmt.Sprintf("file-cache.%s", name), cfg.Config["source"])
		if err != nil {
			return nil, fmt.Errorf("error building file-cache source: %w", err)
		}
		cacheCfg, err := decodeSource[cache.FileConfig](name, cfg.Config)
		if err != nil {
			return nil, fmt.Errorf("error decoding file-cache config: %w", err)
		}
		return repo.NewCache(src, cache.NewFileStorage(*cacheCfg)), nil

	case "memory-cache":
		src, err := newCacheSource(ctx, fmt.Sprintf("memory-cache.%s", name), cfg.Config["source"])
		if err != nil {
			return nil, fmt.Errorf("error building memory-cache source: %w", err)
		}
		cacheCfg, err := decodeSource[cache.LRUConfig](name, cfg.Config)
		if err != nil {
			return nil, fmt.Errorf("error decoding memory-cache config: %w", err)
		}
		return repo.NewCache(src, cache.NewLRUStorage(*cacheCfg)), nil

	case "upstream":
		cacheCfg, err := decodeSource[repo.UpstreamConfig](name, cfg.Config)
		if err != nil {
			return nil, fmt.Errorf("error decoding upstream config: %w", err)
		}
		return repo.UpstreamFromConfig(*cacheCfg)
	}

	return nil, fmt.Errorf("unknown repo type %q", cfg.Type)
}

func decodeSource[T any](name string, src any) (*T, error) {
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
	srcCfg, err := decodeSource[RepoConfig](name, src)
	if err != nil {
		return nil, fmt.Errorf("error building file-cache source: %w", err)
	}
	return BuildRepo(ctx, name, *srcCfg)
}
