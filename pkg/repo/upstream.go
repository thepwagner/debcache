package repo

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// Upstream is a remote repository.
type Upstream struct {
	baseURL url.URL
	client  *http.Client
}

var _ Repo = (*Upstream)(nil)

func NewUpstream(baseURL url.URL) *Upstream {
	return &Upstream{
		baseURL: baseURL,
		client:  http.DefaultClient,
	}
}

func (u Upstream) InRelease(ctx context.Context, dist string) ([]byte, error) {
	return u.get(ctx, "dists", dist, "InRelease")
}

func (u Upstream) ByHash(ctx context.Context, dist, component, arch, digest string) ([]byte, error) {
	return u.get(ctx, "dists", dist, component, fmt.Sprintf("binary-%s", arch), "by-hash", "SHA256", digest)
}

func (u Upstream) Pool(ctx context.Context, component, pkg, filename string) ([]byte, error) {
	return u.get(ctx, "pool", component, string(pkg[0]), pkg, filename)
}

func (u Upstream) get(ctx context.Context, path ...string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.baseURL.JoinPath(path...).String(), nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := u.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upstream proxy: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("upstream proxy status: %s", resp.Status)
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, resp.Body); err != nil {
		return nil, fmt.Errorf("upstream proxy read: %w", err)
	}
	return buf.Bytes(), nil
}
