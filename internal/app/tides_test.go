package app_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"tideproxy/internal/app"
)

func TestApplication_handleTides_placeholderStatus(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/tides?lat=0&lon=0", http.NoBody)
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()

	deps := app.Dependencies{}
	application := app.NewApplication(deps)
	application.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("status: got %d want %d", rec.Code, http.StatusNotImplemented)
	}
}
