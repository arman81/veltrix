package domain

import "time"

// ---------------------------------------------------------------------------
// Placement — the output of the placement engine
// ---------------------------------------------------------------------------
//
// A Placement is the decision of WHERE and HOW a job runs on the cluster.
// It binds a Job to a specific GPU (or set of GPUs) on a specific Node,
// using a specific strategy (full GPU, MIG partition, or MPS sharing).
//
// The placement lifecycle:
//   1. Placement engine creates a Placement (status: Decided)
//   2. Control plane sends instruction to the node agent (status: Applying)
//   3. Agent configures the GPU and launches the container (status: Active)
//   4. Job completes or fails → placement is released (status: Released)
//
// A Placement is immutable once Released. Historical placements are kept
// for the feedback controller to compare predicted vs actual resource usage.
// ---------------------------------------------------------------------------

// PlacementStatus tracks the lifecycle of a placement decision.
type PlacementStatus string

const (
	// PlacementStatusDecided — the placement engine has made a decision
	// but the agent has not yet been instructed.
	PlacementStatusDecided PlacementStatus = "decided"

	// PlacementStatusApplying — the instruction has been sent to the agent.
	// The agent is configuring the GPU (setting up MIG/MPS, pulling image, etc.).
	PlacementStatusApplying PlacementStatus = "applying"

	// PlacementStatusActive — the job is running on the assigned GPU.
	// Telemetry is being collected and compared against predictions.
	PlacementStatusActive PlacementStatus = "active"

	// PlacementStatusReleased — the job has finished and the GPU resources
	// have been freed. Terminal state.
	PlacementStatusReleased PlacementStatus = "released"

	// PlacementStatusFailed — the agent failed to apply the placement.
	// The job returns to the queue for re-scheduling. Terminal state.
	PlacementStatusFailed PlacementStatus = "failed"
)

// ---------------------------------------------------------------------------
// Resource Prediction — what the prediction engine estimated
// ---------------------------------------------------------------------------
//
// Stored with each placement so the feedback controller can compute
// prediction accuracy: abs(predicted - actual) / predicted.
// This feedback is used to retrain/adjust the prediction engine.
// ---------------------------------------------------------------------------

type ResourcePrediction struct {
	// PredictedVRAMBytes is the estimated peak GPU memory usage.
	PredictedVRAMBytes int64

	// PredictedUtilization is the estimated average GPU utilization (0–100).
	PredictedUtilization float64

	// PredictedRuntimeSeconds is the estimated job duration.
	PredictedRuntimeSeconds int64

	// Confidence is the prediction engine's confidence score (0.0–1.0).
	// Low confidence triggers more conservative placement (e.g., full GPU
	// instead of MIG, to avoid OOM from underestimation).
	Confidence float64
}

// ---------------------------------------------------------------------------
// GPU Assignment — which specific GPU resource is assigned
// ---------------------------------------------------------------------------
//
// For a full-GPU placement, this points to a single GPU.
// For MIG, it points to a specific MIG instance on a GPU.
// For MPS, it points to a GPU that is shared with other jobs.
// For multi-GPU jobs, the Placement has multiple GPUAssignments.
// ---------------------------------------------------------------------------

type GPUAssignment struct {
	// GPUID is the ID of the assigned GPU.
	GPUID string

	// DeviceIndex is the GPU's index on the node (for CUDA_VISIBLE_DEVICES).
	DeviceIndex int

	// MIGInstanceID is set when Strategy == GPUStrategyMIG.
	// Identifies the specific MIG partition assigned to this job.
	MIGInstanceID string
}

// ---------------------------------------------------------------------------
// Placement — the domain entity
// ---------------------------------------------------------------------------

type Placement struct {
	// --- Identity ---

	// ID is a globally unique identifier for this placement (UUID v4).
	ID string

	// --- Binding ---

	// JobID is the job this placement is for.
	JobID string

	// NodeID is the node where the job will run.
	NodeID string

	// GPUAssignments lists the GPU resources assigned to this job.
	// Single-GPU jobs have exactly one assignment.
	// Multi-GPU jobs have one assignment per GPU.
	GPUAssignments []GPUAssignment

	// Strategy is how the GPU is being used for this job.
	Strategy GPUStrategy

	// --- Decision Metadata ---

	// Prediction is the resource estimate that informed this placement.
	// Used by the feedback controller after the job completes.
	Prediction ResourcePrediction

	// NodeScore is the score the placement engine gave this node (0.0–1.0).
	// Higher is better. Stored for debugging and tuning the scoring algorithm.
	NodeScore float64

	// Reason is a human-readable explanation of why this placement was chosen.
	// Example: "node-7 selected: 40GB free VRAM, 35% utilization, MPS compatible"
	Reason string

	// --- Lifecycle ---

	// Status is the current state of the placement.
	Status PlacementStatus

	// DecidedAt is when the placement engine made the decision.
	DecidedAt time.Time

	// AppliedAt is when the agent confirmed GPU configuration was complete.
	AppliedAt *time.Time

	// ReleasedAt is when the GPU resources were freed.
	ReleasedAt *time.Time
}
