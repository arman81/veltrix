package domain

// ---------------------------------------------------------------------------
// GPU — represents a single physical GPU on a node
// ---------------------------------------------------------------------------
//
// A GPU is the fundamental schedulable resource in Veltrix. The placement
// engine decides how to allocate each GPU:
//   - Full GPU: one job gets exclusive access
//   - MIG: GPU is hardware-partitioned into isolated slices
//   - MPS: multiple jobs share the GPU via CUDA Multi-Process Service
//
// GPU state is updated continuously by the node agent's telemetry collector
// and cached in Redis for fast scheduling decisions.
// ---------------------------------------------------------------------------

// GPUStatus represents the operational state of a GPU.
type GPUStatus string

const (
	// GPUStatusAvailable — GPU is online and ready to accept workloads.
	GPUStatusAvailable GPUStatus = "available"

	// GPUStatusAllocated — GPU is fully allocated to one or more jobs.
	// No additional jobs can be placed without eviction or MIG reconfiguration.
	GPUStatusAllocated GPUStatus = "allocated"

	// GPUStatusDraining — GPU is being drained for maintenance or MIG reconfiguration.
	// Running jobs will complete but no new jobs will be placed.
	GPUStatusDraining GPUStatus = "draining"

	// GPUStatusOffline — GPU is unreachable or has a hardware fault.
	// The feedback controller sets this when telemetry stops arriving.
	GPUStatusOffline GPUStatus = "offline"

	// GPUStatusReconfiguring — GPU is undergoing MIG reconfiguration.
	// This requires a GPU reset and takes seconds to minutes.
	// CRITICAL: no workloads can run during reconfiguration.
	GPUStatusReconfiguring GPUStatus = "reconfiguring"
)

// GPUModel identifies the GPU hardware model.
// This determines available MIG profiles, VRAM, SM count, and compute capability.
type GPUModel string

const (
	GPUModelA100_40GB GPUModel = "a100_40gb"
	GPUModelA100_80GB GPUModel = "a100_80gb"
	GPUModelH100_80GB GPUModel = "h100_80gb"
)

// ---------------------------------------------------------------------------
// GPU Strategy — how the GPU is currently being used
// ---------------------------------------------------------------------------
//
// Each GPU operates in exactly one strategy at a time:
//   - FullGPU: exclusive access, maximum performance, no sharing
//   - MIG: hardware-partitioned, strong isolation, fixed after config
//   - MPS: software sharing, no isolation, best for inference co-location
//   - Idle: no active strategy, ready for any mode
//
// Switching between MIG and non-MIG requires a GPU reset (GPUStatusReconfiguring).
// Switching between FullGPU and MPS does NOT require a reset.
// ---------------------------------------------------------------------------

type GPUStrategy string

const (
	GPUStrategyIdle    GPUStrategy = "idle"
	GPUStrategyFullGPU GPUStrategy = "full_gpu"
	GPUStrategyMIG     GPUStrategy = "mig"
	GPUStrategyMPS     GPUStrategy = "mps"
)

// ---------------------------------------------------------------------------
// MIG Profile — a hardware partition configuration
// ---------------------------------------------------------------------------
//
// MIG profiles are defined by NVIDIA and vary by GPU model. Each profile
// specifies a fixed slice of memory and compute. An A100-80GB can be split
// into up to 7 instances (7x 1g.10gb) or fewer larger instances.
//
// CONSTRAINT: MIG profiles on a single GPU must be compatible.
// Not all combinations are valid. The MIG controller on the agent
// validates combinations before applying them.
// ---------------------------------------------------------------------------

type MIGProfile struct {
	// Name is the NVIDIA profile identifier (e.g., "1g.10gb", "3g.40gb").
	Name string

	// MemoryBytes is the VRAM allocated to this profile, in bytes.
	MemoryBytes int64

	// SMCount is the number of streaming multiprocessors in this slice.
	SMCount int

	// SMFraction is the fraction of total GPU SMs (e.g., 1.0/7.0 for 1g.10gb).
	// Useful for utilization calculations.
	SMFraction float64
}

// MIGInstance represents an active MIG partition on a GPU.
// Created when the agent applies a MIG configuration.
type MIGInstance struct {
	// ID is the unique identifier for this MIG instance (from NVML).
	ID string

	// Profile is the MIG profile this instance was created from.
	Profile MIGProfile

	// JobID is the job currently assigned to this instance.
	// Empty string if the instance is unoccupied.
	JobID string
}

// ---------------------------------------------------------------------------
// GPU Telemetry — real-time metrics from NVML
// ---------------------------------------------------------------------------
//
// Collected by the node agent every few seconds via NVML.
// Pushed to the control plane via OTel.
// Cached in Redis for fast access by the placement engine.
// Stored in Postgres for historical analysis by the feedback controller.
// ---------------------------------------------------------------------------

type GPUTelemetry struct {
	// Utilization is the GPU compute utilization percentage (0–100).
	// Measured by NVML as the fraction of time the GPU kernels are executing.
	Utilization float64

	// MemoryUsedBytes is the current GPU memory usage in bytes.
	MemoryUsedBytes int64

	// MemoryTotalBytes is the total GPU memory in bytes.
	MemoryTotalBytes int64

	// PowerDrawWatts is the current power consumption in watts.
	PowerDrawWatts float64

	// TemperatureCelsius is the GPU core temperature.
	TemperatureCelsius int

	// PCIeThroughputMBps is the current PCIe bandwidth usage in MB/s.
	// High values indicate data transfer bottlenecks.
	PCIeThroughputMBps float64

	// SMOccupancy is the fraction of SMs actively executing warps (0.0–1.0).
	// This is a more accurate measure of compute utilization than the NVML
	// utilization percentage, which only measures time-based activity.
	SMOccupancy float64
}

// ---------------------------------------------------------------------------
// GPU — the domain entity
// ---------------------------------------------------------------------------

type GPU struct {
	// --- Identity ---

	// ID is a globally unique identifier for this GPU (UUID v4).
	ID string

	// NodeID is the ID of the node this GPU is physically installed in.
	NodeID string

	// DeviceIndex is the GPU's index on the node (0-based).
	// Maps to CUDA_VISIBLE_DEVICES and NVML device index.
	DeviceIndex int

	// --- Hardware ---

	// Model identifies the GPU hardware (A100-40GB, A100-80GB, H100-80GB).
	Model GPUModel

	// VRAMBytes is the total GPU memory in bytes.
	VRAMBytes int64

	// TotalSMs is the total number of streaming multiprocessors.
	TotalSMs int

	// MIGCapable indicates whether this GPU supports Multi-Instance GPU.
	// A100 and H100 support MIG. Older GPUs do not.
	MIGCapable bool

	// --- Current State ---

	// Status is the operational state of the GPU.
	Status GPUStatus

	// Strategy is how the GPU is currently being used (full/MIG/MPS/idle).
	Strategy GPUStrategy

	// MIGInstances lists the active MIG partitions on this GPU.
	// Empty if Strategy != GPUStrategyMIG.
	MIGInstances []MIGInstance

	// ActiveJobIDs lists the jobs currently running on this GPU.
	// For FullGPU strategy: at most 1 job.
	// For MIG strategy: one job per MIG instance.
	// For MPS strategy: multiple jobs sharing the GPU.
	ActiveJobIDs []string

	// --- Telemetry ---

	// Telemetry is the most recent telemetry snapshot from the agent.
	// Updated every collection interval (typically 5 seconds).
	// Nil if no telemetry has been received yet.
	Telemetry *GPUTelemetry
}
