// Command api starts the Veltrix control plane API server.
//
// This is the main entry point for the Veltrix control plane. It wires
// together all components and starts the HTTP server.
//
// Startup sequence:
//   1. Load configuration from environment variables
//   2. Connect to Postgres (state store)
//   3. Connect to Redis (cache + locks)
//   4. Initialize the queue (in-process for MVP)
//   5. Initialize the OTel pipeline
//   6. Create all repositories
//   7. Create control plane services (policy → scheduler → placement → feedback)
//   8. Create and start the API server
//   9. Start background services (scheduler loop, feedback consumer)
//   10. Wait for shutdown signal (SIGINT/SIGTERM)
//   11. Graceful shutdown (drain requests, stop consumers, flush telemetry)
//
// Configuration (environment variables):
//   VELTRIX_PORT           — API server port (default: "8080")
//   VELTRIX_POSTGRES_HOST  — Postgres host (default: "localhost")
//   VELTRIX_POSTGRES_PORT  — Postgres port (default: "5432")
//   VELTRIX_POSTGRES_DB    — Postgres database (default: "veltrix")
//   VELTRIX_POSTGRES_USER  — Postgres user (default: "postgres")
//   VELTRIX_POSTGRES_PASS  — Postgres password
//   VELTRIX_REDIS_ADDR     — Redis address (default: "localhost:6379")
//   VELTRIX_REDIS_PASS     — Redis password
//   VELTRIX_OTEL_ENDPOINT  — OTel Collector endpoint (default: "localhost:4317")
//   VELTRIX_AUTH_ENABLED   — Enable authentication (default: "false")
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	log.Println("Starting Veltrix control plane...")

	// --- Configuration ---
	port := envOr("VELTRIX_PORT", "8080")
	_ = port

	// TODO: implementation
	// 1. Load all config from environment
	// 2. Connect to Postgres
	//    db, err := postgres.Connect(postgres.Config{...})
	// 3. Connect to Redis
	//    cache := cache.NewRedisCache(cache.RedisConfig{...})
	// 4. Create queue
	//    q := queue.NewMemoryQueue(queue.MemoryQueueConfig{BufferSize: 1000})
	// 5. Initialize OTel
	//    otel, err := telemetry.NewProvider(telemetry.Config{...})
	// 6. Create store (all repositories)
	//    store := postgres.NewStore(db)
	// 7. Create policy engine
	//    policyEngine := policy.NewPolicyEngine()
	// 8. Create scheduler
	//    sched := scheduler.NewScheduler(predictionClient)
	// 9. Create feedback controller
	//    fb := feedback.NewFeedbackController()
	// 10. Create API server
	//     server := api.NewServer(api.Config{Port: port}, sched, fb, auth)
	// 11. Start background services in goroutines
	//     go sched.Start(ctx)
	//     go fb.Start(ctx)
	// 12. Start API server in goroutine
	//     go server.Start()

	// --- Graceful shutdown ---
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down Veltrix control plane...")

	ctx, cancel := context.WithTimeout(context.Background(), 30)
	defer cancel()

	// TODO: shutdown in reverse order
	// server.Stop(ctx)
	// fb.Stop(ctx)
	// sched.Stop(ctx)
	// otel.Shutdown(ctx)
	// cache.Close()
	// store.Close()
	// q.Close()

	_ = ctx

	log.Println("Veltrix control plane stopped.")
}

// envOr returns the value of the environment variable or a default.
func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
