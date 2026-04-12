// Command agent starts the Veltrix node agent.
//
// The node agent runs as a Kubernetes DaemonSet on every GPU node.
// It is the execution plane's representative on the node.
//
// Responsibilities:
//   1. Collect GPU telemetry via NVML every 5 seconds
//   2. Push telemetry to the control plane via OTel SDK (OTLP)
//   3. Receive placement instructions from the control plane (via queue polling)
//   4. Configure GPUs: enable MPS, apply MIG profiles, set CUDA env vars
//   5. Launch job containers (or coordinate with kubelet)
//   6. Report job completion/failure back to the control plane
//   7. Send heartbeats to register/maintain node status
//
// The agent is intentionally simple — it does NOT make scheduling decisions.
// It executes instructions from the control plane and reports status.
//
// Configuration (environment variables):
//   VELTRIX_API_ENDPOINT   — Control plane API URL (default: "http://localhost:8080")
//   VELTRIX_NODE_ID        — Unique node identifier (default: hostname)
//   VELTRIX_OTEL_ENDPOINT  — OTel Collector endpoint (default: "localhost:4317")
//   VELTRIX_COLLECT_INTERVAL — Telemetry collection interval (default: "5s")
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	log.Println("Starting Veltrix node agent...")

	nodeID := envOr("VELTRIX_NODE_ID", hostname())
	_ = nodeID

	// TODO: implementation
	// 1. Load configuration from environment
	// 2. Initialize OTel SDK (push metrics to collector)
	// 3. Discover GPUs on this node via NVML
	// 4. Register node with the control plane (POST /api/v1/nodes)
	// 5. Start telemetry collection loop (NVML → OTel → Collector)
	// 6. Start heartbeat loop (periodic PUT to control plane)
	// 7. Start instruction listener (poll queue for placement decisions)
	// 8. Wait for shutdown signal

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down Veltrix node agent...")

	ctx, cancel := context.WithTimeout(context.Background(), 10)
	defer cancel()
	_ = ctx

	log.Println("Veltrix node agent stopped.")
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func hostname() string {
	h, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return h
}
