// Package middleware provides HTTP middleware for the Veltrix API server.
//
// Middleware runs before handlers and handles cross-cutting concerns:
//   - Authentication: verify the caller's identity
//   - Authorization: verify the caller has permission for the action
//   - Request ID: generate and propagate a unique request ID
//   - Logging: structured request/response logging
//   - Rate limiting: prevent abuse
//
// All middleware follows the standard Go pattern of wrapping http.HandlerFunc.
package middleware

import (
	"context"
	"net/http"
)

// ---------------------------------------------------------------------------
// Context keys for passing auth info through the request context
// ---------------------------------------------------------------------------

type contextKey string

const (
	// ContextKeyTenant is the authenticated tenant ID.
	ContextKeyTenant contextKey = "tenant"

	// ContextKeyUserID is the authenticated user ID.
	ContextKeyUserID contextKey = "user_id"

	// ContextKeyRequestID is the unique request ID for tracing.
	ContextKeyRequestID contextKey = "request_id"
)

// TenantFromContext extracts the tenant ID from the request context.
// Returns empty string if not authenticated.
func TenantFromContext(ctx context.Context) string {
	v, _ := ctx.Value(ContextKeyTenant).(string)
	return v
}

// UserIDFromContext extracts the user ID from the request context.
func UserIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(ContextKeyUserID).(string)
	return v
}

// ---------------------------------------------------------------------------
// AuthMiddleware — verifies identity and permissions
// ---------------------------------------------------------------------------
//
// Authentication strategy (pluggable):
//   - API key: simple, good for service-to-service calls
//   - JWT/OIDC: for user-facing requests from Grafana
//   - mTLS: for agent-to-control-plane communication
//
// The Grafana app plugin sends requests with the user's session token.
// The API server validates this against the configured auth provider.
// ---------------------------------------------------------------------------

// AuthConfig configures the authentication middleware.
type AuthConfig struct {
	// Enabled controls whether authentication is enforced.
	// Set to false for local development.
	Enabled bool

	// APIKeyHeader is the HTTP header for API key authentication.
	// Default: "X-API-Key"
	APIKeyHeader string

	// JWTIssuer is the expected JWT issuer URL (for OIDC validation).
	JWTIssuer string

	// JWTAudience is the expected JWT audience.
	JWTAudience string
}

// AuthMiddleware handles authentication and authorization.
type AuthMiddleware struct {
	config AuthConfig
}

// NewAuthMiddleware creates a new AuthMiddleware with the given configuration.
func NewAuthMiddleware(config AuthConfig) *AuthMiddleware {
	return &AuthMiddleware{config: config}
}

// Wrap returns a new http.HandlerFunc that performs authentication
// before calling the inner handler.
//
// If authentication fails, returns 401 Unauthorized.
// If authorization fails, returns 403 Forbidden.
// If authentication is disabled (dev mode), all requests are allowed
// with tenant="default" and user_id="anonymous".
func (m *AuthMiddleware) Wrap(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !m.config.Enabled {
			// Dev mode: inject default identity
			ctx := context.WithValue(r.Context(), ContextKeyTenant, "default")
			ctx = context.WithValue(ctx, ContextKeyUserID, "anonymous")
			next(w, r.WithContext(ctx))
			return
		}

		// TODO: implementation
		// 1. Check for API key header → validate against stored keys
		// 2. Check for Authorization: Bearer <jwt> → validate JWT
		// 3. Extract tenant and user_id from validated token
		// 4. Inject into context
		// 5. Call next handler

		http.Error(w, `{"error":"unauthorized","code":"AUTH_REQUIRED"}`, http.StatusUnauthorized)
	}
}
