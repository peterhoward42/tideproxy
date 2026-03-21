package app

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"
	"time"
)

func TestWriteTidesAPIResponse_nilResponse(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	err := WriteTidesAPIResponse(rec, nil)
	if !errors.Is(err, errNilTidesAPIResponse) {
		t.Fatalf("error: got %v want %v", err, errNilTidesAPIResponse)
	}
}

func TestWriteTidesAPIResponse_writesJSON(t *testing.T) {
	t.Parallel()

	at := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	in := IncomingResponse{
		Copyright: "attr",
		Extremes: []IncomingExtreme{
			{Type: "Low", Time: time.Date(2025, 6, 1, 8, 0, 0, 0, time.UTC), HeightMetres: 0.5},
		},
	}
	resp, err := SynthesiseTidesAPIResponse(&in, at)
	if err != nil {
		t.Fatalf("SynthesiseTidesAPIResponse: %v", err)
	}

	rec := httptest.NewRecorder()
	if err := WriteTidesAPIResponse(rec, resp); err != nil {
		t.Fatalf("WriteTidesAPIResponse: %v", err)
	}

	if rec.Code != 200 {
		t.Fatalf("status: got %d want 200", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type: got %q", got)
	}

	var decoded TidesAPIResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("json.Unmarshal body: %v", err)
	}
	if decoded.Datum != resp.Datum || decoded.Attribution != resp.Attribution {
		t.Fatalf("decoded mismatch: %#v vs %#v", decoded, resp)
	}
	if len(decoded.Tides) != len(resp.Tides) {
		t.Fatalf("len(Tides): got %d want %d", len(decoded.Tides), len(resp.Tides))
	}
}
