package app

import (
	"errors"
	"net/http"
	"time"
)

var (
	// ErrNilHTTPClient is returned by [NewDependencies] when httpClient is nil.
	ErrNilHTTPClient = errors.New("app: HTTPClient is required")
	// ErrEmptyWorldTidesAPIKey is returned by [NewDependencies] when worldTidesAPIKey is empty.
	ErrEmptyWorldTidesAPIKey = errors.New("app: WorldTidesAPIKey is required")
	// ErrNilClock is returned by [NewDependencies] when clock is nil.
	ErrNilClock = errors.New("app: Clock is required")
)

// HTTPDoer performs outbound HTTP requests. [*http.Client] satisfies HTTPDoer.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// TimeSource supplies the current instant for synthesising time-bounded upstream
// requests.
type TimeSource interface {
	Now() time.Time
}

// WallClock implements [TimeSource] using the system wall clock.
type WallClock struct{}

func (WallClock) Now() time.Time {
	return time.Now()
}

// Dependencies aggregates interfaces to external systems. Add fields only when required.
type Dependencies struct {
	HTTPClient       HTTPDoer
	WorldTidesAPIKey string
	Clock            TimeSource
}

// NewDependencies returns [Dependencies] with all fields set, or an error if httpClient or
// clock is nil or worldTidesAPIKey is empty.
func NewDependencies(httpClient HTTPDoer, worldTidesAPIKey string, clock TimeSource) (Dependencies, error) {
	if httpClient == nil {
		return Dependencies{}, ErrNilHTTPClient
	}
	if worldTidesAPIKey == "" {
		return Dependencies{}, ErrEmptyWorldTidesAPIKey
	}
	if clock == nil {
		return Dependencies{}, ErrNilClock
	}
	return Dependencies{
		HTTPClient:       httpClient,
		WorldTidesAPIKey: worldTidesAPIKey,
		Clock:            clock,
	}, nil
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
