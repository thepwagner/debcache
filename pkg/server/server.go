package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/lmittmann/tint"
	"gopkg.in/yaml.v3"
)

func Run(ctx context.Context) error {
	logger := slog.New(tint.NewHandler(os.Stderr, &tint.Options{
		Level:      slog.LevelDebug,
		TimeFormat: time.RFC3339,
	}))
	slog.SetDefault(logger)

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	handler, err := NewHandler(cfg)
	if err != nil {
		return err
	}
	return runHandler(ctx, cfg.Server.Addr, handler)
}

func loadConfig() (*Config, error) {
	f, err := os.Open("debcache.yml")
	if err != nil {
		return nil, fmt.Errorf("error opening config: %w", err)
	}
	defer f.Close()

	var cfg Config
	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("error decoding config: %w", err)
	}

	if cfg.Server.Addr == "" {
		cfg.Server.Addr = ":8080"
	}

	return &cfg, nil
}

func runHandler(ctx context.Context, addr string, handler http.Handler) error {
	srv := &http.Server{
		Addr:    addr,
		Handler: handler,
	}
	go func() {
		<-ctx.Done()
		shutdownCtx := context.Background()
		slog.InfoContext(shutdownCtx, "shutting down")
		_ = srv.Shutdown(shutdownCtx)
	}()

	slog.Info("starting server", slog.String("addr", addr))
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("error binding server: %w", err)
	}
	return nil
}
