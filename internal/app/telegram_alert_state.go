package app

import (
	"context"
	"log"
	"strings"
	"time"
)

// TelegramAlertStateStore persists the UTC calendar hour of the last successful
// quota-exhaustion Telegram alert for cross-instance dedupe.
type TelegramAlertStateStore interface {
	ReadLastSentHour(ctx context.Context) (hour string, err error)
	WriteLastSentHour(ctx context.Context, hour string) error
}

// NoopTelegramAlertStateStore reports no prior send and ignores writes. Used when
// TELEGRAM_ALERT_STATE_BUCKET and TELEGRAM_ALERT_STATE_PATH are unset (e.g. local dev).
type NoopTelegramAlertStateStore struct{}

func (NoopTelegramAlertStateStore) ReadLastSentHour(context.Context) (string, error) {
	return "", nil
}

func (NoopTelegramAlertStateStore) WriteLastSentHour(context.Context, string) error {
	return nil
}

// UTCCalendarHour formats t as a UTC calendar hour bucket (2006-01-02T15).
func UTCCalendarHour(t time.Time) string {
	return t.UTC().Truncate(time.Hour).Format("2006-01-02T15")
}

func (a *Application) notifyTelegramCreditsExhausted(ctx context.Context) {
	currentHour := UTCCalendarHour(a.deps.Clock.Now())

	lastSent, err := a.deps.TelegramAlertState.ReadLastSentHour(ctx)
	if err != nil {
		log.Printf("telegram alert state: read: %v", err)
	} else if lastSent == currentHour {
		return
	}

	if err := a.deps.Telegram.Send(ctx, telegramCreditsExhaustedAlert); err != nil {
		log.Printf("telegram: %v", err)
		return
	}

	if err := a.deps.TelegramAlertState.WriteLastSentHour(ctx, currentHour); err != nil {
		log.Printf("telegram alert state: write: %v", err)
	}
}

func normalizeStoredHour(data []byte) string {
	return strings.TrimSpace(string(data))
}
