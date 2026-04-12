// Package scheduler implements the Veltrix job scheduler.
//
// The scheduler is the first control plane service a job encounters after
// the API server. It manages a priority queue of pending jobs and orchestrates
// the scheduling pipeline:
//
//   1. Admit: validate the job against quotas and cluster capacity
//   2. Enqueue: insert into the priority queue
//   3. Dequeue: pop the highest-priority job when resources are available
//   4. Predict: call the prediction engine to estimate resource requirements
//   5. Place: delegate to the placement engine for node/GPU selection
//   6. Dispatch: send placement instructions to the node agent via the queue
//
// The scheduler runs as a continuous loop. On each iteration, it checks
// for available resources and attempts to schedule the next job in the queue.
// It also listens for job completion events to trigger re-scheduling of
// preempted or waiting jobs.
//
// Concurrency: the scheduler is single-threaded by design. All state
// mutations go through a single goroutine to avoid race conditions on
// the priority queue. External callers interact via the Scheduler interface,
// which is safe for concurrent use.
package scheduler

import (
	"context"

	"veltrix/internal/controlplane/domain"
)

// ---------------------------------------------------------------------------
// Scheduler — the public interface
// ---------------------------------------------------------------------------
//
// All control plane components and the API server interact with the scheduler
// through this interface. Implementations must be safe for concurrent use.
// ---------------------------------------------------------------------------

type Scheduler interface {
	// Submit adds a new job to the scheduling queue.
	//
	// The job's status must be Pending. The scheduler validates the job,
	// checks tenant quotas (via the policy engine), and enqueues it.
	// Returns an error if the job is invalid or the tenant has exceeded quotas.
	//
	// This is an asynchronous operation — the job is queued, not immediately
	// scheduled. The caller should watch for status changes via events.
	Submit(ctx context.Context, job *domain.Job) error

	// Cancel removes a job from the queue or requests termination if running.
	//
	// If the job is Pending/Scheduled: removed from queue, status → Cancelled.
	// If the job is Running: the agent is instructed to stop the container.
	// If the job is in a terminal state: returns an error (cannot cancel).
	Cancel(ctx context.Context, jobID string) error

	// UpdatePriority changes a job's priority in the queue.
	//
	// Only valid for jobs in Pending or Scheduled status. The job is
	// re-positioned in the priority queue. Returns an error if the job
	// is already running or completed.
	UpdatePriority(ctx context.Context, jobID string, priority domain.JobPriority) error

	// GetQueueDepth returns the number of jobs waiting in the queue.
	// Used by the API server for dashboard metrics.
	GetQueueDepth(ctx context.Context) (int, error)

	// GetQueuedJobs returns all jobs currently in the queue, ordered by priority.
	// Used by the API server for the job scheduling UI in Grafana.
	GetQueuedJobs(ctx context.Context) ([]*domain.Job, error)

	// Start begins the scheduler's main loop.
	// Blocks until the context is cancelled. Should be called in a goroutine.
	Start(ctx context.Context) error

	// Stop gracefully shuts down the scheduler.
	// Waits for the current scheduling cycle to complete.
	Stop(ctx context.Context) error
}

// ---------------------------------------------------------------------------
// PredictionClient — interface to the prediction engine (Python gRPC service)
// ---------------------------------------------------------------------------
//
// The scheduler calls the prediction engine before placement to get
// resource estimates. This interface abstracts the gRPC call so the
// scheduler can be tested with a mock prediction engine.
// ---------------------------------------------------------------------------

type PredictionClient interface {
	// Predict estimates the GPU resource usage for a job.
	//
	// Inputs: the job's workload type, framework, model spec, and resource requests.
	// Outputs: predicted VRAM usage, GPU utilization, and runtime.
	//
	// If the prediction engine is unavailable, the scheduler falls back to
	// the job's declared resource requirements (conservative estimate).
	Predict(ctx context.Context, job *domain.Job) (*domain.ResourcePrediction, error)
}

// ---------------------------------------------------------------------------
// Default implementation
// ---------------------------------------------------------------------------

// DefaultScheduler is the production implementation of the Scheduler interface.
type DefaultScheduler struct {
	// prediction is the client to the prediction engine (Python gRPC service).
	prediction PredictionClient

	// TODO: these dependencies will be injected via constructor
	// placement placement.PlacementEngine
	// policy    policy.PolicyEngine
	// jobs      repository.JobRepository
	// queue     queue.Queue
	// cache     cache.Cache
}

// NewScheduler creates a new DefaultScheduler with all dependencies.
//
// Dependencies are injected via this constructor — the scheduler never
// creates its own database connections, queue clients, etc.
func NewScheduler(prediction PredictionClient) *DefaultScheduler {
	return &DefaultScheduler{
		prediction: prediction,
	}
}

func (s *DefaultScheduler) Submit(ctx context.Context, job *domain.Job) error {
	// TODO: implementation
	// 1. Validate job fields
	// 2. Check tenant quota via policy engine
	// 3. Persist job to state store (status: Pending)
	// 4. Enqueue in priority queue
	// 5. Publish jobs.submitted event
	return nil
}

func (s *DefaultScheduler) Cancel(ctx context.Context, jobID string) error {
	// TODO: implementation
	// 1. Look up job in state store
	// 2. If pending/scheduled: remove from queue, update status
	// 3. If running: publish cancel instruction to agent via queue
	// 4. If terminal: return error
	return nil
}

func (s *DefaultScheduler) UpdatePriority(ctx context.Context, jobID string, priority domain.JobPriority) error {
	// TODO: implementation
	// 1. Look up job in state store
	// 2. Validate job is in queue (Pending/Scheduled)
	// 3. Update priority in queue (remove + re-insert)
	// 4. Update priority in state store
	return nil
}

func (s *DefaultScheduler) GetQueueDepth(ctx context.Context) (int, error) {
	// TODO: implementation
	return 0, nil
}

func (s *DefaultScheduler) GetQueuedJobs(ctx context.Context) ([]*domain.Job, error) {
	// TODO: implementation
	return nil, nil
}

func (s *DefaultScheduler) Start(ctx context.Context) error {
	// TODO: implementation
	// Main scheduling loop:
	// for {
	//   select {
	//   case <-ctx.Done(): return
	//   default:
	//     1. Check for available GPU resources (from cache)
	//     2. Dequeue highest-priority job
	//     3. Call prediction engine
	//     4. Call placement engine
	//     5. Publish placement decision to agent
	//     6. Update job status to Scheduled → Placing
	//   }
	// }
	return nil
}

func (s *DefaultScheduler) Stop(ctx context.Context) error {
	// TODO: implementation
	// Signal the main loop to stop and wait for graceful shutdown
	return nil
}
