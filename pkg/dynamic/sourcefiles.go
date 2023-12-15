package dynamic

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/thepwagner/debcache/pkg/debian"
)

type FileSource struct {
	dir string
}

var _ PackageSource = (*FileSource)(nil)

func (s FileSource) Packages(ctx context.Context, component, architecture string) ([]debian.Paragraph, time.Time, error) {
	files, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, time.Time{}, err
	}

	var ret []debian.Paragraph
	var latest time.Time
	for _, file := range files {
		info, err := file.Info()
		if err != nil {
			return nil, time.Time{}, err
		}

		if mt := info.ModTime(); mt.After(latest) {
			latest = mt
		}

		f, err := os.Open(filepath.Join(s.dir, file.Name()))
		if err != nil {
			return nil, time.Time{}, err
		}

		pkg, err := debian.ParagraphFromDeb(f)
		if err != nil {
			return nil, time.Time{}, err
		} else if pkg != nil {
			ret = append(ret, *pkg)
		}
	}

	return ret, latest, nil
}
