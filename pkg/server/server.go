package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
)

func Run(ctx context.Context) error {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	slog.SetDefault(logger)

	h := NewHandler()

	srv := &http.Server{
		Addr:    ":8080",
		Handler: h,
	}

	go func() {
		<-ctx.Done()
		slog.InfoContext(ctx, "shutting down")
		_ = srv.Shutdown(context.Background())
	}()

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
