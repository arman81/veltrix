// Package domain defines the core business entities for the Veltrix control plane.
//
// These types represent the fundamental concepts that the control plane reasons about:
// jobs, GPUs, nodes, placements, and policies. Every control plane service operates
// on these types. The data plane never imports this package — it deals in raw bytes
// and generic interfaces.
//
// Design principle: domain types are pure data. No database tags, no JSON tags,
// no framework dependencies. Serialization is handled at the boundary (API handlers,
// repository implementations).
package domain

import "time"

// ---------------------------------------------------------------------------
// Job Status — the lifecycle state machine
// ---------------------------------------------------------------------------
//
// A job moves through these states:
//
//   Pending → Scheduled → Placing → Running → Completed
//                                           → Failed
//   (any state) → Cancelled
//   Running → Preempted → Pending (re-enters the queue)
//
// State transitions are enforced by the scheduler. Invalid transitions
// (e.g., Completed → Running) must be rejected.
// ---------------------------------------------------------------------------

type JobStatus string

const (
	// JobStatusPending — job has been submitted and is waiting in the priority queue.
	// This is the initial state for every job.
	JobStatusPending JobStatus = "pending"

	// JobStatusScheduled — the scheduler has accepted the job and the prediction
	// engine has estimated its resource requirements. Waiting for placement.
	JobStatusScheduled JobStatus = "scheduled"

	// JobStatusPlacing — the placement engine has selected a node and GPU.
	// The agent is being instructed to configure the GPU (MIG/MPS/full).
	JobStatusPlacing JobStatus = "placing"

	// JobStatusRunning — the job is actively executing on a GPU.
	// Telemetry is being collected. The feedback controller is monitoring.
	JobStatusRunning JobStatus = "running"

	// JobStatusCompleted — the job finished successfully.
	// Terminal state. Metrics are finalized.
	JobStatusCompleted JobStatus = "completed"

	// JobStatusFailed — the job failed due to an error (OOM, crash, timeout).
	// Terminal state. The failure reason is stored in Job.FailureReason.
	JobStatusFailed JobStatus = "failed"

	// JobStatusCancelled — the job was cancelled by a user or policy.
	// Terminal state. Can happen from any non-terminal state.
	JobStatusCancelled JobStatus = "cancelled"

	// JobStatusPreempted — the job was evicted by a higher-priority job.
	// Non-terminal: the job re-enters the queue as Pending with its original priority.
	JobStatusPreempted JobStatus = "preempted"
)

// ---------------------------------------------------------------------------
// Job Priority
// ---------------------------------------------------------------------------
//
// Priority is a numeric value from 0–100. Higher = more urgent.
// The scheduler uses this to order the priority queue.
// The policy engine may override priority based on tenant SLAs.
// ---------------------------------------------------------------------------

type JobPriority int

const (
	JobPriorityLow      JobPriority = 0
	JobPriorityNormal   JobPriority = 50
	JobPriorityHigh     JobPriority = 80
	JobPriorityCritical JobPriority = 100
)

// ---------------------------------------------------------------------------
// Workload Type
// ---------------------------------------------------------------------------
//
// The workload type affects scheduling strategy:
//   - Training: high VRAM, long-running, tolerates co-location poorly
//   - Inference: low VRAM, latency-sensitive, ideal for MPS co-location
//   - FineTuning: moderate VRAM, medium duration, similar to training
//   - Evaluation: read-heavy, short bursts, can share GPUs freely
//   - Preprocessing: CPU-heavy with occasional GPU, lowest GPU priority
// ---------------------------------------------------------------------------

type WorkloadType string

const (
	WorkloadTypeTraining      WorkloadType = "training"
	WorkloadTypeInference     WorkloadType = "inference"
	WorkloadTypeFineTuning    WorkloadType = "fine_tuning"
	WorkloadTypeEvaluation    WorkloadType = "evaluation"
	WorkloadTypePreprocessing WorkloadType = "preprocessing"
)

// ---------------------------------------------------------------------------
// Resource Requirements
// ---------------------------------------------------------------------------
//
// Defines what a job needs from the GPU. The prediction engine may override
// these values based on historical data (e.g., a PyTorch ResNet-50 job
// always uses ~4GB regardless of what the user requests).
// ---------------------------------------------------------------------------

// ComputePrecision indicates the floating-point precision the workload uses.
// This affects SM utilization and memory bandwidth requirements.
type ComputePrecision string

const (
	PrecisionFP32  ComputePrecision = "fp32"
	PrecisionFP16  ComputePrecision = "fp16"
	PrecisionBF16  ComputePrecision = "bf16"
	PrecisionINT8  ComputePrecision = "int8"
	PrecisionMixed ComputePrecision = "mixed"
)

type ResourceRequirements struct {
	// VRAMBytes is the minimum GPU memory required, in bytes.
	// The placement engine uses this to filter GPUs with insufficient memory
	// and to decide MIG partition sizes.
	VRAMBytes int64

	// MinSMs is the minimum number of streaming multiprocessors required.
	// 0 means no preference — the system decides based on workload type.
	// Relevant for MIG partitioning where SM count is fixed per slice.
	MinSMs int

	// Precision is the compute precision the workload will use.
	// Affects throughput estimation: FP16/BF16 jobs get ~2x the TFLOPS of FP32.
	Precision ComputePrecision

	// MinGPUCount is the number of GPUs required (for distributed training).
	// 1 for single-GPU jobs. >1 triggers multi-GPU placement with NVLink awareness.
	MinGPUCount int

	// MaxPowerWatts is the maximum power budget for this workload.
	// 0 means no limit. Used by the policy engine for thermal headroom checks.
	MaxPowerWatts int
}

// ---------------------------------------------------------------------------
// Model Spec — what the prediction engine needs
// ---------------------------------------------------------------------------
//
// The prediction engine uses these fields to estimate GPU utilization,
// memory usage, and runtime before a job is placed. Without this data,
// the prediction engine falls back to workload-type-based heuristics.
// ---------------------------------------------------------------------------

type ModelSpec struct {
	// ModelName is a human-readable identifier (e.g., "resnet50", "llama-70b").
	// Used for lookup in the prediction engine's historical data.
	ModelName string

	// ModelSizeBytes is the total parameter size in bytes.
	// For a 7B parameter FP16 model: 7e9 * 2 = 14GB.
	ModelSizeBytes int64

	// BatchSize is the training/inference batch size.
	// Directly affects VRAM usage and SM occupancy.
	BatchSize int

	// SequenceLength is the input sequence length (for transformer models).
	// 0 for non-sequence models (CNNs, etc.).
	SequenceLength int
}

// ---------------------------------------------------------------------------
// Container Spec — what the agent needs to run
// ---------------------------------------------------------------------------
//
// The node agent receives this spec and creates a Kubernetes pod
// (or directly launches a container) with the appropriate GPU configuration.
// ---------------------------------------------------------------------------

type ContainerSpec struct {
	// Image is the container image to run (e.g., "pytorch/pytorch:2.1.0-cuda12.1").
	Image string

	// Command is the entrypoint command (e.g., ["python", "train.py"]).
	Command []string

	// Args are arguments passed to the command.
	Args []string

	// EnvVars are environment variables injected into the container.
	// The agent also injects CUDA-specific vars (CUDA_VISIBLE_DEVICES,
	// CUDA_MPS_PIPE_DIRECTORY, etc.) — those should not be set here.
	EnvVars map[string]string

	// WorkingDir is the working directory inside the container.
	WorkingDir string
}

// ---------------------------------------------------------------------------
// Job — the core domain entity
// ---------------------------------------------------------------------------
//
// A Job represents a GPU workload submitted to Veltrix. It flows through
// the entire control plane:
//
//   1. API Server creates it (status: Pending)
//   2. Scheduler picks it from the queue (status: Scheduled)
//   3. Prediction Engine estimates resource usage
//   4. Placement Engine selects a node/GPU (status: Placing)
//   5. Node Agent configures GPU and launches container (status: Running)
//   6. Feedback Controller monitors execution
//   7. Job completes or fails (status: Completed/Failed)
//
// Jobs are immutable once completed. Active jobs can be cancelled or preempted.
// ---------------------------------------------------------------------------

type Job struct {
	// --- Identity ---

	// ID is a globally unique identifier (UUID v4).
	ID string

	// Name is a human-readable name provided by the user.
	// Must be unique within a (Tenant, Namespace) scope.
	Name string

	// --- Ownership ---

	// Tenant identifies the team/org that owns this job.
	// Used by the policy engine for quota enforcement and isolation rules.
	Tenant string

	// Namespace is the Kubernetes namespace where the job's pod will run.
	Namespace string

	// --- Workload Definition ---

	// WorkloadType classifies the job for scheduling decisions.
	WorkloadType WorkloadType

	// Framework is the ML framework (e.g., "pytorch", "deepspeed", "tensorrt").
	// Used by the prediction engine and by the agent for runtime optimization.
	Framework string

	// Model contains metadata about the ML model being run.
	// Used by the prediction engine. Nil if not applicable (e.g., preprocessing).
	Model *ModelSpec

	// Container defines what to actually run on the GPU.
	Container ContainerSpec

	// --- Resource Requirements ---

	// Resources defines the minimum GPU resources this job needs.
	// May be overridden by the prediction engine's estimates.
	Resources ResourceRequirements

	// --- Lifecycle State ---

	// Status is the current lifecycle state. See JobStatus constants.
	Status JobStatus

	// Priority determines queue ordering. Higher = scheduled first.
	Priority JobPriority

	// FailureReason is set when Status == JobStatusFailed.
	// Contains a human-readable description of why the job failed.
	FailureReason string

	// --- Timestamps ---

	SubmittedAt time.Time  // When the job was created via the API.
	ScheduledAt *time.Time // When the scheduler accepted the job.
	StartedAt   *time.Time // When the container started running on the GPU.
	CompletedAt *time.Time // When the job reached a terminal state.
}
