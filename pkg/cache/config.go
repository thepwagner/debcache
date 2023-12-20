package cache

import (
	"fmt"
	"log/slog"
	"net/url"
	"path/filepath"
	"time"
)

type Config struct {
	URL string `yaml:"url"`
}

func StorageFromConfig(cfg Config) (Storage, error) {
	if cfg.URL == "" {
		slog.Warn("no cache URL specified, using in-memory")
		return NewLRUStorage(100, time.Hour), nil
	}

	u, err := url.Parse(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("error parsing cache URL: %w", err)
	}
	switch u.Scheme {
	case "file":
		p := filepath.Join(u.Hostname(), u.Path)
		slog.Debug("decorating in file cache", slog.String("path", p))
		return NewFileStorage(p, time.Hour), nil

	default:
		return nil, fmt.Errorf("unsupported cache scheme %q", u.Scheme)
	}
}
