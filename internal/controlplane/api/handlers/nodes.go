package handlers

import (
	"net/http"
)

// ---------------------------------------------------------------------------
// NodeHandler — HTTP handlers for node and GPU operations
// ---------------------------------------------------------------------------
//
// These endpoints are read-only. Nodes register themselves via the agent
// heartbeat, not via the API. The API only exposes node state for
// monitoring and the Grafana cluster overview dashboard.
// ---------------------------------------------------------------------------

// NodeResponse is the JSON representation of a node in API responses.
type NodeResponse struct {
	ID             string        `json:"id"`
	Hostname       string        `json:"hostname"`
	Status         string        `json:"status"`
	GPUCount       int           `json:"gpu_count"`
	GPUsAllocated  int           `json:"gpus_allocated"`
	GPUsAvailable  int           `json:"gpus_available"`
	CPUCores       int           `json:"cpu_cores"`
	MemoryBytes    int64         `json:"memory_bytes"`
	AgentVersion   string        `json:"agent_version"`
	LastHeartbeat  string        `json:"last_heartbeat"`
	Labels         map[string]string `json:"labels,omitempty"`
	GPUs           []GPUResponse `json:"gpus,omitempty"` // Only included in single-node GET
}

// GPUResponse is the JSON representation of a GPU in API responses.
type GPUResponse struct {
	ID          string  `json:"id"`
	DeviceIndex int     `json:"device_index"`
	Model       string  `json:"model"`
	VRAMBytes   int64   `json:"vram_bytes"`
	Status      string  `json:"status"`
	Strategy    string  `json:"strategy"`
	Utilization float64 `json:"utilization"`
	MemoryUsed  int64   `json:"memory_used_bytes"`
	Temperature int     `json:"temperature_celsius"`
	PowerDraw   float64 `json:"power_draw_watts"`
	ActiveJobs  int     `json:"active_jobs"`
}

// ListNodesResponse is the JSON response for GET /api/v1/nodes.
type ListNodesResponse struct {
	Nodes      []NodeResponse `json:"nodes"`
	TotalCount int            `json:"total_count"`
}

type NodeHandler struct {
	// TODO: dependencies will be injected via constructor
	// nodes repository.NodeRepository
	// gpus  repository.GPURepository
	// cache cache.Cache
}

// NewNodeHandler creates a new NodeHandler.
func NewNodeHandler() *NodeHandler {
	return &NodeHandler{}
}

// List handles GET /api/v1/nodes — list all nodes in the cluster.
//
// Query parameters:
//   - status: filter by node status (e.g., ?status=ready)
//   - label: filter by label (e.g., ?label=gpu-generation:ampere)
func (h *NodeHandler) List(w http.ResponseWriter, r *http.Request) {
	// TODO: implementation
	// 1. Parse filter query parameters
	// 2. Query node repository
	// 3. For each node, aggregate GPU summary (count, allocated, available)
	// 4. Convert to ListNodesResponse
	// 5. Return 200
	writeError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "not implemented")
}

// Get handles GET /api/v1/nodes/:id — get a single node with full GPU details.
//
// Unlike List (which returns summary GPU counts), this endpoint returns
// the full GPU state for each GPU on the node, including telemetry.
func (h *NodeHandler) Get(w http.ResponseWriter, r *http.Request) {
	_ = r.PathValue("id")
	// TODO: implementation
	// 1. Look up node by ID
	// 2. Load all GPUs for this node (with latest telemetry from cache)
	// 3. Convert to NodeResponse with GPUs populated
	// 4. Return 200, or 404 if not found
	writeError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "not implemented")
}

// ListGPUs handles GET /api/v1/nodes/:id/gpus — list GPUs on a specific node.
//
// Returns detailed GPU state including MIG instances, active jobs, and
// real-time telemetry. Used by the Grafana cluster overview to render
// per-GPU cards.
func (h *NodeHandler) ListGPUs(w http.ResponseWriter, r *http.Request) {
	_ = r.PathValue("id")
	// TODO: implementation
	// 1. Look up node by ID (return 404 if not found)
	// 2. Load all GPUs for this node
	// 3. Enrich with latest telemetry from Redis cache
	// 4. Convert to []GPUResponse
	// 5. Return 200
	writeError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "not implemented")
}
