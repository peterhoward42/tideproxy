package app

import (
	"testing"
	"time"
)

func TestSynthesiseTidesAPIResponse_nilIncoming(t *testing.T) {
	t.Parallel()
	_, err := SynthesiseTidesAPIResponse(nil, time.Unix(0, 0).UTC())
	if err != errNilIncomingResponse {
		t.Fatalf("error: got %v want %v", err, errNilIncomingResponse)
	}
}

func TestSynthesiseTidesAPIResponse_mapsFieldsAndWindow(t *testing.T) {
	t.Parallel()

	at := time.Date(2026, 3, 21, 15, 30, 0, 0, time.UTC)
	// Window follows overview.md and matches outputWindowDays from output_request.go.
	wantWindowStart := time.Date(2026, 3, 21, 0, 0, 0, 0, time.UTC)
	wantExpiresAt := wantWindowStart.Add(outputWindowDays * 24 * time.Hour)

	tHigh := time.Date(2026, 3, 21, 6, 12, 0, 0, time.UTC)
	in := IncomingResponse{
		Copyright: "fixture attribution",
		Extremes: []IncomingExtreme{
			{Type: "High", Time: tHigh, HeightMetres: 4.81},
			{Type: "Low", Time: tHigh.Add(time.Hour), HeightMetres: 1.2},
		},
	}

	got, err := SynthesiseTidesAPIResponse(&in, at)
	if err != nil {
		t.Fatalf("SynthesiseTidesAPIResponse: %v", err)
	}
	if got.Datum != chartDatum {
		t.Fatalf("Datum: got %q want %q", got.Datum, chartDatum)
	}
	if !got.WindowStart.Equal(wantWindowStart) {
		t.Fatalf("WindowStart: got %v want %v", got.WindowStart, wantWindowStart)
	}
	if !got.ExpiresAt.Equal(wantExpiresAt) {
		t.Fatalf("ExpiresAt: got %v want %v", got.ExpiresAt, wantExpiresAt)
	}
	if got.Attribution != in.Copyright {
		t.Fatalf("Attribution: got %q want %q", got.Attribution, in.Copyright)
	}
	if len(got.Tides) != len(in.Extremes) {
		t.Fatalf("len(Tides): got %d want %d", len(got.Tides), len(in.Extremes))
	}
	for i := range in.Extremes {
		if got.Tides[i].Type != in.Extremes[i].Type {
			t.Fatalf("Tides[%d].Type: got %q want %q", i, got.Tides[i].Type, in.Extremes[i].Type)
		}
		if !got.Tides[i].Time.Equal(in.Extremes[i].Time.UTC()) {
			t.Fatalf("Tides[%d].Time: got %v want %v", i, got.Tides[i].Time, in.Extremes[i].Time.UTC())
		}
		if got.Tides[i].HeightMetres != in.Extremes[i].HeightMetres {
			t.Fatalf("Tides[%d].HeightMetres: got %v want %v", i, got.Tides[i].HeightMetres, in.Extremes[i].HeightMetres)
		}
	}
}

func TestSynthesiseTidesAPIResponse_emptyExtremes(t *testing.T) {
	t.Parallel()

	at := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	in := IncomingResponse{Copyright: "c", Extremes: []IncomingExtreme{}}

	got, err := SynthesiseTidesAPIResponse(&in, at)
	if err != nil {
		t.Fatalf("SynthesiseTidesAPIResponse: %v", err)
	}
	if got.Tides == nil || len(got.Tides) != 0 {
		t.Fatalf("Tides: got %#v want empty non-nil slice", got.Tides)
	}
}
