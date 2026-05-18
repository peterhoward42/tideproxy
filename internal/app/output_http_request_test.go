package app

import (
	"context"
	"errors"
	"net/http"
	"testing"
)

func TestBuildOutputHTTPRequest_valid(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	out := &OutputRequest{
		Scheme:        "https",
		Host:          worldTidesHTTPSHost,
		Path:          worldTidesAPIv3Path,
		Lat:           51.5,
		Lon:           -0.12,
		Datum:         chartDatum,
		Units:         heightUnitsMeters,
		Extremes:      true,
		StartUnix:     1700000000,
		LengthSeconds: 345600,
	}

	req, err := BuildOutputHTTPRequest(ctx, out, "test-api-key")
	if err != nil {
		t.Fatalf("BuildOutputHTTPRequest: %v", err)
	}
	if req.Method != http.MethodGet {
		t.Fatalf("method: got %q want GET", req.Method)
	}
	if req.Body != nil {
		t.Fatal("expected nil body")
	}
	if req.Context() != ctx {
		t.Fatal("expected request context to match input context")
	}

	q := req.URL.Query()
	if got := q.Get("key"); got != "test-api-key" {
		t.Fatalf("key: got %q", got)
	}
	if got := q.Get("lat"); got != "51.5" {
		t.Fatalf("lat: got %q", got)
	}
	if got := q.Get("lon"); got != "-0.12" {
		t.Fatalf("lon: got %q", got)
	}
	if got := q.Get("datum"); got != chartDatum {
		t.Fatalf("datum: got %q", got)
	}
	if got := q.Get("units"); got != heightUnitsMeters {
		t.Fatalf("units: got %q", got)
	}
	if _, ok := q["extremes"]; !ok {
		t.Fatal("expected extremes query parameter present")
	}
	if got := q.Get("start"); got != "1700000000" {
		t.Fatalf("start: got %q", got)
	}
	if got := q.Get("length"); got != "345600" {
		t.Fatalf("length: got %q", got)
	}

	want := "https://www.worldtides.info/api/v3?" + q.Encode()
	if req.URL.String() != want {
		t.Fatalf("full URL: got %q want %q", req.URL.String(), want)
	}
}

func TestBuildOutputHTTPRequest_extremesOmittedWhenFalse(t *testing.T) {
	t.Parallel()

	out := &OutputRequest{
		Scheme:        "https",
		Host:          worldTidesHTTPSHost,
		Path:          worldTidesAPIv3Path,
		Lat:           0,
		Lon:           0,
		Datum:         chartDatum,
		Units:         heightUnitsMeters,
		Extremes:      false,
		StartUnix:     1,
		LengthSeconds: 60,
	}

	req, err := BuildOutputHTTPRequest(context.Background(), out, "k")
	if err != nil {
		t.Fatalf("BuildOutputHTTPRequest: %v", err)
	}
	if _, ok := req.URL.Query()["extremes"]; ok {
		t.Fatal("did not expect extremes parameter when Extremes is false")
	}
}

func TestBuildOutputHTTPRequest_nilOutputRequest(t *testing.T) {
	t.Parallel()

	_, err := BuildOutputHTTPRequest(context.Background(), nil, "key")
	if !errors.Is(err, errNilOutputRequest) {
		t.Fatalf("error: got %v want %v", err, errNilOutputRequest)
	}
}

func TestBuildOutputHTTPRequest_emptyAPIKey(t *testing.T) {
	t.Parallel()

	out := &OutputRequest{
		Scheme: "https", Host: "h", Path: "/p",
		Lat: 1, Lon: 2, Datum: "CD", Units: "meters",
		Extremes: true, StartUnix: 1, LengthSeconds: 2,
	}
	_, err := BuildOutputHTTPRequest(context.Background(), out, "")
	if !errors.Is(err, errEmptyAPIKey) {
		t.Fatalf("error: got %v want %v", err, errEmptyAPIKey)
	}
}

func TestBuildOutputHTTPRequest_incompleteEndpoint(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		out  *OutputRequest
	}{
		{
			name: "empty scheme",
			out: &OutputRequest{
				Scheme: "", Host: "h", Path: "/p",
				Lat: 1, Lon: 2, Datum: "CD", Units: "meters",
				Extremes: true, StartUnix: 1, LengthSeconds: 2,
			},
		},
		{
			name: "empty host",
			out: &OutputRequest{
				Scheme: "https", Host: "", Path: "/p",
				Lat: 1, Lon: 2, Datum: "CD", Units: "meters",
				Extremes: true, StartUnix: 1, LengthSeconds: 2,
			},
		},
		{
			name: "empty path",
			out: &OutputRequest{
				Scheme: "https", Host: "h", Path: "",
				Lat: 1, Lon: 2, Datum: "CD", Units: "meters",
				Extremes: true, StartUnix: 1, LengthSeconds: 2,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := BuildOutputHTTPRequest(context.Background(), tt.out, "key")
			if !errors.Is(err, errIncompleteOutputEndpoint) {
				t.Fatalf("error: got %v want %v", err, errIncompleteOutputEndpoint)
			}
		})
	}
}
