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
		} else if mt := info.ModTime(); mt.After(latest) {
			latest = mt
		}

		fn := filepath.Join(s.dir, file.Name())
		pkg, err := ParagraphFromDebFile(fn)
		if err != nil {
			return nil, time.Time{}, err
		} else if pkg == nil {
			slog.Warn("no control file found", "file", fn)
			continue
		}

		if err := addFileData(*pkg, fn, info); err != nil {
			return nil, time.Time{}, err
		}
		ret = append(ret, *pkg)
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

func addFileData(pkg debian.Paragraph, fn string, info fs.FileInfo) error {
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
	pkg["Filename"] = "pool/main/p/pkg/" + info.Name()
	pkg["Size"] = fmt.Sprintf("%d", info.Size())
	pkg["MD5sum"] = fmt.Sprintf("%x", md5sum.Sum(nil))
	pkg["SHA256"] = fmt.Sprintf("%x", sha256sum.Sum(nil))
	return nil
}
