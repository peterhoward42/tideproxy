package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

var (
	// ErrEmptyTelegramBotToken is returned by [NewTelegramBotNotifier] when botToken is empty.
	ErrEmptyTelegramBotToken = errors.New("app: TelegramBotToken is required")
	// ErrEmptyTelegramChatID is returned by [NewTelegramBotNotifier] when chatID is empty.
	ErrEmptyTelegramChatID = errors.New("app: TelegramChatID is required")
	// ErrNilTelegramNotifier is returned by [NewDependencies] when telegram is nil.
	ErrNilTelegramNotifier = errors.New("app: TelegramNotifier is required")
)

// TelegramNotifier sends text messages via Telegram.
type TelegramNotifier interface {
	Send(ctx context.Context, text string) error
}

// TelegramBotNotifier sends messages using Telegram's sendMessage Bot API.
type TelegramBotNotifier struct {
	httpClient HTTPDoer
	botToken   string
	chatID     string
}

// NewTelegramBotNotifier returns a notifier configured with botToken and chatID.
func NewTelegramBotNotifier(httpClient HTTPDoer, botToken, chatID string) (TelegramBotNotifier, error) {
	if httpClient == nil {
		return TelegramBotNotifier{}, ErrNilHTTPClient
	}
	if botToken == "" {
		return TelegramBotNotifier{}, ErrEmptyTelegramBotToken
	}
	if chatID == "" {
		return TelegramBotNotifier{}, ErrEmptyTelegramChatID
	}
	return TelegramBotNotifier{
		httpClient: httpClient,
		botToken:   botToken,
		chatID:     chatID,
	}, nil
}

// Send posts text to the configured Telegram chat.
func (n TelegramBotNotifier) Send(ctx context.Context, text string) error {
	payload, err := json.Marshal(map[string]string{
		"chat_id": n.chatID,
		"text":    text,
	})
	if err != nil {
		return fmt.Errorf("telegram: marshal sendMessage body: %w", err)
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", n.botToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("telegram: build sendMessage request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("telegram: sendMessage request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("telegram: read sendMessage response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram: sendMessage HTTP %d: %s", resp.StatusCode, bytes.TrimSpace(body))
	}

	var decoded struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		return fmt.Errorf("telegram: decode sendMessage response: %w", err)
	}
	if !decoded.OK {
		if decoded.Description == "" {
			return errors.New("telegram: sendMessage rejected")
		}
		return fmt.Errorf("telegram: sendMessage rejected: %s", decoded.Description)
	}
	return nil
}
