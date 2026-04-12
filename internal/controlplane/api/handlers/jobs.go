// Package handlers implements the HTTP request handlers for the Veltrix API.
//
// Each handler is responsible for:
//   1. Parsing the HTTP request (path params, query params, body)
//   2. Validating input (reject malformed data before it reaches services)
//   3. Calling the appropriate control plane service
//   4. Serializing the response (domain types → JSON)
//   5. Setting the correct HTTP status code
//
// Handlers do NOT contain business logic. They are pure translation layers
// between HTTP and the control plane services.
package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"veltrix/internal/controlplane/api/middleware"
	"veltrix/internal/controlplane/domain"
	"veltrix/internal/controlplane/scheduler"
)

// ---------------------------------------------------------------------------
// Request / Response types — JSON serialization contracts
// ---------------------------------------------------------------------------
//
// These types define the API's JSON contract. They are separate from domain
// types because:
//   - Domain types have no JSON tags (serialization is a boundary concern)
//   - API types may omit internal fields (e.g., no raw FailureReason in create)
//   - API types use string enums, not Go types
//
// These types are what the Grafana app plugin sees.
// ---------------------------------------------------------------------------

// CreateJobRequest is the JSON body for POST /api/v1/jobs.
type CreateJobRequest struct {
	Name         string `json:"name"`
	WorkloadType string `json:"workload_type"` // "training", "inference", etc.
	Framework    string `json:"framework"`     // "pytorch", "deepspeed", etc.
	Priority     int    `json:"priority"`      // 0–100

	// Tenant is extracted from the auth context, not the request body.

	Namespace string `json:"namespace"`

	// Resources
	VRAMBytes   int64  `json:"vram_bytes"`
	MinSMs      int    `json:"min_sms,omitempty"`
	Precision   string `json:"precision,omitempty"` // "fp32", "fp16", "bf16", "int8", "mixed"
	MinGPUCount int    `json:"min_gpu_count,omitempty"`

	// Model spec (optional — improves prediction accuracy)
	ModelName      string `json:"model_name,omitempty"`
	ModelSizeBytes int64  `json:"model_size_bytes,omitempty"`
	BatchSize      int    `json:"batch_size,omitempty"`
	SequenceLength int    `json:"sequence_length,omitempty"`

	// Container spec
	Image      string            `json:"image"`
	Command    []string          `json:"command"`
	Args       []string          `json:"args,omitempty"`
	EnvVars    map[string]string `json:"env_vars,omitempty"`
	WorkingDir string            `json:"working_dir,omitempty"`
}

// JobResponse is the JSON representation of a job in API responses.
type JobResponse struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Status       string  `json:"status"`
	Priority     int     `json:"priority"`
	Tenant       string  `json:"tenant"`
	Namespace    string  `json:"namespace"`
	WorkloadType string  `json:"workload_type"`
	Framework    string  `json:"framework"`
	SubmittedAt  string  `json:"submitted_at"`
	ScheduledAt  *string `json:"scheduled_at,omitempty"`
	StartedAt    *string `json:"started_at,omitempty"`
	CompletedAt  *string `json:"completed_at,omitempty"`
}

// UpdatePriorityRequest is the JSON body for PUT /api/v1/jobs/:id/priority.
type UpdatePriorityRequest struct {
	Priority int `json:"priority"` // 0–100
}

// ListJobsResponse is the JSON response for GET /api/v1/jobs.
type ListJobsResponse struct {
	Jobs       []JobResponse `json:"jobs"`
	TotalCount int           `json:"total_count"`
	NextCursor string        `json:"next_cursor,omitempty"`
}

// ---------------------------------------------------------------------------
// JobHandler — HTTP handlers for job operations
// ---------------------------------------------------------------------------

type JobHandler struct {
	scheduler scheduler.Scheduler
}

// NewJobHandler creates a new JobHandler.
func NewJobHandler(sched scheduler.Scheduler) *JobHandler {
	return &JobHandler{scheduler: sched}
}

// Create handles POST /api/v1/jobs — submit a new GPU workload.
//
// The handler:
//   1. Parses and validates the request body
//   2. Extracts tenant from auth context
//   3. Converts to domain.Job
//   4. Calls scheduler.Submit()
//   5. Returns 201 Created with the job ID
func (h *JobHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	// Validate required fields
	if req.Name == "" || req.Image == "" || req.WorkloadType == "" {
		writeError(w, http.StatusBadRequest, "MISSING_FIELDS", "name, image, and workload_type are required")
		return
	}

	tenant := middleware.TenantFromContext(r.Context())

	job := &domain.Job{
		Name:         req.Name,
		Tenant:       tenant,
		Namespace:    req.Namespace,
		WorkloadType: domain.WorkloadType(req.WorkloadType),
		Framework:    req.Framework,
		Priority:     domain.JobPriority(req.Priority),
		Status:       domain.JobStatusPending,
		SubmittedAt:  time.Now(),
		Resources: domain.ResourceRequirements{
			VRAMBytes:   req.VRAMBytes,
			MinSMs:      req.MinSMs,
			Precision:   domain.ComputePrecision(req.Precision),
			MinGPUCount: req.MinGPUCount,
		},
		Container: domain.ContainerSpec{
			Image:      req.Image,
			Command:    req.Command,
			Args:       req.Args,
			EnvVars:    req.EnvVars,
			WorkingDir: req.WorkingDir,
		},
	}

	// Set model spec if provided
	if req.ModelName != "" || req.ModelSizeBytes > 0 {
		job.Model = &domain.ModelSpec{
			ModelName:      req.ModelName,
			ModelSizeBytes: req.ModelSizeBytes,
			BatchSize:      req.BatchSize,
			SequenceLength: req.SequenceLength,
		}
	}

	if err := h.scheduler.Submit(r.Context(), job); err != nil {
		writeError(w, http.StatusInternalServerError, "SUBMIT_FAILED", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": job.ID})
}

// Get handles GET /api/v1/jobs/:id — retrieve a single job.
func (h *JobHandler) Get(w http.ResponseWriter, r *http.Request) {
	_ = r.PathValue("id")
	// TODO: implementation
	// 1. Get job from repository by ID
	// 2. Check tenant authorization
	// 3. Convert to JobResponse
	// 4. Return 200 with job, or 404 if not found
	writeError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "not implemented")
}

// List handles GET /api/v1/jobs — list jobs with filtering.
//
// Query parameters:
//   - status: filter by job status (e.g., ?status=running)
//   - tenant: filter by tenant (admin only; regular users see their own)
//   - cursor: pagination cursor from a previous response
//   - limit: max results per page (default 50, max 200)
func (h *JobHandler) List(w http.ResponseWriter, r *http.Request) {
	// TODO: implementation
	// 1. Parse query parameters
	// 2. Validate limit bounds
	// 3. Query repository with filters
	// 4. Convert to ListJobsResponse
	// 5. Return 200
	writeError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "not implemented")
}

// Cancel handles DELETE /api/v1/jobs/:id — cancel a job.
func (h *JobHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if err := h.scheduler.Cancel(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "CANCEL_FAILED", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// UpdatePriority handles PUT /api/v1/jobs/:id/priority — change job priority.
func (h *JobHandler) UpdatePriority(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req UpdatePriorityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	if req.Priority < 0 || req.Priority > 100 {
		writeError(w, http.StatusBadRequest, "INVALID_PRIORITY", "priority must be between 0 and 100")
		return
	}

	if err := h.scheduler.UpdatePriority(r.Context(), id, domain.JobPriority(req.Priority)); err != nil {
		writeError(w, http.StatusInternalServerError, "UPDATE_FAILED", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Helper: consistent error response format
// ---------------------------------------------------------------------------

func writeError(w http.ResponseWriter, status int, code string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
		"code":  code,
	})
}
