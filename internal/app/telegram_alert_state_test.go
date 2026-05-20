package app

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
)

type memoryTelegramAlertStateStore struct {
	lastSent string
	readErr  error
	writeErr error
	writes   []string
}

func (m *memoryTelegramAlertStateStore) ReadLastSentHour(context.Context) (string, error) {
	if m.readErr != nil {
		return "", m.readErr
	}
	return m.lastSent, nil
}

func (m *memoryTelegramAlertStateStore) WriteLastSentHour(_ context.Context, hour string) error {
	m.writes = append(m.writes, hour)
	if m.writeErr != nil {
		return m.writeErr
	}
	m.lastSent = hour
	return nil
}

func TestUTCCalendarHour(t *testing.T) {
	t.Parallel()

	at := time.Date(2021, 5, 5, 14, 30, 0, 0, time.FixedZone("BST", 3600))
	if got := UTCCalendarHour(at); got != "2021-05-05T13" {
		t.Fatalf("UTCCalendarHour: got %q want 2021-05-05T13", got)
	}
}

func TestApplication_notifyTelegramCreditsExhausted(t *testing.T) {
	t.Parallel()

	at := time.Date(2021, 5, 5, 14, 30, 0, 0, time.UTC)
	wantHour := "2021-05-05T14"

	tests := []struct {
		name         string
		lastSent     string
		readErr      error
		writeErr     error
		telegramErr     error
		wantSendCalls   int
		wantWrite       bool
		wantWritten     string
	}{
		{
			name:          "never sent",
			wantSendCalls: 1,
			wantWrite:     true,
			wantWritten:   wantHour,
		},
		{
			name:          "same hour skips",
			lastSent:      wantHour,
			wantSendCalls: 0,
			wantWrite:     false,
		},
		{
			name:          "different hour sends",
			lastSent:      "2021-05-05T13",
			wantSendCalls: 1,
			wantWrite:     true,
			wantWritten:   wantHour,
		},
		{
			name:          "read error still sends",
			readErr:       errors.New("gcs down"),
			wantSendCalls: 1,
			wantWrite:     true,
			wantWritten:   wantHour,
		},
		{
			name:          "telegram error skips write",
			telegramErr:   errors.New("telegram down"),
			wantSendCalls: 1,
			wantWrite:     false,
		},
		{
			name:          "write error after successful send",
			writeErr:      errors.New("gcs write failed"),
			wantSendCalls: 1,
			wantWrite:     true,
			wantWritten:   wantHour,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			telegram := &recordingTelegramNotifier{err: tt.telegramErr}
			state := &memoryTelegramAlertStateStore{
				lastSent: tt.lastSent,
				readErr:  tt.readErr,
				writeErr: tt.writeErr,
			}
			deps, err := NewDependencies(
				stubHTTPDoer{},
				"k",
				fixedClock{t: at},
				telegram,
				state,
			)
			if err != nil {
				t.Fatalf("NewDependencies: %v", err)
			}
			app := NewApplication(deps)
			app.notifyTelegramCreditsExhausted(context.Background())

			if telegram.sendCalls != tt.wantSendCalls {
				t.Fatalf("telegram send calls: got %d want %d", telegram.sendCalls, tt.wantSendCalls)
			}
			if tt.wantSendCalls > 0 && len(telegram.texts) == 1 && telegram.texts[0] != telegramCreditsExhaustedAlert {
				t.Fatalf("telegram text: got %q", telegram.texts[0])
			}

			wrote := len(state.writes) == 1
			if wrote != tt.wantWrite {
				t.Fatalf("state write: got %v want %v (writes=%v)", wrote, tt.wantWrite, state.writes)
			}
			if tt.wantWrite && state.writes[0] != tt.wantWritten {
				t.Fatalf("written hour: got %q want %q", state.writes[0], tt.wantWritten)
			}
		})
	}
}

type recordingTelegramNotifier struct {
	texts    []string
	err      error
	sendCalls int
}

func (r *recordingTelegramNotifier) Send(_ context.Context, text string) error {
	r.sendCalls++
	if r.err != nil {
		return r.err
	}
	r.texts = append(r.texts, text)
	return nil
}

type stubHTTPDoer struct{}

func (stubHTTPDoer) Do(*http.Request) (*http.Response, error) {
	return nil, errors.New("unused")
}

type fixedClock struct {
	t time.Time
}

func (c fixedClock) Now() time.Time { return c.t }
