package httpapi

import "net/http"

// corsOrigin is the UI dev server origin. It can be overridden with the
// CORS_ORIGIN environment variable; see cmd/server/main.go.
var corsOrigin = "http://localhost:3000"

// withCORS wraps a handler with CORS headers that allow the Next.js dev server.
func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			origin = "*"
		}
		// Restrict to the configured UI origin when it is present.
		if corsOrigin != "" && corsOrigin != "*" && origin != corsOrigin {
			origin = corsOrigin
		}

		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// SetCORSOrigin lets the main package override the default UI origin.
func SetCORSOrigin(origin string) {
	if origin != "" {
		corsOrigin = origin
	}
}
