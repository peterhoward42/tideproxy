package tideproxy

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"

	"cloud.google.com/go/storage"
	"github.com/peterhoward42/tideproxy/internal/app"
)

var (
	handlerOnce sync.Once
	rootHandler http.Handler
	handlerErr  error
)

func prepareHandler() {
	httpClient := http.DefaultClient
	telegram, err := app.NewTelegramBotNotifier(httpClient, os.Getenv("TELEGRAM_BOT_TOKEN"), os.Getenv("TELEGRAM_CHAT_ID"))
	if err != nil {
		handlerErr = err
		return
	}

	alertState, err := telegramAlertStateFromEnv(context.Background())
	if err != nil {
		handlerErr = err
		return
	}

	deps, err := app.NewDependencies(httpClient, os.Getenv("WORLDTIDES_API_KEY"), app.WallClock{}, telegram, alertState)
	if err != nil {
		handlerErr = err
		return
	}
	rootHandler = app.WithCORS(app.NewApplication(deps))
}

func telegramAlertStateFromEnv(ctx context.Context) (app.TelegramAlertStateStore, error) {
	bucket := os.Getenv("TELEGRAM_ALERT_STATE_BUCKET")
	path := os.Getenv("TELEGRAM_ALERT_STATE_PATH")
	switch {
	case bucket == "" && path == "":
		return app.NoopTelegramAlertStateStore{}, nil
	case bucket != "" && path != "":
		client, err := storage.NewClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("storage client: %w", err)
		}
		return app.NewGCSTelegramAlertStateStore(client, bucket, path)
	default:
		return nil, fmt.Errorf("TELEGRAM_ALERT_STATE_BUCKET and TELEGRAM_ALERT_STATE_PATH must both be set or both empty")
	}
}

// TidesProxy is the Google Cloud Functions (2nd gen) HTTP entry point; gcloud
// --entry-point must match this identifier.
func TidesProxy(w http.ResponseWriter, r *http.Request) {
	handlerOnce.Do(prepareHandler)
	if handlerErr != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	rootHandler.ServeHTTP(w, r)
}
