package app

import "net/http"

// WithCORS wraps handler with headers and preflight handling so browser clients on
// other origins can call GET /v1/tides. Credentials are not supported (Allow-Origin is *).
func WithCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		if r.Method == http.MethodOptions && r.URL.Path == "/v1/tides" {
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			if reqH := r.Header.Get("Access-Control-Request-Headers"); reqH != "" {
				w.Header().Set("Access-Control-Allow-Headers", reqH)
			} else {
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Accept")
			}
			w.Header().Set("Access-Control-Max-Age", "86400")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
