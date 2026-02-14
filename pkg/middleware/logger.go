package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

type responseWriter struct {
	http.ResponseWriter

	statusCode int
	written    bool
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	if !rw.written {
		rw.statusCode = statusCode
		rw.written = true
	}
	rw.ResponseWriter.WriteHeader(statusCode)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.statusCode = http.StatusOK
		rw.written = true
	}
	return rw.ResponseWriter.Write(b)
}

func Logging(next http.Handler) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
			written:        false,
		}

		next.ServeHTTP(rw, r)

		duration := time.Since(start)

		logLevel := slog.LevelInfo
		if rw.statusCode >= 400 && rw.statusCode < 500 {
			logLevel = slog.LevelWarn
		} else if rw.statusCode >= 500 {
			logLevel = slog.LevelError
		}

		slog.Log(r.Context(), logLevel, "HTTP request completed",
			"method", r.Method,
			"path", r.URL.Path,
			"status_code", rw.statusCode,
			"duration_ms", duration.Milliseconds(),
			"user_agent", r.Header.Get("User-Agent"),
			"remote_addr", r.RemoteAddr,
		)
	})
}
