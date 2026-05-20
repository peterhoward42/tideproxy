package app

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestTelegramBotNotifier_Send_success(t *testing.T) {
	t.Parallel()

	var gotMethod, gotURL, gotContentType string
	var gotBody []byte
	fake := &fakeHTTPDoer{
		doFn: func(req *http.Request) (*http.Response, error) {
			gotMethod = req.Method
			gotURL = req.URL.String()
			gotContentType = req.Header.Get("Content-Type")
			gotBody, _ = io.ReadAll(req.Body)
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"ok":true,"result":{}}`)),
			}, nil
		},
	}

	notifier, err := NewTelegramBotNotifier(fake, "bot-token", "12345")
	if err != nil {
		t.Fatalf("NewTelegramBotNotifier: %v", err)
	}

	if err := notifier.Send(context.Background(), "hello"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("method: got %q want POST", gotMethod)
	}
	if gotURL != "https://api.telegram.org/botbot-token/sendMessage" {
		t.Fatalf("url: got %q", gotURL)
	}
	if gotContentType != "application/json" {
		t.Fatalf("Content-Type: got %q want application/json", gotContentType)
	}
	if !bytes.Contains(gotBody, []byte(`"chat_id":"12345"`)) {
		t.Fatalf("body missing chat_id: %s", gotBody)
	}
	if !bytes.Contains(gotBody, []byte(`"text":"hello"`)) {
		t.Fatalf("body missing text: %s", gotBody)
	}
}

func TestTelegramBotNotifier_Send_transportError(t *testing.T) {
	t.Parallel()

	want := errors.New("network down")
	fake := &fakeHTTPDoer{
		doFn: func(*http.Request) (*http.Response, error) {
			return nil, want
		},
	}

	notifier, err := NewTelegramBotNotifier(fake, "bot-token", "12345")
	if err != nil {
		t.Fatalf("NewTelegramBotNotifier: %v", err)
	}

	err = notifier.Send(context.Background(), "hello")
	if err == nil {
		t.Fatal("Send: got nil want error")
	}
	if !strings.Contains(err.Error(), want.Error()) {
		t.Fatalf("Send: got %v want wrapped %v", err, want)
	}
}

func TestTelegramBotNotifier_Send_apiRejected(t *testing.T) {
	t.Parallel()

	fake := &fakeHTTPDoer{
		doFn: func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"ok":false,"description":"chat not found"}`)),
			}, nil
		},
	}

	notifier, err := NewTelegramBotNotifier(fake, "bot-token", "12345")
	if err != nil {
		t.Fatalf("NewTelegramBotNotifier: %v", err)
	}

	err = notifier.Send(context.Background(), "hello")
	if err == nil {
		t.Fatal("Send: got nil want error")
	}
	if !strings.Contains(err.Error(), "chat not found") {
		t.Fatalf("Send: got %v", err)
	}
}

func TestTelegramBotNotifier_Send_nonOKHTTPStatus(t *testing.T) {
	t.Parallel()

	fake := &fakeHTTPDoer{
		doFn: func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusBadGateway,
				Body:       io.NopCloser(strings.NewReader("bad gateway")),
			}, nil
		},
	}

	notifier, err := NewTelegramBotNotifier(fake, "bot-token", "12345")
	if err != nil {
		t.Fatalf("NewTelegramBotNotifier: %v", err)
	}

	err = notifier.Send(context.Background(), "hello")
	if err == nil {
		t.Fatal("Send: got nil want error")
	}
	if !strings.Contains(err.Error(), "HTTP 502") {
		t.Fatalf("Send: got %v", err)
	}
}

func TestNewTelegramBotNotifier_validation(t *testing.T) {
	t.Parallel()

	fake := &fakeHTTPDoer{
		doFn: func(*http.Request) (*http.Response, error) {
			t.Fatal("unexpected HTTP call")
			return nil, nil
		},
	}

	tests := []struct {
		name     string
		token    string
		chatID   string
		client   HTTPDoer
		wantErr  error
	}{
		{name: "nil client", token: "t", chatID: "1", client: nil, wantErr: ErrNilHTTPClient},
		{name: "empty token", token: "", chatID: "1", client: fake, wantErr: ErrEmptyTelegramBotToken},
		{name: "empty chat id", token: "t", chatID: "", client: fake, wantErr: ErrEmptyTelegramChatID},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := NewTelegramBotNotifier(tc.client, tc.token, tc.chatID)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("NewTelegramBotNotifier: got %v want %v", err, tc.wantErr)
			}
		})
	}
}

type fakeHTTPDoer struct {
	doFn func(*http.Request) (*http.Response, error)
}

func (f *fakeHTTPDoer) Do(req *http.Request) (*http.Response, error) {
	return f.doFn(req)
}
