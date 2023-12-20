package dynamic_test

import (
	"os"
	"testing"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thepwagner/debcache/pkg/dynamic"
)

const keyPath = "testdata/key.asc"

func TestEntityFromConfig(t *testing.T) {
	t.Parallel()
	const keyID = "5E2A467F0CA65061"

	t.Run("no key", func(t *testing.T) {
		t.Parallel()
		_, err := dynamic.EntityFromConfig(dynamic.SigningConfig{})
		assert.Error(t, err)
	})

	t.Run("from path", func(t *testing.T) {
		t.Parallel()
		ent, err := dynamic.EntityFromConfig(dynamic.SigningConfig{SigningKeyPath: keyPath})
		require.NoError(t, err)
		assert.Equal(t, keyID, ent.PrimaryKey.KeyIdString())
	})

	t.Run("from string", func(t *testing.T) {
		t.Parallel()
		b, err := os.ReadFile(keyPath)
		require.NoError(t, err)
		ent, err := dynamic.EntityFromConfig(dynamic.SigningConfig{SigningKey: string(b)})
		require.NoError(t, err)
		assert.Equal(t, keyID, ent.PrimaryKey.KeyIdString())
	})

	t.Run("invalid key", func(t *testing.T) {
		t.Parallel()
		_, err := dynamic.EntityFromConfig(dynamic.SigningConfig{SigningKey: "not a valid key"})
		require.Error(t, err)
	})
}

func testKey(tb testing.TB) *openpgp.Entity {
	tb.Helper()
	ent, err := dynamic.EntityFromConfig(dynamic.SigningConfig{SigningKeyPath: keyPath})
	require.NoError(tb, err)
	return ent
}
