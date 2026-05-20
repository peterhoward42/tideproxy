package tideproxy

import (
	"net/http"
	"os"
	"sync"

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
	deps, err := app.NewDependencies(httpClient, os.Getenv("WORLDTIDES_API_KEY"), app.WallClock{}, telegram)
	if err != nil {
		handlerErr = err
		return
	}
	rootHandler = app.WithCORS(app.NewApplication(deps))
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
