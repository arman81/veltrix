package domain

import "time"

// ---------------------------------------------------------------------------
// Node — represents a physical or virtual machine in the GPU cluster
// ---------------------------------------------------------------------------
//
// A node is a Kubernetes worker node with one or more GPUs. The Veltrix
// agent runs as a DaemonSet on every GPU node, reporting telemetry and
// executing placement decisions.
//
// The placement engine scores nodes based on:
//   - Available GPU resources (VRAM, SMs, strategy compatibility)
//   - Thermal headroom (can the node sustain additional power draw?)
//   - Fragmentation (prefer nodes where remaining capacity is usable)
//   - Topology (NVLink connectivity for multi-GPU jobs)
//
// Node state is the aggregate of its GPUs plus host-level resources.
// ---------------------------------------------------------------------------

// NodeStatus represents the operational state of a node.
type NodeStatus string

const (
	// NodeStatusReady — node is online, agent is reporting, GPUs are functional.
	NodeStatusReady NodeStatus = "ready"

	// NodeStatusNotReady — node is registered but the agent is not responding.
	// GPUs on this node will not receive new placements.
	NodeStatusNotReady NodeStatus = "not_ready"

	// NodeStatusDraining — node is being drained for maintenance.
	// Running jobs will complete but no new jobs will be placed on any GPU.
	NodeStatusDraining NodeStatus = "draining"

	// NodeStatusCordoned — node is manually excluded from scheduling.
	// Similar to `kubectl cordon`. Existing jobs continue running.
	NodeStatusCordoned NodeStatus = "cordoned"
)

// ---------------------------------------------------------------------------
// Node Resources — host-level (non-GPU) resources
// ---------------------------------------------------------------------------
//
// While Veltrix primarily optimizes GPU utilization, host resources can be
// a bottleneck. A node with 8 GPUs but only 64GB RAM may not be able to
// run 8 data-loading processes simultaneously.
// ---------------------------------------------------------------------------

type NodeResources struct {
	// CPUCores is the total number of CPU cores on the node.
	CPUCores int

	// CPUAvailableCores is the number of CPU cores not reserved by other workloads.
	CPUAvailableCores int

	// MemoryBytes is the total host RAM in bytes.
	MemoryBytes int64

	// MemoryAvailableBytes is the free host RAM in bytes.
	MemoryAvailableBytes int64

	// StorageBytes is the total local storage (NVMe) in bytes.
	StorageBytes int64

	// StorageAvailableBytes is the free local storage in bytes.
	StorageAvailableBytes int64
}

// ---------------------------------------------------------------------------
// Node Topology — interconnect and GPU connectivity
// ---------------------------------------------------------------------------
//
// For multi-GPU jobs (distributed training), the placement engine needs to
// know which GPUs are connected via NVLink vs PCIe. NVLink provides
// 600 GB/s bandwidth (A100) vs ~32 GB/s for PCIe Gen4 x16.
//
// This is critical for jobs using NCCL for all-reduce operations.
// ---------------------------------------------------------------------------

type NodeTopology struct {
	// NVLinkPairs lists GPU device index pairs connected via NVLink.
	// Example: [[0,1], [2,3]] means GPU 0↔1 and GPU 2↔3 have NVLink.
	NVLinkPairs [][2]int

	// NUMAZones maps GPU device indices to their NUMA zone.
	// GPUs in the same NUMA zone share memory controllers with nearby CPUs.
	// Example: {0: 0, 1: 0, 2: 1, 3: 1} means GPUs 0,1 are in NUMA 0.
	NUMAZones map[int]int
}

// ---------------------------------------------------------------------------
// Node — the domain entity
// ---------------------------------------------------------------------------

type Node struct {
	// --- Identity ---

	// ID is a globally unique identifier for this node (UUID v4).
	ID string

	// Hostname is the node's hostname as reported by the OS.
	Hostname string

	// KubernetesName is the node name in the Kubernetes cluster.
	// Used when creating pods for job execution.
	KubernetesName string

	// --- State ---

	// Status is the operational state of the node.
	Status NodeStatus

	// GPUs lists all GPUs installed on this node.
	// Ordered by device index.
	GPUs []GPU

	// Resources is the host-level (non-GPU) resource snapshot.
	Resources NodeResources

	// Topology describes GPU interconnect and NUMA layout.
	Topology NodeTopology

	// --- Agent Metadata ---

	// AgentVersion is the version of the Veltrix agent running on this node.
	AgentVersion string

	// LastHeartbeat is the timestamp of the last agent heartbeat.
	// If this is older than the heartbeat timeout (e.g., 30s), the node
	// transitions to NodeStatusNotReady.
	LastHeartbeat time.Time

	// --- Labels ---

	// Labels are user-defined key-value pairs for scheduling affinity.
	// Example: {"gpu-generation": "ampere", "region": "us-east-1"}
	// The policy engine can use labels in placement constraints.
	Labels map[string]string
}
