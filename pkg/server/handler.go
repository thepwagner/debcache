package server

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/thepwagner/debcache/pkg/dynamic"
	"github.com/thepwagner/debcache/pkg/repo"
)

type Handler struct {
	mux *chi.Mux

	repos map[string]repo.Repo
}

func NewHandler() *Handler {
	h := &Handler{
		mux:   chi.NewRouter(),
		repos: map[string]repo.Repo{},
	}
	h.mux.Use(middleware.RequestID)
	h.mux.Use(Logger)
	h.mux.Get("/{repo}/dists/{dist}/InRelease", h.InRelease)

	h.mux.Get("/{repo}/dists/{dist}/{component}/binary-{architecture}/Packages", h.Packages)
	h.mux.Get("/{repo}/dists/{dist}/{component}/binary-{architecture}/Packages{compression:(.[gx]z|)}", h.Packages)
	h.mux.Get("/{repo}/dists/{dist}/{component}/binary-{architecture}/by-hash/{digestAlgo}/{digest}", h.ByHash)

	h.mux.Get("/{repo}/pool/{component}/{p}/{package}/{filename}", h.Pool)

	// FIXME: hacks should come from config
	u, _ := url.Parse("https://deb.debian.org/debian")
	h.repos["debian"] = repo.NewCached(repo.NewUpstream(*u), repo.NewFileCacheStorage("tmp"))
	u, _ = url.Parse("https://deb.debian.org/debian-security")
	h.repos["debian-security"] = repo.NewCached(repo.NewUpstream(*u), repo.NewFileCacheStorage("tmp"))

	repo, err := dynamic.NewRepo()
	if err != nil {
		panic(err)
	}
	h.repos["dynamic"] = repo
	return h
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func (h Handler) InRelease(w http.ResponseWriter, r *http.Request) {
	repoName := chi.URLParam(r, "repo")
	dist := chi.URLParam(r, "dist")
	slog.Info("handling InRelease",
		slog.String("request_id", middleware.GetReqID(r.Context())),
		slog.String("repo", repoName),
		slog.String("dist", dist),
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
	dist := chi.URLParam(r, "dist")
	component := chi.URLParam(r, "component")
	arch := chi.URLParam(r, "architecture")
	compression := repo.ParseCompression(chi.URLParam(r, "compression"))
	slog.Info("handling Packages",
		slog.String("request_id", middleware.GetReqID(r.Context())),
		slog.String("repo", repoName),
		slog.String("dist", dist),
		slog.String("component", component),
		slog.String("arch", arch),
		slog.Any("compression", compression),
	)

	rep, ok := h.repos[repoName]
	if !ok {
		http.NotFound(w, r)
		return
	}
	fmt.Println(r)

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

	dist := chi.URLParam(r, "dist")
	component := chi.URLParam(r, "component")
	arch := chi.URLParam(r, "architecture")
	digest := chi.URLParam(r, "digest")
	slog.Info("handling ByHash",
		slog.String("request_id", middleware.GetReqID(r.Context())),
		slog.String("repo", repoName),
		slog.String("dist", dist),
		slog.String("component", component),
		slog.String("arch", arch),
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
	repo, ok := h.repos[repoName]
	if !ok {
		http.NotFound(w, r)
		return
	}

	component := chi.URLParam(r, "component")
	pkg := chi.URLParam(r, "package")
	filename := chi.URLParam(r, "filename")
	slog.Info("handling Pool",
		slog.String("request_id", middleware.GetReqID(r.Context())),
		slog.String("repo", repoName),
		slog.String("component", component),
		slog.String("package", pkg),
		slog.String("filename", filename),
	)

	b, err := repo.Pool(r.Context(), component, pkg, filename)
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
