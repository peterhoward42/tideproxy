package app_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"tideproxy/internal/app"
)

type fakeHTTPDoer struct {
	doFn func(*http.Request) (*http.Response, error)
}

func (f *fakeHTTPDoer) Do(req *http.Request) (*http.Response, error) {
	return f.doFn(req)
}

type fixedClock struct {
	t time.Time
}

func (c fixedClock) Now() time.Time {
	return c.t
}

func mustDecodeAPIError(t *testing.T, body []byte) (code, message string) {
	t.Helper()
	var v struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &v); err != nil {
		t.Fatalf("json.Unmarshal error body: %v", err)
	}
	return v.Error.Code, v.Error.Message
}

// validWorldTidesExtremesJSON returns a minimal WorldTides v3 extremes payload
// that passes [app.ParseIncomingResponse] validation.
func validWorldTidesExtremesJSON(t *testing.T) []byte {
	t.Helper()
	m := map[string]any{
		"status":          200,
		"copyright":       "upstream attribution fixture",
		"requestDatum":    "CD",
		"responseDatum":   "CD",
		"extremes":        []map[string]any{{"dt": int64(1710994320), "height": 4.81, "type": "High"}},
		"responseLat":     51.5,
		"responseLon":     -0.12,
	}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	return b
}

func TestApplication_handleTides_upstreamSuccessReturnsProxyJSON(t *testing.T) {
	t.Parallel()

	at := time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC)
	apiKey := "fixture-api-key"
	upstreamBody := validWorldTidesExtremesJSON(t)

	var captured *http.Request
	fake := &fakeHTTPDoer{
		doFn: func(req *http.Request) (*http.Response, error) {
			captured = req
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/vnd.test+json"}},
				Body:       io.NopCloser(bytes.NewReader(upstreamBody)),
			}, nil
		},
	}

	deps := app.Dependencies{
		HTTPClient:       fake,
		WorldTidesAPIKey: apiKey,
		Clock:            fixedClock{t: at},
	}
	application := app.NewApplication(deps)

	lat, lon := 51.5, -0.12
	q := "/v1/tides?lat=" + strconv.FormatFloat(lat, 'f', -1, 64) + "&lon=" + strconv.FormatFloat(lon, 'f', -1, 64)
	req := httptest.NewRequest(http.MethodGet, q, http.NoBody)
	rec := httptest.NewRecorder()
	application.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d want %d body=%q", rec.Code, http.StatusOK, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type: got %q want application/json; charset=utf-8", got)
	}

	var body struct {
		Tides []struct {
			Type         string  `json:"type"`
			Time         string  `json:"time"`
			HeightMetres float64 `json:"heightMetres"`
		} `json:"tides"`
		Datum       string `json:"datum"`
		WindowStart string `json:"windowStart"`
		ExpiresAt   string `json:"expiresAt"`
		Attribution string `json:"attribution"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response JSON: %v", err)
	}
	if body.Datum != "CD" {
		t.Fatalf("datum: got %q", body.Datum)
	}
	if body.WindowStart != "2026-03-21T00:00:00Z" || body.ExpiresAt != "2026-03-24T00:00:00Z" {
		t.Fatalf("window: windowStart=%q expiresAt=%q", body.WindowStart, body.ExpiresAt)
	}
	if body.Attribution != "upstream attribution fixture" {
		t.Fatalf("attribution: got %q", body.Attribution)
	}
	if len(body.Tides) != 1 || body.Tides[0].Type != "High" || body.Tides[0].HeightMetres != 4.81 {
		t.Fatalf("tides: %#v", body.Tides)
	}
	if body.Tides[0].Time != "2024-03-21T04:12:00Z" {
		t.Fatalf("tide time: got %q", body.Tides[0].Time)
	}

	in := app.IncomingRequest{Lat: lat, Lon: lon}
	out, err := app.SynthesiseOutputRequest(&in, at)
	if err != nil {
		t.Fatalf("SynthesiseOutputRequest: %v", err)
	}
	wantReq, err := app.BuildOutputHTTPRequest(context.Background(), out, apiKey)
	if err != nil {
		t.Fatalf("BuildOutputHTTPRequest: %v", err)
	}
	if captured == nil {
		t.Fatal("expected HTTP client Do to be called")
	}
	if captured.URL.String() != wantReq.URL.String() {
		t.Fatalf("request URL: got %q want %q", captured.URL.String(), wantReq.URL.String())
	}
	if captured.Method != http.MethodGet {
		t.Fatalf("method: got %q", captured.Method)
	}
}

func TestApplication_handleTides_invalidQuery(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		query      string
		wantCode   string
		wantStatus int
	}{
		{
			name:       "missing lat",
			query:      "/v1/tides?lon=0",
			wantCode:   "INVALID_QUERY",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing lon",
			query:      "/v1/tides?lat=0",
			wantCode:   "INVALID_QUERY",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "lat not a number",
			query:      "/v1/tides?lat=x&lon=0",
			wantCode:   "INVALID_QUERY",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "lat out of range",
			query:      "/v1/tides?lat=91&lon=0",
			wantCode:   "INVALID_QUERY",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			at := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
			deps := app.Dependencies{
				HTTPClient:       nil,
				WorldTidesAPIKey: "key",
				Clock:            fixedClock{t: at},
			}
			application := app.NewApplication(deps)

			req := httptest.NewRequest(http.MethodGet, tt.query, http.NoBody)
			rec := httptest.NewRecorder()
			application.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status: got %d want %d body=%q", rec.Code, tt.wantStatus, rec.Body.String())
			}
			code, _ := mustDecodeAPIError(t, rec.Body.Bytes())
			if code != tt.wantCode {
				t.Fatalf("error code: got %q want %q", code, tt.wantCode)
			}
		})
	}
}

func TestApplication_handleTides_upstreamJSONDoesNotValidate(t *testing.T) {
	t.Parallel()

	at := time.Date(2023, 3, 3, 0, 0, 0, 0, time.UTC)
	fake := &fakeHTTPDoer{
		doFn: func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("{}")),
			}, nil
		},
	}
	deps := app.Dependencies{
		HTTPClient:       fake,
		WorldTidesAPIKey: "k",
		Clock:            fixedClock{t: at},
	}
	application := app.NewApplication(deps)

	req := httptest.NewRequest(http.MethodGet, "/v1/tides?lat=0&lon=0", http.NoBody)
	rec := httptest.NewRecorder()
	application.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status: got %d want %d", rec.Code, http.StatusBadGateway)
	}
	code, msg := mustDecodeAPIError(t, rec.Body.Bytes())
	if code != "UPSTREAM_ERROR" || msg != "Failed to retrieve tidal data" {
		t.Fatalf("error: code=%q msg=%q", code, msg)
	}
}

func TestApplication_handleTides_upstreamMalformedJSON(t *testing.T) {
	t.Parallel()

	at := time.Date(2023, 3, 3, 0, 0, 0, 0, time.UTC)
	fake := &fakeHTTPDoer{
		doFn: func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("{")),
			}, nil
		},
	}
	deps := app.Dependencies{
		HTTPClient:       fake,
		WorldTidesAPIKey: "k",
		Clock:            fixedClock{t: at},
	}
	application := app.NewApplication(deps)

	req := httptest.NewRequest(http.MethodGet, "/v1/tides?lat=0&lon=0", http.NoBody)
	rec := httptest.NewRecorder()
	application.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status: got %d want %d", rec.Code, http.StatusBadGateway)
	}
	code, _ := mustDecodeAPIError(t, rec.Body.Bytes())
	if code != "UPSTREAM_ERROR" {
		t.Fatalf("error code: got %q", code)
	}
}

func TestApplication_handleTides_upstreamErrorStatus(t *testing.T) {
	t.Parallel()

	at := time.Date(2021, 5, 5, 0, 0, 0, 0, time.UTC)
	fake := &fakeHTTPDoer{
		doFn: func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusServiceUnavailable,
				Body:       io.NopCloser(strings.NewReader("unavailable")),
			}, nil
		},
	}
	deps := app.Dependencies{
		HTTPClient:       fake,
		WorldTidesAPIKey: "k",
		Clock:            fixedClock{t: at},
	}
	application := app.NewApplication(deps)

	req := httptest.NewRequest(http.MethodGet, "/v1/tides?lat=0&lon=0", http.NoBody)
	rec := httptest.NewRecorder()
	application.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status: got %d want %d", rec.Code, http.StatusBadGateway)
	}
	code, msg := mustDecodeAPIError(t, rec.Body.Bytes())
	if code != "UPSTREAM_ERROR" || msg != "Failed to retrieve tidal data" {
		t.Fatalf("error: code=%q msg=%q", code, msg)
	}
}

func TestApplication_handleTides_upstreamDoError(t *testing.T) {
	t.Parallel()

	at := time.Date(2021, 5, 5, 0, 0, 0, 0, time.UTC)
	fake := &fakeHTTPDoer{
		doFn: func(*http.Request) (*http.Response, error) {
			return nil, io.ErrUnexpectedEOF
		},
	}
	deps := app.Dependencies{
		HTTPClient:       fake,
		WorldTidesAPIKey: "k",
		Clock:            fixedClock{t: at},
	}
	application := app.NewApplication(deps)

	req := httptest.NewRequest(http.MethodGet, "/v1/tides?lat=0&lon=0", http.NoBody)
	rec := httptest.NewRecorder()
	application.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status: got %d want %d", rec.Code, http.StatusBadGateway)
	}
	code, _ := mustDecodeAPIError(t, rec.Body.Bytes())
	if code != "UPSTREAM_ERROR" {
		t.Fatalf("error code: got %q", code)
	}
}

func TestApplication_handleTides_emptyAPIKey(t *testing.T) {
	t.Parallel()

	at := time.Date(2021, 5, 5, 0, 0, 0, 0, time.UTC)
	deps := app.Dependencies{
		HTTPClient:       nil,
		WorldTidesAPIKey: "",
		Clock:            fixedClock{t: at},
	}
	application := app.NewApplication(deps)

	req := httptest.NewRequest(http.MethodGet, "/v1/tides?lat=0&lon=0", http.NoBody)
	rec := httptest.NewRecorder()
	application.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status: got %d want %d", rec.Code, http.StatusInternalServerError)
	}
	code, _ := mustDecodeAPIError(t, rec.Body.Bytes())
	if code != "INTERNAL_ERROR" {
		t.Fatalf("error code: got %q", code)
	}
}

func TestApplication_handleTides_nilHTTPClient(t *testing.T) {
	t.Parallel()

	at := time.Date(2021, 5, 5, 0, 0, 0, 0, time.UTC)
	deps := app.Dependencies{
		HTTPClient:       nil,
		WorldTidesAPIKey: "configured",
		Clock:            fixedClock{t: at},
	}
	application := app.NewApplication(deps)

	req := httptest.NewRequest(http.MethodGet, "/v1/tides?lat=0&lon=0", http.NoBody)
	rec := httptest.NewRecorder()
	application.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status: got %d want %d", rec.Code, http.StatusInternalServerError)
	}
	code, _ := mustDecodeAPIError(t, rec.Body.Bytes())
	if code != "INTERNAL_ERROR" {
		t.Fatalf("error code: got %q", code)
	}
}

func TestApplication_handleTides_defaultContentTypeWhenUpstreamOmits(t *testing.T) {
	t.Parallel()

	at := time.Date(2022, 2, 2, 0, 0, 0, 0, time.UTC)
	body := validWorldTidesExtremesJSON(t)
	fake := &fakeHTTPDoer{
		doFn: func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(body)),
			}, nil
		},
	}
	deps := app.Dependencies{
		HTTPClient:       fake,
		WorldTidesAPIKey: "k",
		Clock:            fixedClock{t: at},
	}
	application := app.NewApplication(deps)

	req := httptest.NewRequest(http.MethodGet, "/v1/tides?lat=1&lon=2", http.NoBody)
	rec := httptest.NewRecorder()
	application.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type: got %q", got)
	}
}
