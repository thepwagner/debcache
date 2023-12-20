package dynamic

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/ProtonMail/go-crypto/openpgp"
)

// SigningConfig defines how to sign the repository.
type SigningConfig struct {
	SigningKey     string `yaml:"signingKey"`
	SigningKeyPath string `yaml:"signingKeyPath"`
}

// EntityFromConfig reads an Entity from the SigningConfig.
func EntityFromConfig(cfg SigningConfig) (*openpgp.Entity, error) {
	var entity *openpgp.Entity
	var err error
	if cfg.SigningKey != "" {
		slog.Debug("reading key from config")
		entity, err = EntityFromReader(strings.NewReader(cfg.SigningKey))
	} else if cfg.SigningKeyPath != "" {
		slog.Debug("reading key from file", slog.String("path", cfg.SigningKeyPath))
		var f io.ReadCloser
		f, err = os.Open(cfg.SigningKeyPath)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		entity, err = EntityFromReader(f)
	} else {
		return nil, fmt.Errorf("no signing key provided")
	}
	if err != nil {
		return nil, fmt.Errorf("reading key: %w", err)
	}
	return entity, nil
}

// EntityFromReader reads an Entity from an io.Reader.
func EntityFromReader(in io.Reader) (*openpgp.Entity, error) {
	keyRing, err := openpgp.ReadArmoredKeyRing(in)
	if err != nil {
		return nil, fmt.Errorf("decoding key: %w", err)
	}
	return keyRing[0], nil
}
