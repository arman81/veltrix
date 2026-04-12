// Package placement implements the Veltrix placement engine.
//
// The placement engine answers two questions for every job:
//   1. WHICH node should this job run on?
//   2. HOW should the GPU be configured? (full GPU, MIG, or MPS)
//
// It does this by scoring every eligible node and selecting the best one.
//
// Node scoring considers:
//   - Available VRAM (can the node fit this job?)
//   - Current utilization (prefer less-loaded nodes)
//   - Thermal headroom (can the node sustain the additional power draw?)
//   - Fragmentation (prefer nodes where remaining capacity is still usable)
//   - Topology (prefer NVLink-connected GPUs for multi-GPU jobs)
//   - Policy constraints (isolation, affinity, co-location rules)
//
// Strategy selection logic:
//   - If job requires > 50% VRAM or policy requires isolation → full GPU
//   - If job fits a MIG profile and MIG is already active → MIG partition
//   - If job is inference, low utilization, and co-location allowed → MPS
//   - Fallback: full GPU (safest option)
//
// The placement engine is stateless — it reads current cluster state from
// Redis (hot cache) and Postgres (authoritative), makes a decision, and
// returns it. It does NOT modify cluster state directly.
package placement

import (
	"context"

	"veltrix/internal/controlplane/domain"
)

// ---------------------------------------------------------------------------
// PlacementEngine — the public interface
// ---------------------------------------------------------------------------

type PlacementEngine interface {
	// Place selects a node and GPU strategy for the given job.
	//
	// The prediction must be provided — the placement engine uses predicted
	// VRAM and utilization to score nodes and select a strategy.
	//
	// Returns a fully populated Placement (status: Decided) or an error if
	// no suitable node/GPU is available. In that case, the scheduler should
	// keep the job in the queue and retry later.
	Place(ctx context.Context, job *domain.Job, prediction *domain.ResourcePrediction) (*domain.Placement, error)

	// ScoreNodes evaluates all eligible nodes for a job and returns them
	// ranked by suitability (highest score first).
	//
	// This is exposed separately for debugging and the Grafana UI — operators
	// can see why a particular node was chosen or rejected.
	ScoreNodes(ctx context.Context, job *domain.Job, prediction *domain.ResourcePrediction) ([]NodeScore, error)
}

// ---------------------------------------------------------------------------
// NodeScore — the result of scoring a single node
// ---------------------------------------------------------------------------

type NodeScore struct {
	// NodeID is the node being scored.
	NodeID string

	// Score is the overall suitability score (0.0–1.0). Higher is better.
	Score float64

	// SelectedGPUID is the best GPU on this node for the job.
	SelectedGPUID string

	// SelectedStrategy is the recommended GPU strategy for this node.
	SelectedStrategy domain.GPUStrategy

	// Eligible indicates whether the node passes all hard constraints.
	// A node can be ineligible (Eligible=false) but still scored, for debugging.
	Eligible bool

	// RejectReason explains why the node was rejected (if Eligible=false).
	// Example: "insufficient VRAM: 10GB available, 40GB required"
	RejectReason string

	// Breakdown shows how the score was computed.
	Breakdown ScoreBreakdown
}

// ScoreBreakdown decomposes the overall score into individual factors.
// Each factor is a score from 0.0–1.0, weighted and summed to produce
// the overall score. Weights are configurable.
type ScoreBreakdown struct {
	// VRAMScore — higher when more VRAM is available relative to the job's needs.
	// 1.0 = GPU has 2x+ the required VRAM. 0.0 = exactly at the limit.
	VRAMScore float64

	// UtilizationScore — higher when the GPU is less loaded.
	// 1.0 = GPU idle. 0.0 = GPU at 100% utilization.
	UtilizationScore float64

	// ThermalScore — higher when the node has more thermal/power headroom.
	// 1.0 = GPU at idle power. 0.0 = GPU at TDP.
	ThermalScore float64

	// FragmentationScore — higher when placing this job leaves the remaining
	// VRAM in a usable chunk (not slivers that can't fit any job).
	FragmentationScore float64

	// TopologyScore — higher when GPU placement optimizes for interconnect.
	// 1.0 = NVLink-connected GPUs for multi-GPU job. 0.0 = PCIe only.
	TopologyScore float64

	// AffinityScore — higher when the node matches affinity rules.
	// 1.0 = all preferred labels match. 0.0 = no preferred labels match.
	AffinityScore float64
}

// ---------------------------------------------------------------------------
// Default implementation
// ---------------------------------------------------------------------------

// DefaultPlacementEngine is the production implementation.
type DefaultPlacementEngine struct {
	// TODO: dependencies will be injected via constructor
	// policy policy.PolicyEngine
	// nodes  repository.NodeRepository
	// gpus   repository.GPURepository
	// cache  cache.GPUStateCache
}

// NewPlacementEngine creates a new DefaultPlacementEngine.
func NewPlacementEngine() *DefaultPlacementEngine {
	return &DefaultPlacementEngine{}
}

func (e *DefaultPlacementEngine) Place(ctx context.Context, job *domain.Job, prediction *domain.ResourcePrediction) (*domain.Placement, error) {
	// TODO: implementation
	// 1. Get all nodes in Ready state
	// 2. Filter by hard constraints (VRAM, policy, affinity)
	// 3. Score remaining nodes
	// 4. Select the highest-scoring eligible node
	// 5. Select the best GPU on that node
	// 6. Decide strategy (full GPU / MIG / MPS)
	// 7. Create and return Placement
	return nil, nil
}

func (e *DefaultPlacementEngine) ScoreNodes(ctx context.Context, job *domain.Job, prediction *domain.ResourcePrediction) ([]NodeScore, error) {
	// TODO: implementation
	// 1. Get all nodes
	// 2. For each node: check eligibility, compute score breakdown
	// 3. Sort by score descending
	// 4. Return all scores (including ineligible nodes, for debugging)
	return nil, nil
}
