// Package feedback implements the Veltrix feedback controller.
//
// The feedback controller closes the loop between execution and scheduling.
// It continuously consumes GPU telemetry from the data plane and uses it to:
//
//   1. Compare predicted vs actual resource usage (prediction accuracy)
//   2. Update the prediction engine with actual outcomes
//   3. Detect anomalies (unexpected OOM, thermal throttling, underutilization)
//   4. Trigger re-scheduling when conditions change significantly
//   5. Generate optimization recommendations
//
// Without the feedback controller, Veltrix is an open-loop system —
// scheduling decisions are made once and never revisited. The feedback
// controller makes it closed-loop: decisions improve over time.
//
// Data flow:
//
//   Node Agent → OTel → [metrics.ingested queue] → Feedback Controller
//                                                         │
//                          ┌──────────────────────────────┘
//                          │
//                          ├── updates Prediction Engine
//                          ├── writes to metrics history (Postgres)
//                          ├── detects anomalies → alerts
//                          └── triggers re-scheduling → Scheduler
//
// The feedback controller runs as a continuous consumer of the
// metrics.ingested queue topic.
package feedback

import (
	"context"

	"veltrix/internal/controlplane/domain"
)

// ---------------------------------------------------------------------------
// FeedbackController — the public interface
// ---------------------------------------------------------------------------

type FeedbackController interface {
	// ProcessMetrics handles a batch of GPU telemetry from a single node.
	//
	// For each GPU in the batch:
	//   1. Update the hot cache (Redis) with latest state
	//   2. Store in metrics history (Postgres)
	//   3. Compare against active placement's predictions
	//   4. If anomaly detected, trigger appropriate action
	//
	// This is the hot path — called every telemetry interval (5s) per node.
	// Must be fast. Heavy analysis should be deferred to background jobs.
	ProcessMetrics(ctx context.Context, nodeID string, metrics []GPUMetricEvent) error

	// ProcessJobCompleted handles a job completion event.
	//
	// Compares the job's predicted resource usage against actual usage over
	// the job's lifetime. Sends the comparison to the prediction engine
	// for model improvement.
	ProcessJobCompleted(ctx context.Context, jobID string) error

	// GetPredictionAccuracy returns prediction accuracy statistics.
	//
	// Used by the Grafana dashboard to show how well the prediction engine
	// is performing. Returns accuracy breakdown by workload type and framework.
	GetPredictionAccuracy(ctx context.Context) (*AccuracyReport, error)

	// Start begins consuming telemetry events from the queue.
	// Blocks until the context is cancelled. Should be called in a goroutine.
	Start(ctx context.Context) error

	// Stop gracefully shuts down the feedback controller.
	Stop(ctx context.Context) error
}

// ---------------------------------------------------------------------------
// GPUMetricEvent — a single telemetry data point from the queue
// ---------------------------------------------------------------------------

type GPUMetricEvent struct {
	// GPUID identifies which GPU this metric is from.
	GPUID string

	// DeviceIndex is the GPU's index on the node.
	DeviceIndex int

	// Telemetry is the actual metrics snapshot.
	Telemetry domain.GPUTelemetry

	// Timestamp is when the metric was collected (Unix nanoseconds).
	Timestamp int64
}

// ---------------------------------------------------------------------------
// Anomaly — a detected deviation from expected behavior
// ---------------------------------------------------------------------------

type AnomalyType string

const (
	// AnomalyTypeOOM — GPU memory usage exceeded predicted maximum.
	// Action: alert operator, consider preempting co-located jobs.
	AnomalyTypeOOM AnomalyType = "oom"

	// AnomalyTypeThermalThrottle — GPU temperature exceeded safe threshold.
	// Action: alert operator, drain node if persistent.
	AnomalyTypeThermalThrottle AnomalyType = "thermal_throttle"

	// AnomalyTypeUnderutilization — GPU utilization is far below prediction.
	// Action: candidate for co-location via MPS. Inform scheduler.
	AnomalyTypeUnderutilization AnomalyType = "underutilization"

	// AnomalyTypeOverutilization — GPU utilization exceeds safe co-location threshold.
	// Action: consider migrating one of the co-located jobs.
	AnomalyTypeOverutilization AnomalyType = "overutilization"

	// AnomalyTypeStaleTelemetry — no metrics received from a GPU within the timeout.
	// Action: mark GPU as offline, alert operator.
	AnomalyTypeStaleTelemetry AnomalyType = "stale_telemetry"
)

type Anomaly struct {
	// Type classifies the anomaly.
	Type AnomalyType

	// GPUID is the affected GPU.
	GPUID string

	// NodeID is the affected node.
	NodeID string

	// JobIDs lists the jobs running on the affected GPU.
	JobIDs []string

	// Message is a human-readable description.
	Message string

	// Severity is 1 (info), 2 (warning), or 3 (critical).
	Severity int
}

// ---------------------------------------------------------------------------
// AccuracyReport — prediction engine performance metrics
// ---------------------------------------------------------------------------

type AccuracyReport struct {
	// OverallAccuracy is the mean accuracy across all predictions (0.0–1.0).
	OverallAccuracy float64

	// VRAMAccuracy is the mean accuracy of VRAM predictions.
	VRAMAccuracy float64

	// UtilizationAccuracy is the mean accuracy of GPU utilization predictions.
	UtilizationAccuracy float64

	// RuntimeAccuracy is the mean accuracy of runtime predictions.
	RuntimeAccuracy float64

	// SampleCount is the number of completed jobs used in this report.
	SampleCount int

	// ByWorkloadType breaks down accuracy per workload type.
	ByWorkloadType map[domain.WorkloadType]float64

	// ByFramework breaks down accuracy per ML framework.
	ByFramework map[string]float64
}

// ---------------------------------------------------------------------------
// Default implementation
// ---------------------------------------------------------------------------

// DefaultFeedbackController is the production implementation.
type DefaultFeedbackController struct {
	// TODO: dependencies will be injected via constructor
	// scheduler  scheduler.Scheduler    (for triggering re-scheduling)
	// prediction scheduler.PredictionClient (for updating models)
	// metrics    repository.MetricsRepository
	// placements repository.PlacementRepository
	// cache      cache.GPUStateCache
	// queue      queue.Subscriber
}

// NewFeedbackController creates a new DefaultFeedbackController.
func NewFeedbackController() *DefaultFeedbackController {
	return &DefaultFeedbackController{}
}

func (c *DefaultFeedbackController) ProcessMetrics(ctx context.Context, nodeID string, metrics []GPUMetricEvent) error {
	// TODO: implementation
	// 1. For each GPU metric:
	//    a. Update Redis cache with latest state
	//    b. Batch-insert into Postgres metrics table
	//    c. Look up active placement for this GPU
	//    d. Compare actual vs predicted utilization/VRAM
	//    e. If deviation > threshold → create Anomaly
	// 2. Process any anomalies (alert, re-schedule, etc.)
	return nil
}

func (c *DefaultFeedbackController) ProcessJobCompleted(ctx context.Context, jobID string) error {
	// TODO: implementation
	// 1. Load the placement for this job
	// 2. Aggregate actual metrics over the job's lifetime
	// 3. Compute prediction error:
	//    - VRAM: abs(predicted - actual_peak) / predicted
	//    - Utilization: abs(predicted - actual_mean) / predicted
	//    - Runtime: abs(predicted - actual) / predicted
	// 4. Store the comparison in the predictions table
	// 5. Send feedback to the prediction engine
	return nil
}

func (c *DefaultFeedbackController) GetPredictionAccuracy(ctx context.Context) (*AccuracyReport, error) {
	// TODO: implementation
	// 1. Query predictions table for recent completed jobs
	// 2. Compute accuracy statistics
	// 3. Group by workload type and framework
	return nil, nil
}

func (c *DefaultFeedbackController) Start(ctx context.Context) error {
	// TODO: implementation
	// 1. Subscribe to metrics.ingested queue topic
	// 2. Subscribe to jobs.completed queue topic
	// 3. Process events in a loop until context is cancelled
	return nil
}

func (c *DefaultFeedbackController) Stop(ctx context.Context) error {
	// TODO: implementation
	return nil
}
