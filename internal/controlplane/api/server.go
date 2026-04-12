// Package api implements the Veltrix REST API server.
//
// This is the control plane's front door — the single entry point for all
// external interactions. The Grafana app plugin, CLI, and SDK all talk to
// this server.
//
// Responsibilities:
//   - HTTP routing and request/response serialization
//   - Authentication and authorization (via middleware)
//   - Input validation (reject malformed requests before they reach services)
//   - Rate limiting
//   - Request tracing (OpenTelemetry)
//
// The API server is deliberately thin. It does NOT contain business logic.
// Every handler delegates to a control plane service (scheduler, policy engine,
// etc.) and translates the result to an HTTP response.
//
// API design:
//   - RESTful JSON over HTTP/2
//   - Versioned: /api/v1/...
//   - Consistent error format: {"error": "message", "code": "ERROR_CODE"}
//   - Pagination: cursor-based for lists
//   - Filtering: query parameters for common filters
package api

import (
	"context"
	"net/http"

	"veltrix/internal/controlplane/api/handlers"
	"veltrix/internal/controlplane/api/middleware"
	"veltrix/internal/controlplane/feedback"
	"veltrix/internal/controlplane/scheduler"
)

// ---------------------------------------------------------------------------
// Server — the HTTP server
// ---------------------------------------------------------------------------

// Server is the Veltrix API server. It wraps an http.Server and registers
// all routes, middleware, and handlers.
type Server struct {
	httpServer *http.Server

	// --- Handlers ---
	jobHandler     *handlers.JobHandler
	nodeHandler    *handlers.NodeHandler
	metricsHandler *handlers.MetricsHandler

	// --- Middleware ---
	auth *middleware.AuthMiddleware
}

// Config holds the API server configuration.
type Config struct {
	// Port is the TCP port to listen on (e.g., "8080").
	Port string

	// AllowedOrigins is the list of CORS origins for the Grafana plugin.
	// Example: ["http://localhost:3000", "https://grafana.internal"]
	AllowedOrigins []string
}

// NewServer creates a new API server with all dependencies wired.
//
// The server does not start listening until Start() is called.
// All dependencies (scheduler, feedback controller, repositories) are
// injected here — the server never creates its own.
func NewServer(
	cfg Config,
	sched scheduler.Scheduler,
	fb feedback.FeedbackController,
	auth *middleware.AuthMiddleware,
) *Server {
	jobHandler := handlers.NewJobHandler(sched)
	nodeHandler := handlers.NewNodeHandler()
	metricsHandler := handlers.NewMetricsHandler(fb)

	s := &Server{
		jobHandler:     jobHandler,
		nodeHandler:    nodeHandler,
		metricsHandler: metricsHandler,
		auth:           auth,
	}

	mux := s.registerRoutes()

	s.httpServer = &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: mux,
	}

	return s
}

// registerRoutes sets up the HTTP route table.
//
// Route structure:
//
//	/api/v1/jobs          POST    — submit a new job
//	/api/v1/jobs          GET     — list jobs (filterable by status, tenant)
//	/api/v1/jobs/:id      GET     — get a single job
//	/api/v1/jobs/:id      DELETE  — cancel a job
//	/api/v1/jobs/:id/priority PUT — update job priority
//
//	/api/v1/nodes         GET     — list all nodes
//	/api/v1/nodes/:id     GET     — get a single node with GPU details
//	/api/v1/nodes/:id/gpus GET    — list GPUs on a node
//
//	/api/v1/metrics/accuracy GET  — prediction accuracy report
//	/api/v1/metrics/cluster  GET  — cluster-wide utilization summary
//
//	/api/v1/policies      POST    — create a policy
//	/api/v1/policies      GET     — list policies
//	/api/v1/policies/:id  PUT     — update a policy
//	/api/v1/policies/:id  DELETE  — delete a policy
//
//	/healthz              GET     — health check (no auth)
//	/readyz               GET     — readiness check (no auth)
func (s *Server) registerRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	// Health checks — no auth required
	mux.HandleFunc("GET /healthz", s.healthz)
	mux.HandleFunc("GET /readyz", s.readyz)

	// Job routes
	mux.HandleFunc("POST /api/v1/jobs", s.auth.Wrap(s.jobHandler.Create))
	mux.HandleFunc("GET /api/v1/jobs", s.auth.Wrap(s.jobHandler.List))
	mux.HandleFunc("GET /api/v1/jobs/{id}", s.auth.Wrap(s.jobHandler.Get))
	mux.HandleFunc("DELETE /api/v1/jobs/{id}", s.auth.Wrap(s.jobHandler.Cancel))
	mux.HandleFunc("PUT /api/v1/jobs/{id}/priority", s.auth.Wrap(s.jobHandler.UpdatePriority))

	// Node routes
	mux.HandleFunc("GET /api/v1/nodes", s.auth.Wrap(s.nodeHandler.List))
	mux.HandleFunc("GET /api/v1/nodes/{id}", s.auth.Wrap(s.nodeHandler.Get))
	mux.HandleFunc("GET /api/v1/nodes/{id}/gpus", s.auth.Wrap(s.nodeHandler.ListGPUs))

	// Metrics routes
	mux.HandleFunc("GET /api/v1/metrics/accuracy", s.auth.Wrap(s.metricsHandler.GetAccuracy))
	mux.HandleFunc("GET /api/v1/metrics/cluster", s.auth.Wrap(s.metricsHandler.GetClusterSummary))

	return mux
}

// Start begins listening for HTTP requests.
// Blocks until the server is shut down. Should be called in a goroutine.
func (s *Server) Start() error {
	return s.httpServer.ListenAndServe()
}

// Stop gracefully shuts down the HTTP server.
// Waits for in-flight requests to complete.
func (s *Server) Stop(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// healthz is a liveness probe. Returns 200 if the process is alive.
func (s *Server) healthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

// readyz is a readiness probe. Returns 200 if the server can accept traffic.
// Checks downstream dependencies (DB, cache).
func (s *Server) readyz(w http.ResponseWriter, r *http.Request) {
	// TODO: check Postgres connectivity
	// TODO: check Redis connectivity
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ready"}`))
}
