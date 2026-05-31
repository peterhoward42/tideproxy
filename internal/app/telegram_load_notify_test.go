package app

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestIsLooeDefaultLoad(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		lat  float64
		lon  float64
		want bool
	}{
		{name: "east looe", lat: 50.3545669, lon: -4.4517948, want: true},
		{name: "west looe", lat: 50.3531094, lon: -4.4583049, want: true},
		{name: "looe town", lat: 50.3518739, lon: -4.4527880, want: true},
		{name: "south of envelope", lat: 50.350, lon: -4.452, want: false},
		{name: "west of envelope", lat: 50.353, lon: -4.460, want: false},
		{name: "north of envelope", lat: 50.356, lon: -4.452, want: false},
		{name: "east of envelope", lat: 50.353, lon: -4.450, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := isLooeDefaultLoad(tt.lat, tt.lon); got != tt.want {
				t.Fatalf("isLooeDefaultLoad(%v, %v) = %v want %v", tt.lat, tt.lon, got, tt.want)
			}
		})
	}
}

func TestLoadTelegramAlertForLatLon(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		lat  float64
		lon  float64
		want string
	}{
		{name: "looe town", lat: 50.3518739, lon: -4.4527880, want: telegramLoadDefaultAlert},
		{name: "custom location", lat: 51.5, lon: -0.12, want: telegramLoadCustomAlert},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := loadTelegramAlertForLatLon(tt.lat, tt.lon); got != tt.want {
				t.Fatalf("loadTelegramAlertForLatLon(%v, %v) = %q want %q", tt.lat, tt.lon, got, tt.want)
			}
		})
	}
}

func TestApplication_notifyTelegramLoadSuccess(t *testing.T) {
	t.Parallel()

	if !telegramLoadNotificationsEnabled {
		t.Skip("telegramLoadNotificationsEnabled is false")
	}

	tests := []struct {
		name          string
		lat           float64
		lon           float64
		telegramErr   error
		wantText      string
		wantSendCalls int
	}{
		{
			name:          "default load",
			lat:           50.3518739,
			lon:           -4.4527880,
			wantText:      telegramLoadDefaultAlert,
			wantSendCalls: 1,
		},
		{
			name:          "custom load",
			lat:           1,
			lon:           2,
			wantText:      telegramLoadCustomAlert,
			wantSendCalls: 1,
		},
		{
			name:          "send error is logged not returned",
			lat:           1,
			lon:           2,
			telegramErr:   errors.New("telegram down"),
			wantText:      telegramLoadCustomAlert,
			wantSendCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			telegram := &recordingTelegramNotifier{err: tt.telegramErr}
			deps, err := NewDependencies(stubHTTPDoer{}, "k", WallClock{}, telegram, nil)
			if err != nil {
				t.Fatalf("NewDependencies: %v", err)
			}
			app := NewApplication(deps)

			app.notifyTelegramLoadSuccess(context.Background(), tt.lat, tt.lon)

			if telegram.sendCalls != tt.wantSendCalls {
				t.Fatalf("telegram send calls: got %d want %d texts=%v", telegram.sendCalls, tt.wantSendCalls, telegram.texts)
			}
			if tt.telegramErr == nil {
				if len(telegram.texts) != 1 || telegram.texts[0] != tt.wantText {
					t.Fatalf("telegram text: got %v want [%q]", telegram.texts, tt.wantText)
				}
			} else if len(telegram.texts) != 0 {
				t.Fatalf("telegram texts: got %v want none on send error", telegram.texts)
			}
		})
	}
}

func TestApplication_handleTides_telegramLoadNotification(t *testing.T) {
	t.Parallel()

	if !telegramLoadNotificationsEnabled {
		t.Skip("telegramLoadNotificationsEnabled is false")
	}

	at := time.Date(2022, 2, 2, 0, 0, 0, 0, time.UTC)
	upstreamBody := []byte(`{"status":200,"copyright":"x","requestDatum":"CD","responseDatum":"CD","extremes":[{"dt":1710994320,"height":4.81,"type":"High"}],"responseLat":1,"responseLon":2}`)
	fake := &fakeHTTPDoer{
		doFn: func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(upstreamBody)),
			}, nil
		},
	}

	tests := []struct {
		name     string
		query    string
		wantText string
	}{
		{
			name:     "custom location on success",
			query:    "/v1/tides?lat=1&lon=2",
			wantText: telegramLoadCustomAlert,
		},
		{
			name:     "looe default on success",
			query:    "/v1/tides?lat=50.3518739&lon=-4.4527880",
			wantText: telegramLoadDefaultAlert,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			telegram := &recordingTelegramNotifier{}
			deps, err := NewDependencies(fake, "k", fixedClock{t: at}, telegram, nil)
			if err != nil {
				t.Fatalf("NewDependencies: %v", err)
			}
			app := NewApplication(deps)

			req := httptest.NewRequest(http.MethodGet, tt.query, http.NoBody)
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status: got %d body=%q", rec.Code, rec.Body.String())
			}
			if len(telegram.texts) != 1 || telegram.texts[0] != tt.wantText {
				t.Fatalf("telegram texts: got %v want [%q]", telegram.texts, tt.wantText)
			}
		})
	}
}

func TestApplication_handleTides_skipsTelegramLoadNotificationOnFailure(t *testing.T) {
	t.Parallel()

	if !telegramLoadNotificationsEnabled {
		t.Skip("telegramLoadNotificationsEnabled is false")
	}

	at := time.Date(2022, 2, 2, 0, 0, 0, 0, time.UTC)
	fake := &fakeHTTPDoer{
		doFn: func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusBadGateway,
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"status":502,"error":"upstream down"}`))),
			}, nil
		},
	}
	telegram := &recordingTelegramNotifier{}
	deps, err := NewDependencies(fake, "k", fixedClock{t: at}, telegram, nil)
	if err != nil {
		t.Fatalf("NewDependencies: %v", err)
	}
	app := NewApplication(deps)

	req := httptest.NewRequest(http.MethodGet, "/v1/tides?lat=1&lon=2", http.NoBody)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status: got %d want %d", rec.Code, http.StatusBadGateway)
	}
	if len(telegram.texts) != 0 {
		t.Fatalf("telegram texts: got %v want none", telegram.texts)
	}
}
