package main

import (
	"log"
	"os"

	"github.com/GoogleCloudPlatform/functions-framework-go/funcframework"

	"tideproxy/internal/app"
)

func main() {
	deps := app.Dependencies{}
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
