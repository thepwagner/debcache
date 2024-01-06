package repo

import (
	"context"
)

// Distribution is a Debian distribution (e.g. "bookworm").
type Distribution string

// Component is a Debian component (e.g. "main", "contrib", "non-free").
type Component string

// Architecture is a Debian architecture (e.g. "amd64", "arm64").
type Architecture string

// Language is a Debian translation (e.g. "en").
type Language string

func (d Distribution) String() string { return string(d) }
func (c Component) String() string    { return string(c) }
func (a Architecture) String() string { return string(a) }
func (l Language) String() string     { return string(l) }

// Repo is a source for Debian packages.
type Repo interface {
	// InRelease fetches a signed description of the repository and its contents
	InRelease(ctx context.Context, dist Distribution) ([]byte, error)

	Packages(ctx context.Context, dist Distribution, component Component, arch Architecture, compression Compression) ([]byte, error)
	Translations(ctx context.Context, dist Distribution, component Component, lang Language, compression Compression) ([]byte, error)

	// ByHash fetches metadata (e.g. an architecture's package list) by its hash.
	ByHash(ctx context.Context, dist Distribution, component Component, arch Architecture, digest string) ([]byte, error)

	// Pool fetches a package from the pool.
	Pool(ctx context.Context, filename string) ([]byte, error)

	// SigningKeyPEM returns the signing key in PEM format.
	SigningKeyPEM() ([]byte, error)
}
