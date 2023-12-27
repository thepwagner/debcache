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

	handler, err := NewHandler(ctx, cfg)
	if err != nil {
		return err
	}
	return runHandler(ctx, cfg.Addr, handler)
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
