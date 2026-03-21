package main

import (
	"log"
	"os"

	"github.com/GoogleCloudPlatform/functions-framework-go/funcframework"
	"github.com/peterhoward42/tideproxy"
)

func main() {
	funcframework.RegisterHTTPFunction("/", tideproxy.TidesProxy)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if err := funcframework.Start(port); err != nil {
		log.Fatalf("start: %v", err)
	}
}
