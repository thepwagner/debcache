package debian_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thepwagner/debcache/pkg/debian"
)

func TestParagraphFromDeb(t *testing.T) {
	t.Parallel()
	f, err := os.Open("testdata/foobar_1.2.3_amd64.deb")
	require.NoError(t, err)
	defer f.Close()

	graph, err := debian.ParagraphFromDeb(f)
	require.NoError(t, err)
	assert.Equal(t, &debian.Paragraph{
		"Architecture":   "amd64",
		"Description":    "debcache test package",
		"Installed-Size": "0",
		"Maintainer":     "pwagner",
		"Package":        "foobar",
		"Priority":       "optional",
		"Section":        "",
		"Version":        "1.2.3",
	}, graph)
}
