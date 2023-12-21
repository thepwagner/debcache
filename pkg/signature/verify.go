package signature

import "context"

type Config struct {
	// ChecksumsFile is the name of the checksums file. If set, this is signed instead of the individual .deb.x
	ChecksumsFile string `yaml:"checksums"`

	Cosign CosignConfig `yaml:"cosign"`
}

type Verifier interface {
	Verify(ctx context.Context, deb []byte) (bool, error)
}
