// Package cache — Redis implementation.
//
// RedisCache implements all cache interfaces (GPUStateCache, NodeStateCache,
// LockManager) using a single Redis connection.
//
// Key schema:
//
//   GPU state:    veltrix:gpu:{gpuID}:state     → JSON(GPUTelemetry)   TTL: 30s
//   Node state:   veltrix:node:{nodeID}:state   → JSON(NodeState)     TTL: 30s
//   GPU lock:     veltrix:lock:gpu:{gpuID}      → lock holder ID      TTL: 10s
//   MIG lock:     veltrix:lock:mig:{gpuID}      → lock holder ID      TTL: 5m
//
// All keys are prefixed with "veltrix:" to avoid collisions if Redis is
// shared with other services.
//
// Serialization: JSON (simple, debuggable). If performance becomes an issue,
// switch to msgpack or protobuf.
package cache

import (
	"context"
	"time"

	"veltrix/internal/controlplane/domain"
)

// ---------------------------------------------------------------------------
// Configuration
// ---------------------------------------------------------------------------

// RedisConfig holds Redis connection settings.
type RedisConfig struct {
	// Addr is the Redis server address (e.g., "localhost:6379").
	Addr string

	// Password is the Redis password. Empty for no auth.
	Password string

	// DB is the Redis database number (default: 0).
	DB int

	// GPUStateTTL is the expiration time for GPU state entries.
	// Default: 30 seconds.
	GPUStateTTL time.Duration

	// NodeStateTTL is the expiration time for node state entries.
	// Default: 30 seconds.
	NodeStateTTL time.Duration
}

// ---------------------------------------------------------------------------
// RedisCache — implements GPUStateCache, NodeStateCache, LockManager
// ---------------------------------------------------------------------------

type RedisCache struct {
	config RedisConfig
	// TODO: add redis client (e.g., github.com/redis/go-redis/v9)
}

// NewRedisCache creates a new Redis-backed cache.
func NewRedisCache(config RedisConfig) *RedisCache {
	if config.GPUStateTTL == 0 {
		config.GPUStateTTL = 30 * time.Second
	}
	if config.NodeStateTTL == 0 {
		config.NodeStateTTL = 30 * time.Second
	}

	return &RedisCache{
		config: config,
	}
}

// Close releases the Redis connection.
func (c *RedisCache) Close() error {
	// TODO: close redis client
	return nil
}

// --- GPUStateCache implementation ---

func (c *RedisCache) SetGPUState(ctx context.Context, gpuID string, telemetry *domain.GPUTelemetry) error {
	// TODO: implementation
	// key := fmt.Sprintf("veltrix:gpu:%s:state", gpuID)
	// data, _ := json.Marshal(telemetry)
	// return c.client.Set(ctx, key, data, c.config.GPUStateTTL).Err()
	return nil
}

func (c *RedisCache) GetGPUState(ctx context.Context, gpuID string) (*domain.GPUTelemetry, error) {
	// TODO: implementation
	return nil, nil
}

func (c *RedisCache) GetAllGPUStates(ctx context.Context) (map[string]*domain.GPUTelemetry, error) {
	// TODO: implementation
	// Use SCAN with pattern "veltrix:gpu:*:state" to find all GPU keys
	// Pipeline GET for each key
	return nil, nil
}

// --- NodeStateCache implementation ---

func (c *RedisCache) SetNodeState(ctx context.Context, nodeID string, state *NodeState) error {
	// TODO: implementation
	return nil
}

func (c *RedisCache) GetNodeState(ctx context.Context, nodeID string) (*NodeState, error) {
	// TODO: implementation
	return nil, nil
}

// --- LockManager implementation ---

func (c *RedisCache) AcquireGPULock(ctx context.Context, gpuID string, ttl time.Duration) (bool, error) {
	// TODO: implementation
	// key := fmt.Sprintf("veltrix:lock:gpu:%s", gpuID)
	// Use SET NX with TTL:
	// ok, err := c.client.SetNX(ctx, key, instanceID, ttl).Result()
	// return ok, err
	return false, nil
}

func (c *RedisCache) ReleaseGPULock(ctx context.Context, gpuID string) error {
	// TODO: implementation
	// Use Lua script to check-and-delete (only release if we own the lock)
	return nil
}

func (c *RedisCache) AcquireMIGLock(ctx context.Context, gpuID string, ttl time.Duration) (bool, error) {
	// TODO: implementation
	return false, nil
}

func (c *RedisCache) ReleaseMIGLock(ctx context.Context, gpuID string) error {
	// TODO: implementation
	return nil
}
