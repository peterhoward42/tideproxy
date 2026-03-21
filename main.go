package main

import (
	"log"
	"net/http"
	"os"

	"github.com/GoogleCloudPlatform/functions-framework-go/funcframework"

	"tideproxy/internal/app"
)

func main() {
	deps := app.Dependencies{
		HTTPClient:       http.DefaultClient,
		WorldTidesAPIKey: os.Getenv("WORLDTIDES_API_KEY"),
	}
	application := app.NewApplication(deps)
	funcframework.RegisterHTTPFunction("/", application.ServeHTTP)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if err := funcframework.Start(port); err != nil {
		log.Fatalf("start: %v", err)
	}
}
