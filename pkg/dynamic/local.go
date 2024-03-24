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
	"strings"
	"time"

	"github.com/thepwagner/debcache/pkg/debian"
	"github.com/thepwagner/debcache/pkg/repo"
)

// LocalSource is a package source that reads from a local directory.
type LocalSource struct {
	dir string
}

var _ PackageSource = (*LocalSource)(nil)

type LocalConfig struct {
	Directory string `yaml:"dir"`
}

func NewLocalSource(cfg LocalConfig) *LocalSource {
	return &LocalSource{dir: cfg.Directory}
}

func (s LocalSource) Packages(_ context.Context) (PackageList, time.Time, error) {
	ret := PackageList{}
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

		pkg, err := debian.ParagraphFromDebFile(path)
		if err != nil {
			return err
		} else if pkg == nil {
			slog.Warn("no control file found", "file", path)
			return nil
		}

		if err := s.addFileData(*pkg, path, info); err != nil {
			return err
		}

		arch := repo.Architecture((*pkg)["Architecture"])
		ret.Add("main", arch, *pkg)
		return nil
	})
	if err != nil {
		return nil, time.Time{}, err
	}

	return ret, latest, nil
}

func (s LocalSource) Deb(_ context.Context, filename string) ([]byte, error) {
	filename = strings.TrimPrefix(filename, "main/p/pkg/")
	return os.ReadFile(filepath.Join(s.dir, filename))
}

func (s LocalSource) addFileData(pkg debian.Paragraph, fn string, info fs.FileInfo) error {
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
