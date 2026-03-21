package app

import "net/http"

// handleTides serves GET /v1/tides. Behaviour is not implemented yet.
func (a *Application) handleTides(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusNotImplemented)
	_, _ = w.Write([]byte("not implemented"))
}
