package repo

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
)

// Upstream is a remote repository.
type Upstream struct {
	baseURL url.URL
	client  *http.Client
}

type UpstreamConfig struct {
	URL    string `yaml:"url"`
	Verify bool   `yaml:"verify"`
}

var _ Repo = (*Upstream)(nil)

func UpstreamFromConfig(cfg UpstreamConfig) (*Upstream, error) {
	u, err := url.Parse(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("error parsing upstream URL: %w", err)
	}
	slog.Debug("upstream repo", slog.String("url", u.String()))
	return NewUpstream(*u), nil
}

func NewUpstream(baseURL url.URL) *Upstream {
	return &Upstream{
		baseURL: baseURL,
		client:  http.DefaultClient,
	}
}

func (u Upstream) InRelease(ctx context.Context, dist Distribution) ([]byte, error) {
	return u.get(ctx, "dists", dist.String(), "InRelease")
}

func (u Upstream) Packages(ctx context.Context, dist Distribution, component Component, arch Architecture, compression Compression) ([]byte, error) {
	return u.get(ctx, "dists", dist.String(), component.String(), fmt.Sprintf("binary-%s", arch), "Packages"+compression.Extension())
}

func (u Upstream) ByHash(ctx context.Context, dist Distribution, component Component, arch Architecture, digest string) ([]byte, error) {
	return u.get(ctx, "dists", dist.String(), component.String(), fmt.Sprintf("binary-%s", arch), "by-hash", "SHA256", digest)
}

func (u Upstream) Pool(ctx context.Context, component Component, pkg, filename string) ([]byte, error) {
	prefix := string(pkg[0])
	if strings.HasPrefix(pkg, "lib") {
		prefix = pkg[:4]
	}
	return u.get(ctx, "pool", component.String(), prefix, pkg, filename)
}

func (u Upstream) get(ctx context.Context, path ...string) ([]byte, error) {
	reqURL := u.baseURL.JoinPath(path...).String()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("User-Agent", "debcache/1.0")

	resp, err := u.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("performing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status from upstream %s: %s", reqURL, resp.Status)
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, resp.Body); err != nil {
		return nil, fmt.Errorf("upstream read error: %w", err)
	}
	return buf.Bytes(), nil
}
