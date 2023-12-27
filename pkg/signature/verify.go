package signature

import "context"

type Config struct {
	// ChecksumsFile is the name of the checksums file. If set, this is signed instead of the individual .deb.x
	ChecksumsFile string `yaml:"checksums"`

	Cosign FulcioIdentity `yaml:"cosign"`
}

type Verifier interface {
	Verify(ctx context.Context, version string, deb []byte) (bool, error)
}

func AlwaysPass() Verifier {
	return fixedVerifier(true)
}

func AlwaysFail() Verifier {
	return fixedVerifier(false)
}

type fixedVerifier bool

func (v fixedVerifier) Verify(_ context.Context, _ string, _ []byte) (bool, error) {
	return bool(v), nil
}
