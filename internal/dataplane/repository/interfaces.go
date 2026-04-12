// Package repository defines the data access interfaces for Veltrix.
//
// These interfaces abstract the state store (Postgres) from the control plane.
// Control plane services ONLY interact with data through these interfaces —
// they never import database/sql or write SQL directly.
//
// This separation provides:
//   - Testability: control plane tests use in-memory mocks, not a real DB
//   - Swappability: Postgres can be replaced without touching business logic
//   - Clarity: data access patterns are explicit in the interface
//
// Each interface maps to a single domain aggregate:
//   - JobRepository: CRUD for jobs
//   - NodeRepository: CRUD for nodes
//   - GPURepository: CRUD for GPUs
//   - PlacementRepository: CRUD for placements
//   - MetricsRepository: write metrics, query aggregations
//   - PolicyRepository: CRUD for policies
//
// Implementation is in the postgres subpackage.
package repository

import (
	"context"
	"time"

	"veltrix/internal/controlplane/domain"
)

// ---------------------------------------------------------------------------
// JobRepository — persistence for jobs
// ---------------------------------------------------------------------------

// JobFilter defines query filters for listing jobs.
type JobFilter struct {
	// Status filters by job status. Empty means all statuses.
	Status *domain.JobStatus

	// Tenant filters by tenant. Empty means all tenants.
	Tenant string

	// Namespace filters by namespace. Empty means all namespaces.
	Namespace string

	// Cursor is the pagination cursor (job ID) from a previous query.
	Cursor string

	// Limit is the max number of results. Default 50, max 200.
	Limit int
}

type JobRepository interface {
	// Create inserts a new job. The job's ID must be set by the caller.
	Create(ctx context.Context, job *domain.Job) error

	// GetByID returns a single job by ID. Returns nil if not found.
	GetByID(ctx context.Context, id string) (*domain.Job, error)

	// List returns jobs matching the filter, ordered by priority descending.
	List(ctx context.Context, filter JobFilter) ([]*domain.Job, error)

	// UpdateStatus updates a job's status and the corresponding timestamp.
	// This is the most frequent write operation — must be fast.
	UpdateStatus(ctx context.Context, id string, status domain.JobStatus) error

	// UpdatePriority updates a job's priority.
	UpdatePriority(ctx context.Context, id string, priority domain.JobPriority) error

	// SetFailureReason sets the failure reason for a failed job.
	SetFailureReason(ctx context.Context, id string, reason string) error

	// CountByTenant returns the number of active (non-terminal) jobs for a tenant.
	// Used by the policy engine for quota checks.
	CountByTenant(ctx context.Context, tenant string) (int, error)

	// CountGPUsByTenant returns the number of GPUs currently allocated to a tenant.
	// Used by the policy engine for quota checks.
	CountGPUsByTenant(ctx context.Context, tenant string) (int, error)
}

// ---------------------------------------------------------------------------
// NodeRepository — persistence for nodes
// ---------------------------------------------------------------------------

// NodeFilter defines query filters for listing nodes.
type NodeFilter struct {
	// Status filters by node status. Nil means all statuses.
	Status *domain.NodeStatus

	// Labels filters by required labels. All specified labels must match.
	Labels map[string]string
}

type NodeRepository interface {
	// Upsert creates or updates a node. Called by the agent heartbeat handler.
	// If a node with the same ID exists, its fields are updated.
	Upsert(ctx context.Context, node *domain.Node) error

	// GetByID returns a single node by ID. Returns nil if not found.
	GetByID(ctx context.Context, id string) (*domain.Node, error)

	// List returns all nodes matching the filter.
	List(ctx context.Context, filter NodeFilter) ([]*domain.Node, error)

	// UpdateStatus updates a node's status.
	UpdateStatus(ctx context.Context, id string, status domain.NodeStatus) error

	// UpdateHeartbeat updates the last heartbeat timestamp for a node.
	UpdateHeartbeat(ctx context.Context, id string, t time.Time) error

	// GetStaleNodes returns nodes whose last heartbeat is older than the threshold.
	// Used by the feedback controller to detect offline nodes.
	GetStaleNodes(ctx context.Context, threshold time.Duration) ([]*domain.Node, error)
}

// ---------------------------------------------------------------------------
// GPURepository — persistence for GPUs
// ---------------------------------------------------------------------------

type GPURepository interface {
	// Upsert creates or updates a GPU. Called during node registration.
	Upsert(ctx context.Context, gpu *domain.GPU) error

	// GetByID returns a single GPU by ID. Returns nil if not found.
	GetByID(ctx context.Context, id string) (*domain.GPU, error)

	// ListByNode returns all GPUs on a specific node, ordered by device index.
	ListByNode(ctx context.Context, nodeID string) ([]*domain.GPU, error)

	// ListAvailable returns GPUs that can accept new workloads.
	// Filters by status=Available AND sufficient VRAM.
	// Used by the placement engine to find candidate GPUs.
	ListAvailable(ctx context.Context, minVRAMBytes int64) ([]*domain.GPU, error)

	// UpdateStatus updates a GPU's operational status.
	UpdateStatus(ctx context.Context, id string, status domain.GPUStatus) error

	// UpdateStrategy updates a GPU's current strategy (full/MIG/MPS/idle).
	UpdateStrategy(ctx context.Context, id string, strategy domain.GPUStrategy) error

	// AddActiveJob records that a job is now running on this GPU.
	AddActiveJob(ctx context.Context, gpuID string, jobID string) error

	// RemoveActiveJob records that a job has stopped running on this GPU.
	RemoveActiveJob(ctx context.Context, gpuID string, jobID string) error
}

// ---------------------------------------------------------------------------
// PlacementRepository — persistence for placement decisions
// ---------------------------------------------------------------------------

type PlacementRepository interface {
	// Create inserts a new placement.
	Create(ctx context.Context, placement *domain.Placement) error

	// GetByID returns a single placement by ID. Returns nil if not found.
	GetByID(ctx context.Context, id string) (*domain.Placement, error)

	// GetByJobID returns the active placement for a job. Returns nil if
	// the job has no active placement (pending or completed).
	GetByJobID(ctx context.Context, jobID string) (*domain.Placement, error)

	// UpdateStatus updates a placement's status and the corresponding timestamp.
	UpdateStatus(ctx context.Context, id string, status domain.PlacementStatus) error

	// ListActiveByGPU returns all active placements on a specific GPU.
	// Used by the placement engine to check co-location feasibility.
	ListActiveByGPU(ctx context.Context, gpuID string) ([]*domain.Placement, error)

	// ListActiveByNode returns all active placements on a specific node.
	ListActiveByNode(ctx context.Context, nodeID string) ([]*domain.Placement, error)
}

// ---------------------------------------------------------------------------
// MetricsRepository — persistence for GPU telemetry
// ---------------------------------------------------------------------------

type MetricsRepository interface {
	// BatchInsert writes a batch of GPU metrics. This is the highest-volume
	// write operation — called every telemetry interval per node.
	// Must be optimized for throughput (batch insert, minimal indexes).
	BatchInsert(ctx context.Context, nodeID string, metrics []GPUMetricRow) error

	// GetLatest returns the most recent metric for a GPU.
	GetLatest(ctx context.Context, gpuID string) (*GPUMetricRow, error)

	// GetRange returns metrics for a GPU within a time range.
	// Used by the feedback controller for post-job analysis.
	GetRange(ctx context.Context, gpuID string, from time.Time, to time.Time) ([]GPUMetricRow, error)

	// GetAggregated returns aggregated metrics for a GPU over a time range.
	// Computes avg, max, and p95 for each metric dimension.
	GetAggregated(ctx context.Context, gpuID string, from time.Time, to time.Time) (*AggregatedMetrics, error)
}

// GPUMetricRow is the storage representation of a single telemetry data point.
type GPUMetricRow struct {
	GPUID              string
	Utilization        float64
	MemoryUsedBytes    int64
	PowerDrawWatts     float64
	TemperatureCelsius int
	PCIeThroughputMBps float64
	SMOccupancy        float64
	Timestamp          time.Time
}

// AggregatedMetrics holds computed statistics over a time range.
type AggregatedMetrics struct {
	AvgUtilization float64
	MaxUtilization float64
	P95Utilization float64
	AvgMemoryUsed  int64
	MaxMemoryUsed  int64
	AvgPowerDraw   float64
	MaxPowerDraw   float64
	SampleCount    int
}

// ---------------------------------------------------------------------------
// PolicyRepository — persistence for policies
// ---------------------------------------------------------------------------

type PolicyRepository interface {
	// Create inserts a new policy.
	Create(ctx context.Context, policy *domain.Policy) error

	// GetByID returns a single policy by ID. Returns nil if not found.
	GetByID(ctx context.Context, id string) (*domain.Policy, error)

	// List returns all policies, ordered by priority descending.
	List(ctx context.Context) ([]*domain.Policy, error)

	// ListEnabled returns only enabled policies, ordered by priority descending.
	// Used by the policy engine at startup and on cache refresh.
	ListEnabled(ctx context.Context) ([]*domain.Policy, error)

	// Update updates a policy's fields.
	Update(ctx context.Context, policy *domain.Policy) error

	// Delete removes a policy by ID.
	Delete(ctx context.Context, id string) error
}
