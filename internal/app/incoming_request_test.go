package app

import (
	"encoding/json"
	"errors"
	"io"
	"slices"
	"testing"
)

func mustMarshalJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	return b
}

func mustMarshalIndentJSON(t *testing.T, v any, prefix, indent string) []byte {
	t.Helper()
	b, err := json.MarshalIndent(v, prefix, indent)
	if err != nil {
		t.Fatalf("json.MarshalIndent: %v", err)
	}
	return b
}

func TestUnmarshalIncomingRequest_valid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		payload func(*testing.T) []byte
		want    IncomingRequest
	}{
		{
			name: "typical coordinates",
			payload: func(t *testing.T) []byte {
				return mustMarshalJSON(t, map[string]float64{"lat": 51.5, "lon": -0.12})
			},
			want: IncomingRequest{Lat: 51.5, Lon: -0.12},
		},
		{
			name: "equator prime meridian",
			payload: func(t *testing.T) []byte {
				return mustMarshalJSON(t, map[string]float64{"lat": 0, "lon": 0})
			},
			want: IncomingRequest{Lat: 0, Lon: 0},
		},
		{
			name: "lat bounds min max",
			payload: func(t *testing.T) []byte {
				return mustMarshalJSON(t, map[string]float64{"lat": -90, "lon": 0})
			},
			want: IncomingRequest{Lat: -90, Lon: 0},
		},
		{
			name: "lat max",
			payload: func(t *testing.T) []byte {
				return mustMarshalJSON(t, map[string]float64{"lat": 90, "lon": 0})
			},
			want: IncomingRequest{Lat: 90, Lon: 0},
		},
		{
			name: "lon bounds min max",
			payload: func(t *testing.T) []byte {
				return mustMarshalJSON(t, map[string]float64{"lat": 0, "lon": -180})
			},
			want: IncomingRequest{Lat: 0, Lon: -180},
		},
		{
			name: "lon max",
			payload: func(t *testing.T) []byte {
				return mustMarshalJSON(t, map[string]float64{"lat": 0, "lon": 180})
			},
			want: IncomingRequest{Lat: 0, Lon: 180},
		},
		{
			name: "extra whitespace allowed between values by decoder",
			payload: func(t *testing.T) []byte {
				return mustMarshalIndentJSON(t, map[string]float64{"lat": 1, "lon": 2}, "", "  ")
			},
			want: IncomingRequest{Lat: 1, Lon: 2},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := UnmarshalIncomingRequest(tt.payload(t))
			if err != nil {
				t.Fatalf("UnmarshalIncomingRequest: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %+v want %+v", got, tt.want)
			}
		})
	}
}

func TestUnmarshalIncomingRequest_validationErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		payload func(*testing.T) []byte
		wantErr error
	}{
		{
			name: "missing lat",
			payload: func(t *testing.T) []byte {
				return mustMarshalJSON(t, map[string]float64{"lon": 0})
			},
			wantErr: errMissingLat,
		},
		{
			name: "missing lon",
			payload: func(t *testing.T) []byte {
				return mustMarshalJSON(t, map[string]float64{"lat": 0})
			},
			wantErr: errMissingLon,
		},
		{
			name: "lat null",
			payload: func(t *testing.T) []byte {
				return mustMarshalJSON(t, map[string]any{"lat": nil, "lon": float64(0)})
			},
			wantErr: errMissingLat,
		},
		{
			name: "lon null",
			payload: func(t *testing.T) []byte {
				return mustMarshalJSON(t, map[string]any{"lat": float64(0), "lon": nil})
			},
			wantErr: errMissingLon,
		},
		{
			name: "lat too low",
			payload: func(t *testing.T) []byte {
				return mustMarshalJSON(t, map[string]float64{"lat": -90.0001, "lon": 0})
			},
			wantErr: errLatOutOfRange,
		},
		{
			name: "lat too high",
			payload: func(t *testing.T) []byte {
				return mustMarshalJSON(t, map[string]float64{"lat": 90.0001, "lon": 0})
			},
			wantErr: errLatOutOfRange,
		},
		{
			name: "lon too low",
			payload: func(t *testing.T) []byte {
				return mustMarshalJSON(t, map[string]float64{"lat": 0, "lon": -180.0001})
			},
			wantErr: errLonOutOfRange,
		},
		{
			name: "lon too high",
			payload: func(t *testing.T) []byte {
				return mustMarshalJSON(t, map[string]float64{"lat": 0, "lon": 180.0001})
			},
			wantErr: errLonOutOfRange,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := UnmarshalIncomingRequest(tt.payload(t))
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error: got %v want %v", err, tt.wantErr)
			}
		})
	}
}

func TestUnmarshalIncomingRequest_jsonErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		payload func(*testing.T) []byte
	}{
		{
			name: "invalid syntax",
			payload: func(*testing.T) []byte {
				return []byte{'{'}
			},
		},
		{
			name: "wrong lat type",
			payload: func(t *testing.T) []byte {
				return mustMarshalJSON(t, map[string]any{"lat": "51", "lon": float64(0)})
			},
		},
		{
			name: "unknown field",
			payload: func(t *testing.T) []byte {
				return mustMarshalJSON(t, map[string]any{"lat": float64(0), "lon": float64(0), "extra": float64(1)})
			},
		},
		{
			name: "trailing second value",
			payload: func(t *testing.T) []byte {
				first := mustMarshalJSON(t, map[string]float64{"lat": 0, "lon": 0})
				second := mustMarshalJSON(t, map[string]any{})
				return slices.Concat(first, second)
			},
		},
		{
			name: "number overflow rejected by json",
			payload: func(t *testing.T) []byte {
				return mustMarshalJSON(t, map[string]any{"lat": json.Number("1e400"), "lon": float64(0)})
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := UnmarshalIncomingRequest(tt.payload(t))
			if err == nil {
				t.Fatal("expected error")
			}
			if errors.Is(err, errMissingLat) || errors.Is(err, errMissingLon) ||
				errors.Is(err, errLatOutOfRange) || errors.Is(err, errLonOutOfRange) ||
				errors.Is(err, errLatLonNotFinite) {
				t.Fatalf("unexpected validation error: %v", err)
			}
		})
	}
}

func TestUnmarshalIncomingRequest_trailingGarbage(t *testing.T) {
	t.Parallel()
	valid := mustMarshalJSON(t, map[string]float64{"lat": 0, "lon": 0})
	payload := slices.Concat(valid, []byte(" garbage"))
	_, err := UnmarshalIncomingRequest(payload)
	if err == nil {
		t.Fatal("expected error")
	}
	// Non-JSON trailing text is rejected by the decoder as a syntax error.
	var syn *json.SyntaxError
	if !errors.As(err, &syn) {
		t.Fatalf("expected json.SyntaxError, got %T %v", err, err)
	}
}

func TestUnmarshalIncomingRequest_emptyBody(t *testing.T) {
	t.Parallel()
	_, err := UnmarshalIncomingRequest([]byte{})
	if !errors.Is(err, io.EOF) {
		t.Fatalf("expected io.EOF, got %T %v", err, err)
	}
}
