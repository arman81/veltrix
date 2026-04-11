package main

import (
	"log"
	"os"

	"gpu-optimizer/agent/internal/scheduler"
)

func main() {

	endpoint := os.Getenv("API_ENDPOINT")

	if endpoint == "" {
		endpoint = "api:50051"
	}

	log.Println("Agent started")

	scheduler.Start(endpoint)
}
