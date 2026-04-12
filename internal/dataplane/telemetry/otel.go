// Package telemetry configures the OpenTelemetry pipeline for Veltrix.
//
// Veltrix uses OpenTelemetry (OTel) as the standard for all observability:
//
//   METRICS:
//     GPU telemetry (utilization, memory, power, temperature) flows from
//     the node agent through OTel SDK → OTLP → OTel Collector → Prometheus.
//     Grafana reads from Prometheus for real-time dashboards.
//
//   TRACES:
//     Every job's lifecycle is traced: submit → schedule → place → run → complete.
//     Traces show exactly where time is spent (queue wait, prediction, placement,
//     GPU configuration, execution). Essential for debugging scheduling latency.
//
//   LOGS:
//     Structured logs from all services are exported via OTel to a log backend.
//     Not implemented in MVP — services use standard Go log package.
//
// Why OTel over raw Prometheus:
//   - Push-based: natural for distributed agents (don't need Prometheus to scrape 10k nodes)
//   - Vendor-neutral: enterprise customers use Datadog, Cortex, Thanos — OTel exports to all
//   - Unified: metrics + traces + logs through one SDK
//   - CNCF standard: the industry direction
//
// Architecture:
//
//   Node Agent                Control Plane
//   ┌──────────┐              ┌──────────────┐
//   │ NVML     │              │ API Server   │
//   │   ↓      │              │   ↓          │
//   │ OTel SDK │──OTLP push──▶│ OTel Collector│──▶ Prometheus
//   │ (metrics)│              │ (all signals) │──▶ Jaeger (traces)
//   └──────────┘              └──────────────┘
//
// This package provides setup functions for both the agent-side and
// control-plane-side OTel configuration.
package telemetry

import "context"

// ---------------------------------------------------------------------------
// Configuration
// ---------------------------------------------------------------------------

// Config holds OpenTelemetry pipeline configuration.
type Config struct {
	// ServiceName is the OTel service name (e.g., "veltrix-api", "veltrix-agent").
	// Used in traces and metrics to identify the source service.
	ServiceName string

	// OTLPEndpoint is the address of the OTel Collector's OTLP receiver.
	// Example: "otel-collector:4317" (gRPC) or "otel-collector:4318" (HTTP).
	OTLPEndpoint string

	// Protocol is the OTLP transport protocol: "grpc" or "http".
	Protocol string

	// MetricsEnabled controls whether metrics are exported.
	MetricsEnabled bool

	// TracesEnabled controls whether traces are exported.
	TracesEnabled bool

	// MetricsIntervalSeconds is how often metrics are pushed to the collector.
	// Default: 15 seconds.
	MetricsIntervalSeconds int

	// SampleRate is the trace sampling rate (0.0–1.0).
	// 1.0 = sample everything (development). 0.01 = sample 1% (production).
	SampleRate float64
}

// ---------------------------------------------------------------------------
// Provider — the initialized OTel pipeline
// ---------------------------------------------------------------------------
//
// Provider wraps the OTel SDK components. It must be shut down cleanly
// on process exit to flush any buffered telemetry.
// ---------------------------------------------------------------------------

type Provider struct {
	config Config
	// TODO: add OTel SDK components
	// meterProvider  *metric.MeterProvider
	// tracerProvider *trace.TracerProvider
}

// NewProvider initializes the OTel pipeline and returns a Provider.
//
// This sets up:
//   - OTLP exporter (gRPC or HTTP, based on config)
//   - Meter provider (for metrics)
//   - Tracer provider (for traces)
//   - Resource with service name and version
//
// Call Shutdown() on process exit to flush buffered telemetry.
func NewProvider(cfg Config) (*Provider, error) {
	// TODO: implementation
	// 1. Create OTLP exporter based on protocol
	// 2. Create resource with service.name, service.version
	// 3. Create MeterProvider with periodic reader
	// 4. Create TracerProvider with batch span processor
	// 5. Set global providers

	return &Provider{config: cfg}, nil
}

// Shutdown flushes buffered telemetry and releases resources.
// Should be called with a timeout context on process exit.
//
//   provider, _ := telemetry.NewProvider(cfg)
//   defer provider.Shutdown(context.Background())
func (p *Provider) Shutdown(ctx context.Context) error {
	// TODO: implementation
	// 1. Shutdown MeterProvider (flushes metrics)
	// 2. Shutdown TracerProvider (flushes traces)
	return nil
}

// ---------------------------------------------------------------------------
// GPU Metrics — helper for the node agent
// ---------------------------------------------------------------------------
//
// The node agent uses these functions to record GPU telemetry as OTel metrics.
// The OTel SDK batches and pushes them to the collector at the configured interval.
// ---------------------------------------------------------------------------

// GPUMetricsRecorder provides methods to record GPU telemetry as OTel metrics.
type GPUMetricsRecorder struct {
	// TODO: add OTel instrument handles
	// utilizationGauge metric.Float64ObservableGauge
	// memoryUsedGauge  metric.Int64ObservableGauge
	// powerDrawGauge   metric.Float64ObservableGauge
	// temperatureGauge metric.Int64ObservableGauge
	// pcieThroughput   metric.Float64ObservableGauge
	// smOccupancy      metric.Float64ObservableGauge
}

// NewGPUMetricsRecorder creates instruments for recording GPU metrics.
//
// Each metric is a gauge (point-in-time value, not cumulative).
// Labels: gpu_id, node_id, gpu_model, device_index.
func NewGPUMetricsRecorder() (*GPUMetricsRecorder, error) {
	// TODO: implementation
	// 1. Get meter from global MeterProvider
	// 2. Create gauge instruments for each metric
	// 3. Register callback functions that read latest values
	return &GPUMetricsRecorder{}, nil
}

// RecordGPUTelemetry records a telemetry snapshot for a single GPU.
//
// This is called by the agent's collector every telemetry interval.
// The OTel SDK handles batching and export.
func (r *GPUMetricsRecorder) RecordGPUTelemetry(gpuID string, nodeID string, model string, deviceIndex int, utilization float64, memoryUsed int64, powerDraw float64, temperature int, pcieThroughput float64, smOccupancy float64) {
	// TODO: implementation
	// Record each value with appropriate labels/attributes:
	// gpu_id, node_id, gpu_model, device_index
}
