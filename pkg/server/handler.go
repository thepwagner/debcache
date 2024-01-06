package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/thepwagner/debcache/pkg/debian"
	"github.com/thepwagner/debcache/pkg/repo"
)

type Handler struct {
	mux *chi.Mux

	repos map[string]repo.Repo
}

func NewHandler(ctx context.Context, cfg *Config) (*Handler, error) {
	h := &Handler{
		mux:   chi.NewRouter(),
		repos: map[string]repo.Repo{},
	}
	h.mux.Use(middleware.RequestID)
	h.mux.Use(middleware.RealIP)
	h.mux.Use(Logger)
	h.mux.Get("/{repo}/repo.source", h.RepoSource)

	h.mux.Get("/{repo}/dists/{dist}/InRelease", h.InRelease)

	h.mux.Get("/{repo}/dists/{dist}/{component}/binary-{architecture}/Packages", h.Packages)
	h.mux.Get("/{repo}/dists/{dist}/{component}/binary-{architecture}/Packages{compression:(.[gx]z|)}", h.Packages)
	h.mux.Get("/{repo}/dists/{dist}/{component}/binary-{architecture}/by-hash/{digestAlgo}/{digest}", h.ByHash)

	h.mux.Get("/{repo}/dists/{dist}/{component}/i18n/Translation-{lang:[^.]+}", h.Translations)
	h.mux.Get("/{repo}/dists/{dist}/{component}/i18n/Translation-{lang:[^.]+}.{compression}", h.Translations)
	h.mux.Get("/{repo}/dists/{dist}/{component}/i18n/by-hash/{digestAlgo}/{digest}", h.ByHash)

	h.mux.Get("/{repo}/pool/{component}/{p}/{package}/*", h.Pool)

	for name, cfg := range cfg.Repos {
		repo, err := BuildRepo(ctx, name, cfg)
		if err != nil {
			return nil, fmt.Errorf("error building repo %q: %w", name, err)
		}
		h.repos[name] = repo
	}

	return h, nil
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func (h Handler) RepoSource(w http.ResponseWriter, r *http.Request) {
	repoName := chi.URLParam(r, "repo")
	slog.Info("handling Key",
		slog.String("request_id", middleware.GetReqID(r.Context())),
		slog.String("repo", repoName),
	)

	repo, ok := h.repos[repoName]
	if !ok {
		http.NotFound(w, r)
		return
	}

	signedBy, err := repo.SigningKeyPEM()
	if err != nil {
		slog.Error("repo.SigningKeyPEM", slog.String("error", err.Error()))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(signedBy) == 0 {
		signedBy = []byte("/usr/share/keyrings/debian-archive-keyring.gpg")
	}

	r.URL.Scheme = "http"
	r.URL.Host = r.Host
	r.URL.Path = ""

	suites := "bookworm"
	if strings.Contains(repoName, "-security") {
		suites = "bookworm-security"
	}

	repoGraph := debian.Paragraph{
		"Types":      "deb",
		"URIs":       r.URL.JoinPath(repoName).String(),
		"Suites":     suites,
		"Components": "main",
		"Signed-By":  string(signedBy),
	}

	_ = debian.WriteControlFile(w, repoGraph)
}

func (h Handler) InRelease(w http.ResponseWriter, r *http.Request) {
	repoName := chi.URLParam(r, "repo")
	dist := repo.Distribution(chi.URLParam(r, "dist"))
	slog.Info("handling InRelease",
		slog.String("request_id", middleware.GetReqID(r.Context())),
		slog.String("repo", repoName),
		slog.Any("dist", dist),
	)

	repo, ok := h.repos[repoName]
	if !ok {
		http.NotFound(w, r)
		return
	}

	res, err := repo.InRelease(r.Context(), dist)
	if err != nil {
		slog.Error("repo.InRelease", slog.String("error", err.Error()))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(res) == 0 {
		http.NotFound(w, r)
		return
	}

	_, _ = w.Write(res)
}

func (h Handler) Packages(w http.ResponseWriter, r *http.Request) {
	repoName := chi.URLParam(r, "repo")
	dist := repo.Distribution(chi.URLParam(r, "dist"))
	component := repo.Component(chi.URLParam(r, "component"))
	arch := repo.Architecture(chi.URLParam(r, "architecture"))
	compression := repo.ParseCompression(chi.URLParam(r, "compression"))
	slog.Info("handling Packages",
		slog.String("request_id", middleware.GetReqID(r.Context())),
		slog.String("repo", repoName),
		slog.Any("dist", dist),
		slog.Any("component", component),
		slog.Any("arch", arch),
		slog.Any("compression", compression),
	)

	rep, ok := h.repos[repoName]
	if !ok {
		http.NotFound(w, r)
		return
	}

	res, err := rep.Packages(r.Context(), dist, component, arch, compression)
	if err != nil {
		slog.Error("repo.Packages", slog.String("error", err.Error()))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(res) == 0 {
		http.NotFound(w, r)
		return
	}

	_, _ = w.Write(res)
}

func (h Handler) ByHash(w http.ResponseWriter, r *http.Request) {
	repoName := chi.URLParam(r, "repo")
	dist := repo.Distribution(chi.URLParam(r, "dist"))
	component := repo.Component(chi.URLParam(r, "component"))
	arch := repo.Architecture(chi.URLParam(r, "architecture"))

	repo, ok := h.repos[repoName]
	if !ok {
		http.NotFound(w, r)
		return
	}
	digestAlgo := chi.URLParam(r, "digestAlgo")
	if digestAlgo != "SHA256" {
		http.NotFound(w, r)
		return
	}

	digest := chi.URLParam(r, "digest")
	slog.Info("handling ByHash",
		slog.String("request_id", middleware.GetReqID(r.Context())),
		slog.String("repo", repoName),
		slog.Any("dist", dist),
		slog.Any("component", component),
		slog.Any("arch", arch),
		slog.String("digest", digest),
	)

	res, err := repo.ByHash(r.Context(), dist, component, arch, digest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(res) == 0 {
		http.NotFound(w, r)
		return
	}

	_, _ = w.Write(res)
}

func (h Handler) Pool(w http.ResponseWriter, r *http.Request) {
	repoName := chi.URLParam(r, "repo")
	component := repo.Component(chi.URLParam(r, "component"))
	rep, ok := h.repos[repoName]
	if !ok {
		http.NotFound(w, r)
		return
	}

	pkg := chi.URLParam(r, "package")
	filename := chi.URLParam(r, "*")
	if len(pkg) == 1 {
		component = repo.Component(fmt.Sprintf("%s/%s", component, chi.URLParam(r, "p")))
		split := strings.SplitN(filename, "/", 2)
		pkg = split[0]
		filename = split[1]
	}

	slog.Info("handling Pool",
		slog.String("request_id", middleware.GetReqID(r.Context())),
		slog.String("repo", repoName),
		slog.Any("component", component),
		slog.String("package", pkg),
		slog.String("filename", filename),
	)

	b, err := rep.Pool(r.Context(), component, pkg, filename)
	if err != nil {
		slog.Error("repo.Pool", slog.String("error", err.Error()))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(b) == 0 {
		http.NotFound(w, r)
		return
	}

	_, _ = w.Write(b)
}

func (h Handler) Translations(w http.ResponseWriter, r *http.Request) {
	repoName := chi.URLParam(r, "repo")
	dist := repo.Distribution(chi.URLParam(r, "dist"))
	component := repo.Component(chi.URLParam(r, "component"))
	lang := repo.Language(chi.URLParam(r, "lang"))
	compression := repo.ParseCompression(chi.URLParam(r, "compression"))
	slog.Info("handling Translations",
		slog.String("request_id", middleware.GetReqID(r.Context())),
		slog.String("repo", repoName),
		slog.Any("dist", dist),
		slog.Any("component", component),
		slog.Any("lang", lang),
		slog.Any("compression", compression),
	)

	repo, ok := h.repos[repoName]
	if !ok {
		http.NotFound(w, r)
		return
	}

	res, err := repo.Translations(r.Context(), dist, component, lang, compression)
	if err != nil {
		slog.Error("repo.Translations", slog.String("error", err.Error()))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(res) == 0 {
		http.NotFound(w, r)
		return
	}

	_, _ = w.Write(res)
}
