package app

import (
	"context"
	"log"
)

// telegramLoadNotificationsEnabled gates Telegram alerts for successful tide
// loads. Set to false and redeploy to silence load notifications during a
// traffic spike without affecting credit-exhaustion alerts.
const telegramLoadNotificationsEnabled = true

// Looe default-load envelope: padded bounding box around East Looe, West Looe,
// and Looe (town) Nominatim coordinates.
const (
	looeDefaultLoadLatMin = 50.351
	looeDefaultLoadLatMax = 50.355
	looeDefaultLoadLonMin = -4.459
	looeDefaultLoadLonMax = -4.451
)

const (
	telegramLoadDefaultAlert = "tideproxy:load:default"
	telegramLoadCustomAlert  = "tideproxy:load:custom"
)

func isLooeDefaultLoad(lat, lon float64) bool {
	return lat >= looeDefaultLoadLatMin && lat <= looeDefaultLoadLatMax &&
		lon >= looeDefaultLoadLonMin && lon <= looeDefaultLoadLonMax
}

func loadTelegramAlertForLatLon(lat, lon float64) string {
	if isLooeDefaultLoad(lat, lon) {
		return telegramLoadDefaultAlert
	}
	return telegramLoadCustomAlert
}

func (a *Application) notifyTelegramLoadSuccess(ctx context.Context, lat, lon float64) {
	if !telegramLoadNotificationsEnabled {
		return
	}
	if err := a.deps.Telegram.Send(ctx, loadTelegramAlertForLatLon(lat, lon)); err != nil {
		log.Printf("telegram: %v", err)
	}
}
