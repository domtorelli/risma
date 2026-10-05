package httpx

import (
	"log/slog"
	"net/http"
	"time"
)

// Logging registra metodo, percorso e durata di ogni richiesta.
func Logging(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			next.ServeHTTP(w, r)
			duration := time.Since(start)
			logger.Info("request", "method", r.Method, "path", r.URL.Path, "duration", duration)
		})
	}
}

// Recovery trasforma il panic in una risposta 500 e lo registra nei log.
func Recovery(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					logger.Error("panic recovered", "panic", recovered, "method", r.Method, "path", r.URL.Path)
					Error(w, http.StatusInternalServerError, "errore interno del server")
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
