package debian

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"

	"github.com/blakesmith/ar"
	"github.com/ulikunitz/xz"
)

func ParagraphFromDeb(in io.Reader) (*Paragraph, error) {
	for reader := ar.NewReader(in); ; {
		// find control.tar.gz or die trying
		hdr, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return nil, fmt.Errorf("reading archive: %w", err)
		}

		var controlIn io.Reader
		switch hdr.Name {
		case "control.tar.gz":
			gzIn, err := gzip.NewReader(reader)
			if err != nil {
				return nil, fmt.Errorf("creating gzip reader: %w", err)
			}
			defer gzIn.Close()
			controlIn = gzIn
		case "control.tar.xz":
			controlIn, err = xz.NewReader(reader)
			if err != nil {
				return nil, fmt.Errorf("creating xz reader: %w", err)
			}
		default:
			continue
		}

		// Find ./control within compressed tarball
		for tarR := tar.NewReader(controlIn); ; {
			hdr, err := tarR.Next()
			if errors.Is(err, io.EOF) {
				break
			} else if err != nil {
				return nil, fmt.Errorf("reading archive: %w", err)
			}
			if hdr.Name != "./control" {
				continue
			}

			graphs, err := ParseControlFile(tarR)
			if err != nil {
				return nil, fmt.Errorf("parsing control file: %w", err)
			}
			if len(graphs) == 1 {
				return &graphs[0], nil
			}
		}
	}
	return nil, nil
}
