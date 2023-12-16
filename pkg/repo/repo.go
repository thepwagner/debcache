package repo

import "context"

// Repo is a source for Debian packages.
type Repo interface {
	// InRelease fetches a signed description of the repository and its contents
	InRelease(ctx context.Context, dist string) ([]byte, error)

	Packages(ctx context.Context, dist, component, arch string, compression Compression) ([]byte, error)

	// ByHash fetches metadata (e.g. an architecture's package list) by its hash.
	ByHash(ctx context.Context, dist, component, arch, digest string) ([]byte, error)

	// Pool fetches a package from the pool.
	Pool(ctx context.Context, component, pkg, filename string) ([]byte, error)
}

type Compression string

const (
	CompressionNone = ""
	CompressionGZIP = "gz"
	CompressionXZ   = "xz"
)

func ParseCompression(s string) Compression {
	switch s {
	case "gz", ".gz":
		return CompressionGZIP
	case "xz", ".xz":
		return CompressionXZ
	default:
		return CompressionNone
	}
}

func (c Compression) Extension() string {
	switch c {
	case CompressionGZIP:
		return ".gz"
	case CompressionXZ:
		return ".xz"
	default:
		return ""
	}
}
