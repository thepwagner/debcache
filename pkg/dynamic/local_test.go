package dynamic_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thepwagner/debcache/pkg/dynamic"
)

func TestLocalSource_Packages(t *testing.T) {
	t.Parallel()

	t.Run("no packages found", func(t *testing.T) {
		t.Parallel()
		src := dynamic.NewLocalSource(dynamic.LocalConfig{Directory: "testdata"})
		pkgs, ts, err := src.Packages(context.Background())
		require.NoError(t, err)
		assert.Empty(t, pkgs)
		assert.True(t, ts.IsZero())
	})

	t.Run("packages found", func(t *testing.T) {
		t.Parallel()
		src := dynamic.NewLocalSource(dynamic.LocalConfig{Directory: "../debian/testdata"})
		pkgs, ts, err := src.Packages(context.Background())
		require.NoError(t, err)
		assert.Len(t, pkgs, 1)
		assert.False(t, ts.IsZero())

		assert.Len(t, pkgs["main"], 1)
		assert.Len(t, pkgs["main"]["amd64"], 1)
		pkg := pkgs["main"]["amd64"][0]
		assert.Equal(t, "foobar", pkg["Package"])
		assert.Equal(t, "fbf9896877560712845d314e00112d916919eb670f3400e514baabafe880386b", pkg["SHA256"])
	})
}

func TestLocalSource_Deb(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	src := dynamic.NewLocalSource(dynamic.LocalConfig{Directory: "../debian/testdata"})

	t.Run("package not found", func(t *testing.T) {
		t.Parallel()
		_, err := src.Deb(ctx, "does-not-exist.deb")
		require.Error(t, err)
	})

	t.Run("package found", func(t *testing.T) {
		t.Parallel()
		deb, err := src.Deb(ctx, "foobar_1.2.3_amd64.deb")
		require.NoError(t, err)
		assert.NotEmpty(t, deb)
	})
}
