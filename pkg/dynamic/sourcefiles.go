package dynamic

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/thepwagner/debcache/pkg/debian"
)

type FileSource struct {
	dir string
}

var _ PackageSource = (*FileSource)(nil)

func (s FileSource) Packages(_ context.Context) (PackageList, time.Time, error) {
	var ret []debian.Paragraph
	var latest time.Time
	err := filepath.Walk(s.dir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || filepath.Ext(info.Name()) != ".deb" {
			return nil
		}

		if mt := info.ModTime(); mt.After(latest) {
			latest = mt
		}

		pkg, err := ParagraphFromDebFile(path)
		if err != nil {
			return err
		} else if pkg == nil {
			slog.Warn("no control file found", "file", path)
			return nil
		}

		if err := s.addFileData(*pkg, path, info); err != nil {
			return err
		}
		ret = append(ret, *pkg)
		return nil
	})
	if err != nil {
		return nil, time.Time{}, err
	}

	return PackageList{
		"main": {
			"amd64": ret,
		},
	}, latest, nil
}

func (s FileSource) Deb(_ context.Context, filename string) ([]byte, error) {
	return os.ReadFile(filepath.Join(s.dir, filename))
}

func ParagraphFromDebFile(fn string) (*debian.Paragraph, error) {
	f, err := os.Open(fn)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return debian.ParagraphFromDeb(f)
}

func (s FileSource) addFileData(pkg debian.Paragraph, fn string, info fs.FileInfo) error {
	f, err := os.Open(fn)
	if err != nil {
		return err
	}
	defer f.Close()

	md5sum := md5.New()
	sha256sum := sha256.New()
	if _, err := io.Copy(io.MultiWriter(md5sum, sha256sum), f); err != nil {
		return err
	}

	rel, err := filepath.Rel(s.dir, fn)
	if err != nil {
		return err
	}

	pkg["Filename"] = "pool/main/p/pkg/" + rel
	pkg["Size"] = fmt.Sprintf("%d", info.Size())
	pkg["MD5sum"] = fmt.Sprintf("%x", md5sum.Sum(nil))
	pkg["SHA256"] = fmt.Sprintf("%x", sha256sum.Sum(nil))
	return nil
}
