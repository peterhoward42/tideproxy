package app

import (
	"errors"
	"math"
	"testing"
	"time"
)

func TestSynthesiseOutputRequest_valid(t *testing.T) {
	t.Parallel()

	in := &IncomingRequest{Lat: 51.5, Lon: -0.12}
	got, err := SynthesiseOutputRequest(in)
	if err != nil {
		t.Fatalf("SynthesiseOutputRequest: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil OutputRequest")
	}

	now := time.Now().UTC()
	wantStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	if got.Scheme != "https" || got.Host != worldTidesHTTPSHost || got.Path != worldTidesAPIv3Path {
		t.Fatalf("endpoint: got scheme=%q host=%q path=%q", got.Scheme, got.Host, got.Path)
	}
	if got.Lat != in.Lat || got.Lon != in.Lon {
		t.Fatalf("coordinates: got lat=%v lon=%v want lat=%v lon=%v", got.Lat, got.Lon, in.Lat, in.Lon)
	}
	if got.Datum != chartDatum || got.Units != heightUnitsMeters {
		t.Fatalf("datum/units: got datum=%q units=%q", got.Datum, got.Units)
	}
	if !got.Extremes {
		t.Fatal("expected Extremes true")
	}
	if got.StartUnix != wantStart.Unix() {
		t.Fatalf("StartUnix: got %d want %d", got.StartUnix, wantStart.Unix())
	}
	if got.LengthSeconds != outputWindowSeconds {
		t.Fatalf("LengthSeconds: got %d want %d", got.LengthSeconds, outputWindowSeconds)
	}
}

func TestSynthesiseOutputRequest_nilIncoming(t *testing.T) {
	t.Parallel()

	_, err := SynthesiseOutputRequest(nil)
	if !errors.Is(err, errNilIncomingRequest) {
		t.Fatalf("error: got %v want %v", err, errNilIncomingRequest)
	}
}

func TestSynthesiseOutputRequest_invalidCoordinates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   *IncomingRequest
		want error
	}{
		{
			name: "nan lat",
			in:   &IncomingRequest{Lat: math.NaN(), Lon: 0},
			want: errLatLonNotFinite,
		},
		{
			name: "lat out of range high",
			in:   &IncomingRequest{Lat: 90.0001, Lon: 0},
			want: errLatOutOfRange,
		},
		{
			name: "lon out of range low",
			in:   &IncomingRequest{Lat: 0, Lon: -180.0001},
			want: errLonOutOfRange,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := SynthesiseOutputRequest(tt.in)
			if !errors.Is(err, tt.want) {
				t.Fatalf("error: got %v want %v", err, tt.want)
			}
		})
	}
}
