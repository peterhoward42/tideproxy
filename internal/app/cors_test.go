package app_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"tideproxy/internal/app"
)

func TestWithCORS_GET_v1_tides_setsAllowOrigin(t *testing.T) {
	t.Parallel()

	at := time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC)
	upstreamBody := validWorldTidesExtremesJSON(t)
	fake := &fakeHTTPDoer{
		doFn: func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(bytes.NewReader(upstreamBody)),
			}, nil
		},
	}
	deps := mustNewDeps(t, fake, "key", fixedClock{t: at})
	h := app.WithCORS(app.NewApplication(deps))

	lat, lon := 51.5, -0.12
	q := "/v1/tides?lat=" + strconv.FormatFloat(lat, 'f', -1, 64) + "&lon=" + strconv.FormatFloat(lon, 'f', -1, 64)
	req := httptest.NewRequest(http.MethodGet, q, http.NoBody)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("Access-Control-Allow-Origin: got %q want *", got)
	}
}

func TestWithCORS_OPTIONS_v1_tides_preflight(t *testing.T) {
	t.Parallel()

	h := app.WithCORS(http.NotFoundHandler())
	req := httptest.NewRequest(http.MethodOptions, "/v1/tides", http.NoBody)
	req.Header.Set("Access-Control-Request-Method", "GET")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status: got %d want %d", rec.Code, http.StatusNoContent)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("Access-Control-Allow-Origin: got %q want *", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got != "GET, OPTIONS" {
		t.Fatalf("Access-Control-Allow-Methods: got %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); got != "Authorization, Content-Type, Accept" {
		t.Fatalf("Access-Control-Allow-Headers: got %q", got)
	}
}

func TestWithCORS_OPTIONS_v1_tides_echoesRequestHeaders(t *testing.T) {
	t.Parallel()

	h := app.WithCORS(http.NotFoundHandler())
	req := httptest.NewRequest(http.MethodOptions, "/v1/tides", http.NoBody)
	req.Header.Set("Access-Control-Request-Headers", "X-Custom, Authorization")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Headers"); got != "X-Custom, Authorization" {
		t.Fatalf("Access-Control-Allow-Headers: got %q", got)
	}
}

func TestWithCORS_notFoundStillAllowsOrigin(t *testing.T) {
	t.Parallel()

	h := app.WithCORS(http.NotFoundHandler())
	req := httptest.NewRequest(http.MethodGet, "/nope", http.NoBody)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status: got %d want %d", rec.Code, http.StatusNotFound)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("Access-Control-Allow-Origin: got %q want *", got)
	}
}
