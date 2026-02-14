package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"runtime/debug"
)

// Recover перехватывает панику, предотвращает падение всего сервиса.
func Recover(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)

				slog.Error("Panic recovered",
					"error", err,
					"method", r.Method,
					"path", r.URL.Path,
					"stack", string(debug.Stack()),
				)

				errorResp := map[string]interface{}{
					"error":   "internal_server_error",
					"message": "Internal server error",
					"code":    http.StatusInternalServerError,
				}
				json.NewEncoder(w).Encode(errorResp)
			}
		}()
		next.ServeHTTP(w, r)
	}
}
