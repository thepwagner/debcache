package server

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"golang.org/x/net/context"
)

// Logger is custom chi logging middleware for slog.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		t1 := time.Now()
		defer func() {
			level := slog.LevelInfo
			if ww.Status() != http.StatusOK {
				level = slog.LevelWarn
			}

			slog.Log(context.Background(), level, "returned response",
				slog.String("request_id", middleware.GetReqID(r.Context())),
				slog.Group("request", slog.String("method", r.Method), slog.String("path", r.URL.Path)),
				slog.Group("response", slog.Int("status", ww.Status()), slog.Int("bytes", ww.BytesWritten())),
				slog.Int("duration", int(time.Since(t1).Milliseconds())),
			)
		}()

		next.ServeHTTP(ww, r)
	})
}
