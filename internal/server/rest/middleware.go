package rest

import (
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// corsMiddleware adds CORS headers to every response based on the resource's
// CORSConfig. Handles preflight OPTIONS requests transparently.
func corsMiddleware(origins, methods, headers []string, enabled bool) func(http.Handler) http.Handler {
	originsStr := strings.Join(origins, ", ")
	methodsStr := strings.Join(methods, ", ")
	headersStr := strings.Join(headers, ", ")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !enabled {
				next.ServeHTTP(w, r)
				return
			}

			// Allow-Origin: match against configured list or wildcard.
			origin := r.Header.Get("Origin")
			allowed := false
			for _, o := range origins {
				if o == "*" || o == origin {
					allowed = true
					break
				}
			}
			if allowed {
				if originsStr == "*" {
					w.Header().Set("Access-Control-Allow-Origin", "*")
				} else {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Add("Vary", "Origin")
				}
			}

			w.Header().Set("Access-Control-Allow-Methods", methodsStr)
			w.Header().Set("Access-Control-Allow-Headers", headersStr)

			// Handle preflight.
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// loggingMiddleware logs every HTTP request using zerolog.
// Format matches DatRi's existing structured logging conventions.
func loggingMiddleware(log zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := &wrappedWriter{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(ww, r)

			log.Info().
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Str("query", r.URL.RawQuery).
				Int("status", ww.status).
				Dur("latency_ms", time.Since(start)).
				Str("remote_addr", r.RemoteAddr).
				Msg("request")
		})
	}
}

// wrappedWriter captures the HTTP status code written by handlers so the
// logging middleware can record it after the fact.
type wrappedWriter struct {
	http.ResponseWriter
	status int
}

func (ww *wrappedWriter) WriteHeader(code int) {
	ww.status = code
	ww.ResponseWriter.WriteHeader(code)
}
