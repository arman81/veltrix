// Package postgres implements the repository interfaces using PostgreSQL.
//
// This is the production state store for Veltrix. All persistent state
// (jobs, nodes, GPUs, placements, metrics, policies) lives in Postgres.
//
// Design decisions:
//   - Uses database/sql with lib/pq driver (no ORM)
//   - Prepared statements for all queries (connection pooling friendly)
//   - Transactions for multi-table writes (e.g., job status + placement status)
//   - Batch inserts for metrics (high-throughput path)
//   - Migrations managed via numbered SQL files in /migrations
//
// Connection pooling:
//   - database/sql manages a pool automatically
//   - MaxOpenConns, MaxIdleConns, ConnMaxLifetime are configured via Config
//   - For production: 25 max open, 10 idle, 5 min lifetime
//
// Schema is defined in /migrations/*.sql and applied at startup.
package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// ---------------------------------------------------------------------------
// Configuration
// ---------------------------------------------------------------------------

// Config holds PostgreSQL connection and pool settings.
type Config struct {
	// Host is the database server hostname (e.g., "localhost", "postgres").
	Host string

	// Port is the database server port (default: 5432).
	Port int

	// Database is the database name (e.g., "veltrix").
	Database string

	// User is the database user.
	User string

	// Password is the database password.
	Password string

	// SSLMode is the SSL mode (e.g., "disable", "require", "verify-full").
	SSLMode string

	// MaxOpenConns is the maximum number of open connections in the pool.
	MaxOpenConns int

	// MaxIdleConns is the maximum number of idle connections in the pool.
	MaxIdleConns int

	// ConnMaxLifetime is the maximum lifetime of a connection before it is closed.
	ConnMaxLifetime time.Duration
}

// DSN returns the PostgreSQL connection string.
func (c Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		c.Host, c.Port, c.Database, c.User, c.Password, c.SSLMode,
	)
}

// ---------------------------------------------------------------------------
// Connection
// ---------------------------------------------------------------------------

// Connect establishes a connection pool to PostgreSQL and verifies connectivity.
func Connect(cfg Config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure the connection pool
	if cfg.MaxOpenConns > 0 {
		db.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns > 0 {
		db.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}

	// Verify connectivity
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// ---------------------------------------------------------------------------
// Repository implementations — stubs
// ---------------------------------------------------------------------------
//
// Each repository gets its own file once we implement them:
//   postgres/jobs.go      → JobRepositoryPostgres
//   postgres/nodes.go     → NodeRepositoryPostgres
//   postgres/gpus.go      → GPURepositoryPostgres
//   postgres/placements.go → PlacementRepositoryPostgres
//   postgres/metrics.go   → MetricsRepositoryPostgres
//   postgres/policies.go  → PolicyRepositoryPostgres
//
// For now, we define the struct shells here so the package compiles.
// ---------------------------------------------------------------------------

// Store holds all Postgres-backed repository implementations.
// This is the single object that cmd/api/main.go creates and injects
// into control plane services.
type Store struct {
	db *sql.DB

	Jobs       *JobRepositoryPostgres
	Nodes      *NodeRepositoryPostgres
	GPUs       *GPURepositoryPostgres
	Placements *PlacementRepositoryPostgres
	Metrics    *MetricsRepositoryPostgres
	Policies   *PolicyRepositoryPostgres
}

// NewStore creates a Store with all repository implementations.
func NewStore(db *sql.DB) *Store {
	return &Store{
		db:         db,
		Jobs:       &JobRepositoryPostgres{db: db},
		Nodes:      &NodeRepositoryPostgres{db: db},
		GPUs:       &GPURepositoryPostgres{db: db},
		Placements: &PlacementRepositoryPostgres{db: db},
		Metrics:    &MetricsRepositoryPostgres{db: db},
		Policies:   &PolicyRepositoryPostgres{db: db},
	}
}

// Close closes the database connection pool.
func (s *Store) Close() error {
	return s.db.Close()
}

// --- Stub implementations (each will be moved to its own file) ---

type JobRepositoryPostgres struct{ db *sql.DB }
type NodeRepositoryPostgres struct{ db *sql.DB }
type GPURepositoryPostgres struct{ db *sql.DB }
type PlacementRepositoryPostgres struct{ db *sql.DB }
type MetricsRepositoryPostgres struct{ db *sql.DB }
type PolicyRepositoryPostgres struct{ db *sql.DB }
