package main

import (
	"log"
	"net/http"
	"os"

	"github.com/GoogleCloudPlatform/functions-framework-go/funcframework"

	"tideproxy/internal/app"
)

func main() {
	deps, err := app.NewDependencies(http.DefaultClient, os.Getenv("WORLDTIDES_API_KEY"), app.WallClock{})
	if err != nil {
		log.Fatalf("dependencies: %v", err)
	}
	application := app.NewApplication(deps)
	funcframework.RegisterHTTPFunction("/", app.WithCORS(application))
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if err := funcframework.Start(port); err != nil {
		log.Fatalf("start: %v", err)
	}
}
