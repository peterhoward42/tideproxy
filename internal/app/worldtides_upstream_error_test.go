package app

import (
	"errors"
	"testing"
)

// invalidAPIKeyUpstreamBody is from a live probe (2026-05): HTTP 400,
// {"status":400,"error":"API key is invalid"}.
const invalidAPIKeyUpstreamBody = `{"status":400,"error":"API key is invalid"}`

func TestParseWorldTidesUpstreamError_invalidAPIKey(t *testing.T) {
	t.Parallel()

	got, err := ParseWorldTidesUpstreamError([]byte(invalidAPIKeyUpstreamBody))
	if err != nil {
		t.Fatalf("ParseWorldTidesUpstreamError: %v", err)
	}
	if got.Status != 400 {
		t.Fatalf("Status = %d, want 400", got.Status)
	}
	if got.Error != "API key is invalid" {
		t.Fatalf("Error = %q, want %q", got.Error, "API key is invalid")
	}
	if got.CreditsExhausted() {
		t.Fatal("CreditsExhausted() = true, want false for invalid API key")
	}
}

func TestWorldTidesUpstreamError_CreditsExhausted(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		error string
		want  bool
	}{
		{name: "not enough credits", error: "Not enough credits", want: true},
		{name: "monthly credits", error: "Monthly API credits exhausted", want: true},
		{name: "credit case insensitive", error: "CREDIT limit reached", want: true},
		{name: "invalid api key", error: "API key is invalid", want: false},
		{name: "no location", error: "No location found", want: false},
		{name: "empty", error: "", want: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := WorldTidesUpstreamError{Status: 400, Error: tt.error}
			if got := e.CreditsExhausted(); got != tt.want {
				t.Fatalf("CreditsExhausted() = %v, want %v for error %q", got, tt.want, tt.error)
			}
		})
	}
}

func TestParseWorldTidesUpstreamError_rejectsMissingFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		payload string
		wantErr error
	}{
		{
			name:    "missing status",
			payload: `{"error":"something"}`,
			wantErr: errWorldTidesUpstreamErrorStatus,
		},
		{
			name:    "missing error",
			payload: `{"status":400}`,
			wantErr: errWorldTidesUpstreamErrorText,
		},
		{
			name:    "empty error",
			payload: `{"status":400,"error":""}`,
			wantErr: errWorldTidesUpstreamErrorText,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := ParseWorldTidesUpstreamError([]byte(tt.payload))
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ParseWorldTidesUpstreamError: got %v want %v", err, tt.wantErr)
			}
		})
	}
}
