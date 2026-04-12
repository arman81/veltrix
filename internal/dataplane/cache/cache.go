// Package cache defines the caching and distributed locking interfaces for Veltrix.
//
// The cache serves two purposes:
//
// 1. HOT STATE CACHE — The placement engine needs to read GPU/node state
//    thousands of times per scheduling cycle. Reading from Postgres every time
//    would be too slow. The cache holds the latest telemetry snapshot for every
//    GPU, updated every collection interval (5s) by the feedback controller.
//
// 2. DISTRIBUTED LOCKS — When the placement engine decides to assign a job to
//    a GPU, it must prevent another scheduler instance from assigning a different
//    job to the same GPU simultaneously. Redis-based distributed locks provide
//    this mutual exclusion.
//
// Cache invariant: the cache is ALWAYS a subset of Postgres. If a cache entry
// is missing, the system falls back to Postgres. If the cache disagrees with
// Postgres, Postgres wins. The cache is a performance optimization, not a
// source of truth.
//
// TTL strategy: GPU state entries expire after 30 seconds. If no telemetry
// arrives within that window, the entry is evicted and the feedback controller
// marks the GPU as potentially offline.
package cache

import (
	"context"
	"time"

	"veltrix/internal/controlplane/domain"
)

// ---------------------------------------------------------------------------
// GPUStateCache — hot GPU telemetry for the placement engine
// ---------------------------------------------------------------------------

type GPUStateCache interface {
	// SetGPUState stores the latest telemetry for a GPU.
	// Called by the feedback controller every telemetry interval.
	// The entry expires after the configured TTL.
	SetGPUState(ctx context.Context, gpuID string, telemetry *domain.GPUTelemetry) error

	// GetGPUState retrieves the latest telemetry for a GPU.
	// Returns nil if the entry has expired or was never set.
	GetGPUState(ctx context.Context, gpuID string) (*domain.GPUTelemetry, error)

	// GetAllGPUStates retrieves the latest telemetry for all GPUs.
	// Used by the metrics handler for the cluster summary endpoint.
	// Returns a map of gpuID → telemetry.
	GetAllGPUStates(ctx context.Context) (map[string]*domain.GPUTelemetry, error)
}

// ---------------------------------------------------------------------------
// NodeStateCache — aggregated node health
// ---------------------------------------------------------------------------

type NodeStateCache interface {
	// SetNodeState stores aggregated node health (CPU, RAM, GPU summary).
	SetNodeState(ctx context.Context, nodeID string, state *NodeState) error

	// GetNodeState retrieves the latest node health snapshot.
	GetNodeState(ctx context.Context, nodeID string) (*NodeState, error)
}

// NodeState is the cached representation of a node's current health.
type NodeState struct {
	// GPUsAvailable is the number of GPUs that can accept new workloads.
	GPUsAvailable int

	// GPUsAllocated is the number of GPUs with active jobs.
	GPUsAllocated int

	// AvgGPUUtilization is the average utilization across all GPUs on the node.
	AvgGPUUtilization float64

	// CPUUsagePercent is the current host CPU usage.
	CPUUsagePercent float64

	// MemoryUsagePercent is the current host memory usage.
	MemoryUsagePercent float64

	// LastUpdated is when this snapshot was computed.
	LastUpdated time.Time
}

// ---------------------------------------------------------------------------
// LockManager — distributed locks for placement decisions
// ---------------------------------------------------------------------------
//
// Locking strategy:
//   - GPU lock: acquired before assigning a job to a GPU, released after
//     the placement is committed to Postgres.
//   - MIG lock: acquired before MIG reconfiguration (which requires GPU reset),
//     released after reconfiguration is complete. Longer TTL (minutes, not seconds).
//
// Lock implementation uses Redis SET NX with TTL (Redlock for multi-instance).
// ---------------------------------------------------------------------------

type LockManager interface {
	// AcquireGPULock attempts to acquire an exclusive lock on a GPU.
	//
	// Returns true if the lock was acquired, false if another process holds it.
	// The lock automatically expires after the TTL to prevent deadlocks
	// (e.g., if the lock holder crashes).
	//
	// Typical TTL: 10 seconds (enough to commit the placement to Postgres).
	AcquireGPULock(ctx context.Context, gpuID string, ttl time.Duration) (bool, error)

	// ReleaseGPULock releases a previously acquired GPU lock.
	// Safe to call even if the lock has already expired.
	ReleaseGPULock(ctx context.Context, gpuID string) error

	// AcquireMIGLock attempts to acquire a lock for MIG reconfiguration.
	//
	// MIG reconfiguration requires a GPU reset and affects all instances
	// on the GPU. This lock prevents any new placements on the GPU while
	// reconfiguration is in progress.
	//
	// Typical TTL: 5 minutes (MIG reconfiguration can be slow).
	AcquireMIGLock(ctx context.Context, gpuID string, ttl time.Duration) (bool, error)

	// ReleaseMIGLock releases a MIG reconfiguration lock.
	ReleaseMIGLock(ctx context.Context, gpuID string) error
}
