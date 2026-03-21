package app

import (
	"net/http"
	"time"
)

// HTTPDoer performs outbound HTTP requests. [*http.Client] satisfies HTTPDoer.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// TimeSource supplies the current instant for synthesising time-bounded upstream
// requests. When nil on [Dependencies], wall clock time is used.
type TimeSource interface {
	Now() time.Time
}

// Dependencies aggregates interfaces to external systems. Add fields only when required.
type Dependencies struct {
	HTTPClient       HTTPDoer
	WorldTidesAPIKey string
	Clock            TimeSource
}

// Application owns HTTP request handling for the proxy API.
type Application struct {
	deps Dependencies
}

// NewApplication constructs an [Application] with explicit dependencies.
func NewApplication(deps Dependencies) *Application {
	return &Application{deps: deps}
}

// ServeHTTP routes incoming requests to the appropriate handler method.
func (a *Application) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet && r.URL.Path == "/v1/tides" {
		a.handleTides(w, r)
		return
	}
	http.NotFound(w, r)
}
