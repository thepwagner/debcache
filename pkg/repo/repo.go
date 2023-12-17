package repo

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"

	"github.com/ulikunitz/xz"
)

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

func (c Compression) String() string {
	return string(c)
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

func (c Compression) Compress(data []byte) ([]byte, error) {
	switch c {
	case CompressionGZIP:
		var buf bytes.Buffer
		compressor := gzip.NewWriter(&buf)
		if _, err := compressor.Write(data); err != nil {
			return nil, err
		}
		if err := compressor.Close(); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil

	case CompressionXZ:
		var buf bytes.Buffer
		compressor, err := xz.NewWriter(&buf)
		if err != nil {
			return nil, err
		}
		if _, err := compressor.Write(data); err != nil {
			return nil, err
		}
		if err := compressor.Close(); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil

	case CompressionNone:
		return data, nil

	default:
		return nil, fmt.Errorf("unknown compression %q", c)
	}
}
