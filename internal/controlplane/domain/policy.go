package domain

// ---------------------------------------------------------------------------
// Policy — rules that constrain scheduling and placement decisions
// ---------------------------------------------------------------------------
//
// The policy engine evaluates policies before every placement decision.
// Policies define:
//   - Isolation: which workloads CANNOT share a GPU
//   - Co-location: which workloads CAN (or should) share a GPU
//   - SLA: latency, throughput, and availability guarantees per tenant
//   - Quota: maximum GPU-hours or concurrent jobs per tenant
//   - Affinity: node/GPU selection preferences
//
// Policies have priorities. When policies conflict, the higher-priority
// policy wins. System policies (e.g., safety limits) have the highest priority.
//
// Policies are stored in Postgres and cached in Redis. They are loaded
// at scheduler startup and refreshed on change.
// ---------------------------------------------------------------------------

// PolicyType categorizes the kind of constraint a policy enforces.
type PolicyType string

const (
	// PolicyTypeIsolation — prevents incompatible workloads from sharing GPUs.
	// Example: "training jobs from tenant-A must not share GPUs with any other tenant."
	PolicyTypeIsolation PolicyType = "isolation"

	// PolicyTypeCoLocation — defines rules for when GPU sharing is allowed or preferred.
	// Example: "inference jobs with <20% utilization should be co-located via MPS."
	PolicyTypeCoLocation PolicyType = "co_location"

	// PolicyTypeSLA — defines performance guarantees for a tenant or workload class.
	// Example: "inference jobs for tenant-B must have p99 latency < 50ms."
	PolicyTypeSLA PolicyType = "sla"

	// PolicyTypeQuota — limits resource consumption per tenant.
	// Example: "tenant-C can use at most 16 GPUs concurrently."
	PolicyTypeQuota PolicyType = "quota"

	// PolicyTypeAffinity — specifies node or GPU preferences.
	// Example: "training jobs should prefer nodes with NVLink-connected GPUs."
	PolicyTypeAffinity PolicyType = "affinity"
)

// ---------------------------------------------------------------------------
// Isolation Rule — defines what cannot be co-located
// ---------------------------------------------------------------------------

type IsolationRule struct {
	// TenantExclusive means no two tenants can share the same GPU.
	// When true, a GPU allocated to tenant-A cannot run tenant-B's jobs.
	TenantExclusive bool

	// WorkloadTypeExclusive lists workload types that require exclusive GPU access.
	// Example: ["training"] means training jobs always get a full GPU.
	WorkloadTypeExclusive []WorkloadType

	// FrameworkExclusive lists frameworks that cannot share GPUs.
	// Example: ["deepspeed"] because DeepSpeed uses custom CUDA kernels
	// that may conflict with other processes.
	FrameworkExclusive []string
}

// ---------------------------------------------------------------------------
// Co-Location Rule — defines what can be shared and how
// ---------------------------------------------------------------------------

type CoLocationRule struct {
	// MaxJobsPerGPU is the maximum number of jobs that can share a single GPU.
	// Applies to MPS strategy only. MIG instances are counted separately.
	MaxJobsPerGPU int

	// MaxUtilizationPercent is the threshold above which no more jobs
	// should be co-located onto the GPU. Prevents oversubscription.
	MaxUtilizationPercent float64

	// PreferredStrategy is the default sharing strategy when co-locating.
	// The placement engine may override this based on workload characteristics.
	PreferredStrategy GPUStrategy

	// CompatibleWorkloadTypes lists which workload type pairs can share a GPU.
	// Example: [["inference", "inference"]] means inference can share with inference,
	// but not with training.
	CompatibleWorkloadTypes [][2]WorkloadType
}

// ---------------------------------------------------------------------------
// SLA Rule — performance guarantees
// ---------------------------------------------------------------------------

type SLARule struct {
	// MaxLatencyMs is the maximum acceptable p99 latency in milliseconds.
	// Applies to inference workloads. 0 means no latency constraint.
	MaxLatencyMs int

	// MinThroughput is the minimum acceptable throughput (inferences/sec or samples/sec).
	// 0 means no throughput constraint.
	MinThroughput float64

	// MaxQueueTimeSeconds is the maximum time a job can wait in the queue.
	// If exceeded, the job's priority is temporarily boosted to Critical.
	MaxQueueTimeSeconds int

	// Availability is the target uptime percentage (e.g., 99.9).
	// Affects how aggressively the system preempts this tenant's jobs.
	Availability float64
}

// ---------------------------------------------------------------------------
// Quota Rule — resource consumption limits
// ---------------------------------------------------------------------------

type QuotaRule struct {
	// MaxConcurrentJobs is the maximum number of jobs a tenant can run at once.
	MaxConcurrentJobs int

	// MaxGPUs is the maximum number of GPUs a tenant can occupy simultaneously.
	MaxGPUs int

	// MaxGPUHoursPerDay is the daily GPU-hour budget for a tenant.
	// 0 means unlimited.
	MaxGPUHoursPerDay float64
}

// ---------------------------------------------------------------------------
// Affinity Rule — scheduling preferences
// ---------------------------------------------------------------------------

type AffinityRule struct {
	// RequiredLabels are node labels that MUST be present for scheduling.
	// Example: {"gpu-generation": "ampere"} — only schedule on Ampere nodes.
	RequiredLabels map[string]string

	// PreferredLabels are node labels that SHOULD be present (soft constraint).
	// Nodes with these labels get a scoring bonus from the placement engine.
	PreferredLabels map[string]string

	// RequireNVLink means the job needs NVLink-connected GPUs.
	// Only relevant for multi-GPU jobs. Placement will fail if no NVLink
	// pairs are available.
	RequireNVLink bool
}

// ---------------------------------------------------------------------------
// Policy — the domain entity
// ---------------------------------------------------------------------------

type Policy struct {
	// --- Identity ---

	// ID is a globally unique identifier for this policy (UUID v4).
	ID string

	// Name is a human-readable policy name.
	// Example: "tenant-a-isolation", "inference-colocation-default"
	Name string

	// --- Classification ---

	// Type is the category of constraint this policy enforces.
	Type PolicyType

	// Priority determines which policy wins when two policies conflict.
	// Higher value = higher priority. System policies use 1000+.
	Priority int

	// Enabled controls whether this policy is active.
	// Disabled policies are stored but not evaluated.
	Enabled bool

	// --- Scope ---

	// Tenant is the tenant this policy applies to.
	// Empty string means the policy applies to all tenants.
	Tenant string

	// Namespace is the namespace this policy applies to.
	// Empty string means the policy applies to all namespaces.
	Namespace string

	// --- Rules (exactly one of these is set, based on Type) ---

	Isolation   *IsolationRule
	CoLocation  *CoLocationRule
	SLA         *SLARule
	Quota       *QuotaRule
	Affinity    *AffinityRule
}
