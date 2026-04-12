package handlers

import (
	"net/http"

	"veltrix/internal/controlplane/feedback"
)

// ---------------------------------------------------------------------------
// MetricsHandler — HTTP handlers for metrics and analytics
// ---------------------------------------------------------------------------
//
// These endpoints expose aggregated metrics for the Grafana dashboard.
// Raw time-series metrics are read directly from Prometheus by Grafana —
// these endpoints provide computed/derived data that Prometheus doesn't have.
// ---------------------------------------------------------------------------

// ClusterSummaryResponse is the JSON response for GET /api/v1/metrics/cluster.
// Provides a high-level overview of cluster GPU utilization.
type ClusterSummaryResponse struct {
	// TotalGPUs is the total number of GPUs in the cluster.
	TotalGPUs int `json:"total_gpus"`

	// AllocatedGPUs is the number of GPUs with at least one active job.
	AllocatedGPUs int `json:"allocated_gpus"`

	// AvailableGPUs is the number of GPUs with no active jobs.
	AvailableGPUs int `json:"available_gpus"`

	// AvgUtilization is the cluster-wide average GPU utilization (0–100).
	AvgUtilization float64 `json:"avg_utilization"`

	// AvgMemoryUsage is the cluster-wide average GPU memory usage (0–100).
	AvgMemoryUsage float64 `json:"avg_memory_usage"`

	// TotalJobsRunning is the number of currently running jobs.
	TotalJobsRunning int `json:"total_jobs_running"`

	// TotalJobsQueued is the number of jobs waiting in the scheduler queue.
	TotalJobsQueued int `json:"total_jobs_queued"`

	// StrategyDistribution shows how GPUs are being used.
	// Example: {"full_gpu": 20, "mig": 8, "mps": 12, "idle": 10}
	StrategyDistribution map[string]int `json:"strategy_distribution"`

	// TopTenants shows GPU usage per tenant (for the admin dashboard).
	TopTenants []TenantUsage `json:"top_tenants"`
}

// TenantUsage shows how many GPUs a tenant is consuming.
type TenantUsage struct {
	Tenant      string `json:"tenant"`
	GPUsInUse   int    `json:"gpus_in_use"`
	JobsRunning int    `json:"jobs_running"`
	JobsQueued  int    `json:"jobs_queued"`
}

// AccuracyResponse is the JSON response for GET /api/v1/metrics/accuracy.
type AccuracyResponse struct {
	OverallAccuracy     float64            `json:"overall_accuracy"`
	VRAMAccuracy        float64            `json:"vram_accuracy"`
	UtilizationAccuracy float64            `json:"utilization_accuracy"`
	RuntimeAccuracy     float64            `json:"runtime_accuracy"`
	SampleCount         int                `json:"sample_count"`
	ByWorkloadType      map[string]float64 `json:"by_workload_type"`
	ByFramework         map[string]float64 `json:"by_framework"`
}

type MetricsHandler struct {
	feedback feedback.FeedbackController
}

// NewMetricsHandler creates a new MetricsHandler.
func NewMetricsHandler(fb feedback.FeedbackController) *MetricsHandler {
	return &MetricsHandler{feedback: fb}
}

// GetClusterSummary handles GET /api/v1/metrics/cluster.
//
// Aggregates cluster-wide GPU metrics for the Grafana overview dashboard.
// This data is computed from the Redis cache (latest GPU states) and
// the scheduler (queue depth).
func (h *MetricsHandler) GetClusterSummary(w http.ResponseWriter, r *http.Request) {
	// TODO: implementation
	// 1. Get all GPUs from cache
	// 2. Aggregate: total, allocated, available, avg utilization
	// 3. Get queue depth from scheduler
	// 4. Group GPU usage by strategy
	// 5. Group GPU usage by tenant
	// 6. Return ClusterSummaryResponse
	writeError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "not implemented")
}

// GetAccuracy handles GET /api/v1/metrics/accuracy.
//
// Returns prediction accuracy statistics from the feedback controller.
// Used by the Grafana dashboard to show a "prediction quality" panel.
func (h *MetricsHandler) GetAccuracy(w http.ResponseWriter, r *http.Request) {
	report, err := h.feedback.GetPredictionAccuracy(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "ACCURACY_FAILED", err.Error())
		return
	}

	if report == nil {
		writeError(w, http.StatusNotFound, "NO_DATA", "no prediction data available yet")
		return
	}

	// TODO: convert report to AccuracyResponse and return 200
	_ = report
	writeError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "not implemented")
}
