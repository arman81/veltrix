// Package policy implements the Veltrix policy engine.
//
// The policy engine is the gatekeeper of the control plane. It is consulted
// before every scheduling and placement decision to enforce organizational
// rules, safety limits, and SLA guarantees.
//
// The policy engine does NOT make scheduling decisions — it only answers
// yes/no questions:
//   - "Can this job be admitted?" (quota check)
//   - "Can this job run on this GPU?" (isolation check)
//   - "Can these two jobs share a GPU?" (co-location check)
//   - "Does this placement meet the tenant's SLA?" (SLA check)
//   - "Does this node match the job's affinity rules?" (affinity check)
//
// Policies are stored in Postgres, cached in Redis, and evaluated in-memory.
// They are loaded at startup and refreshed when the cache is invalidated
// (e.g., when an admin creates or updates a policy via the API).
//
// Conflict resolution: when two policies conflict, the one with the higher
// Priority value wins. System policies (safety limits) always have priority 1000+.
package policy

import (
	"context"

	"veltrix/internal/controlplane/domain"
)

// ---------------------------------------------------------------------------
// PolicyEngine — the public interface
// ---------------------------------------------------------------------------

type PolicyEngine interface {
	// CheckAdmission evaluates whether a job can be admitted to the queue.
	//
	// Checks:
	//   - Tenant quota (max concurrent jobs, max GPUs, daily GPU-hours)
	//   - Cluster capacity (is there any theoretical possibility of scheduling?)
	//
	// Called by the scheduler during Submit().
	CheckAdmission(ctx context.Context, job *domain.Job) (*Decision, error)

	// CheckPlacement evaluates whether a specific job can run on a specific GPU.
	//
	// Checks:
	//   - Isolation rules (tenant exclusivity, workload type, framework)
	//   - Co-location rules (max jobs per GPU, utilization threshold)
	//   - Affinity rules (required/preferred labels, NVLink requirements)
	//   - SLA rules (latency, throughput guarantees)
	//
	// Called by the placement engine for every candidate GPU.
	CheckPlacement(ctx context.Context, job *domain.Job, gpu *domain.GPU, node *domain.Node) (*Decision, error)

	// CheckCoLocation evaluates whether two specific jobs can share a GPU.
	//
	// This is a pairwise check. For a GPU with N existing jobs, the placement
	// engine calls this for each existing job paired with the candidate job.
	//
	// Checks:
	//   - Workload type compatibility
	//   - Tenant isolation
	//   - Framework conflicts
	//   - Combined utilization threshold
	CheckCoLocation(ctx context.Context, candidate *domain.Job, existing *domain.Job) (*Decision, error)

	// EvaluatePreemption determines whether a high-priority job should preempt
	// a lower-priority job to free resources.
	//
	// Returns a list of jobs that can be preempted, ordered by preemption cost
	// (prefer preempting jobs that have run the least, or have the lowest priority).
	EvaluatePreemption(ctx context.Context, candidate *domain.Job, running []*domain.Job) ([]*PreemptionCandidate, error)

	// LoadPolicies refreshes the in-memory policy cache from the state store.
	// Called at startup and when policies are created/updated via the API.
	LoadPolicies(ctx context.Context) error
}

// ---------------------------------------------------------------------------
// Decision — the result of a policy evaluation
// ---------------------------------------------------------------------------

type Decision struct {
	// Allowed is true if the policy check passed.
	Allowed bool

	// Reason explains why the decision was made.
	// For denials: "tenant-A quota exceeded: 16/16 GPUs in use"
	// For approvals: "all policies satisfied"
	Reason string

	// ViolatedPolicy is the ID of the policy that caused a denial.
	// Empty string if Allowed is true.
	ViolatedPolicy string
}

// ---------------------------------------------------------------------------
// PreemptionCandidate — a job that can be evicted to free resources
// ---------------------------------------------------------------------------

type PreemptionCandidate struct {
	// Job is the running job that could be preempted.
	Job *domain.Job

	// Cost is the estimated cost of preemption (0.0–1.0).
	// Lower cost = better candidate for preemption.
	// Factors: priority difference, runtime already consumed, SLA impact.
	Cost float64

	// Reason explains why this job is a valid preemption candidate.
	Reason string
}

// ---------------------------------------------------------------------------
// Default implementation
// ---------------------------------------------------------------------------

// DefaultPolicyEngine is the production implementation.
type DefaultPolicyEngine struct {
	// policies is the in-memory cache of all active policies.
	policies []*domain.Policy

	// TODO: dependencies will be injected via constructor
	// policyRepo repository.PolicyRepository
	// cache      cache.Cache
}

// NewPolicyEngine creates a new DefaultPolicyEngine.
func NewPolicyEngine() *DefaultPolicyEngine {
	return &DefaultPolicyEngine{
		policies: make([]*domain.Policy, 0),
	}
}

func (e *DefaultPolicyEngine) CheckAdmission(ctx context.Context, job *domain.Job) (*Decision, error) {
	// TODO: implementation
	// 1. Load quota policies for the job's tenant
	// 2. Count tenant's current active jobs and GPU usage
	// 3. Check against quota limits
	// 4. Return allowed/denied with reason
	return &Decision{Allowed: true, Reason: "all policies satisfied"}, nil
}

func (e *DefaultPolicyEngine) CheckPlacement(ctx context.Context, job *domain.Job, gpu *domain.GPU, node *domain.Node) (*Decision, error) {
	// TODO: implementation
	// 1. Check isolation policies (tenant exclusive, workload type, framework)
	// 2. Check affinity policies (required labels, NVLink)
	// 3. Check SLA policies (latency, throughput)
	// 4. Check co-location policies (max jobs, utilization ceiling)
	// 5. Return first violation found, or allowed
	return &Decision{Allowed: true, Reason: "all policies satisfied"}, nil
}

func (e *DefaultPolicyEngine) CheckCoLocation(ctx context.Context, candidate *domain.Job, existing *domain.Job) (*Decision, error) {
	// TODO: implementation
	// 1. Check if workload types are compatible
	// 2. Check tenant isolation rules
	// 3. Check framework conflicts
	// 4. Return allowed/denied
	return &Decision{Allowed: true, Reason: "all policies satisfied"}, nil
}

func (e *DefaultPolicyEngine) EvaluatePreemption(ctx context.Context, candidate *domain.Job, running []*domain.Job) ([]*PreemptionCandidate, error) {
	// TODO: implementation
	// 1. Filter running jobs by lower priority than candidate
	// 2. Score each by preemption cost
	// 3. Sort by cost ascending (cheapest to preempt first)
	// 4. Return candidates
	return nil, nil
}

func (e *DefaultPolicyEngine) LoadPolicies(ctx context.Context) error {
	// TODO: implementation
	// 1. Fetch all enabled policies from the policy repository
	// 2. Sort by priority descending
	// 3. Replace in-memory cache atomically
	return nil
}
