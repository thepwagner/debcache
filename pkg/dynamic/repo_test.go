package dynamic_test

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"testing"

	"github.com/ProtonMail/go-crypto/openpgp/packet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thepwagner/debcache/pkg/dynamic"
	"github.com/thepwagner/debcache/pkg/repo"
)

func TestRepo(t *testing.T) {
	t.Parallel()

	r := dynamic.NewRepo(testKey(t), TestSource{
		pkgs: dynamic.PackageList{
			"main": {
				"amd64": {
					{
						"Package":      "test",
						"Version":      "1.0.0",
						"Architecture": "amd64",
					},
				},
				"arm64": {
					{
						"Package":      "test",
						"Version":      "1.0.0",
						"Architecture": "arm64",
					},
				},
			},
			"non-free": {
				"amd64": {
					{
						"Package":      "non-free-test",
						"Version":      "1.0.0",
						"Architecture": "amd64",
					},
				},
			},
		},
	})

	ctx := context.Background()
	const dist = "bookworm"

	t.Run("InRelease", func(t *testing.T) {
		t.Parallel()
		rel, err := r.InRelease(ctx, dist)
		require.NoError(t, err)

		inRelease := string(rel)
		t.Log(inRelease)
		assert.Contains(t, inRelease, "-----BEGIN PGP SIGNED MESSAGE-----\n")
		assert.Contains(t, inRelease, "\n-----BEGIN PGP SIGNATURE-----\n")
		assert.Contains(t, inRelease, "\n-----END PGP SIGNATURE-----\n")
		assert.Contains(t, inRelease, "Architectures: amd64 arm64\n")
		assert.Contains(t, inRelease, "Components: main non-free\n")
		assert.Contains(t, inRelease, "ea33fecc7fdfd25ab13ce9cad3258493bba0c80cf3646b6589a7b8dae12c7c2b  49 main/binary-amd64/Packages")
		assert.Contains(t, inRelease, "cc2e941ff9f66e98d23268a249eda3384e6d514a903746e77c8f260f4ca71fa6  49 main/binary-arm64/Packages")
	})

	t.Run("Packages", func(t *testing.T) {
		t.Parallel()

		for _, arch := range []string{"amd64", "arm64"} {
			pkgs, err := r.Packages(ctx, dist, "main", arch, repo.CompressionNone)
			require.NoError(t, err)

			packages := string(pkgs)
			t.Log(packages)
			assert.Contains(t, packages, "Package: test\n")
			assert.Contains(t, packages, "Version: 1.0.0\n")
			assert.Len(t, pkgs, 49)

			digest := fmt.Sprintf("%x", sha256.Sum256(pkgs))
			if arch == "amd64" {
				assert.Equal(t, "ea33fecc7fdfd25ab13ce9cad3258493bba0c80cf3646b6589a7b8dae12c7c2b", digest)
			} else {
				assert.Equal(t, "cc2e941ff9f66e98d23268a249eda3384e6d514a903746e77c8f260f4ca71fa6", digest)
			}
		}
	})
}

type TestSource struct {
	pkgs dynamic.PackageList
}

func (t TestSource) Packages(_ context.Context) (dynamic.PackageList, error) {
	return t.pkgs, nil
}

func (t TestSource) Deb(_ context.Context, _ string) ([]byte, error) {
	return nil, nil
}

func testKey(tb testing.TB) *packet.PrivateKey {
	tb.Helper()
	f, err := os.Open("testdata/key.asc")
	require.NoError(tb, err)
	defer f.Close()
	k, err := dynamic.KeyFromReader(f)
	require.NoError(tb, err)
	return k
}
