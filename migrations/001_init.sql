-- Migration 001: Initial schema for Veltrix state store.
--
-- This creates all tables needed by the control plane. Tables map 1:1
-- to domain entities. The schema is normalized — joins are acceptable
-- because the hot path (placement decisions) reads from Redis, not Postgres.
--
-- Index strategy: index columns used in WHERE clauses by the repository
-- interfaces. Avoid over-indexing — metrics table is write-heavy.

-- ---------------------------------------------------------------------------
-- Jobs
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS jobs (
    id             TEXT PRIMARY KEY,
    name           TEXT NOT NULL,
    tenant         TEXT NOT NULL,
    namespace      TEXT NOT NULL DEFAULT 'default',

    -- Workload
    workload_type  TEXT NOT NULL,  -- training, inference, fine_tuning, evaluation, preprocessing
    framework      TEXT NOT NULL DEFAULT '',

    -- Resources
    vram_bytes     BIGINT NOT NULL DEFAULT 0,
    min_sms        INT NOT NULL DEFAULT 0,
    precision      TEXT NOT NULL DEFAULT 'fp32',
    min_gpu_count  INT NOT NULL DEFAULT 1,
    max_power_watts INT NOT NULL DEFAULT 0,

    -- Model spec (nullable — not all jobs have model metadata)
    model_name       TEXT,
    model_size_bytes BIGINT,
    batch_size       INT,
    sequence_length  INT,

    -- Container spec
    image          TEXT NOT NULL,
    command        JSONB NOT NULL DEFAULT '[]',
    args           JSONB NOT NULL DEFAULT '[]',
    env_vars       JSONB NOT NULL DEFAULT '{}',
    working_dir    TEXT NOT NULL DEFAULT '',

    -- Lifecycle
    status         TEXT NOT NULL DEFAULT 'pending',
    priority       INT NOT NULL DEFAULT 50,
    failure_reason TEXT,

    -- Timestamps
    submitted_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    scheduled_at   TIMESTAMPTZ,
    started_at     TIMESTAMPTZ,
    completed_at   TIMESTAMPTZ
);

-- Most common queries: list by status, filter by tenant
CREATE INDEX idx_jobs_status ON jobs(status);
CREATE INDEX idx_jobs_tenant ON jobs(tenant);
CREATE INDEX idx_jobs_tenant_status ON jobs(tenant, status);

-- ---------------------------------------------------------------------------
-- Nodes
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS nodes (
    id               TEXT PRIMARY KEY,
    hostname         TEXT NOT NULL,
    kubernetes_name  TEXT NOT NULL DEFAULT '',
    status           TEXT NOT NULL DEFAULT 'ready',

    -- Host resources
    cpu_cores           INT NOT NULL DEFAULT 0,
    cpu_available_cores INT NOT NULL DEFAULT 0,
    memory_bytes        BIGINT NOT NULL DEFAULT 0,
    memory_available    BIGINT NOT NULL DEFAULT 0,
    storage_bytes       BIGINT NOT NULL DEFAULT 0,
    storage_available   BIGINT NOT NULL DEFAULT 0,

    -- Topology (stored as JSON for flexibility)
    nvlink_pairs     JSONB NOT NULL DEFAULT '[]',
    numa_zones       JSONB NOT NULL DEFAULT '{}',

    -- Agent
    agent_version    TEXT NOT NULL DEFAULT '',
    last_heartbeat   TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Labels
    labels           JSONB NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_nodes_status ON nodes(status);

-- ---------------------------------------------------------------------------
-- GPUs
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS gpus (
    id            TEXT PRIMARY KEY,
    node_id       TEXT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    device_index  INT NOT NULL,

    -- Hardware
    model         TEXT NOT NULL,  -- a100_40gb, a100_80gb, h100_80gb
    vram_bytes    BIGINT NOT NULL,
    total_sms     INT NOT NULL DEFAULT 0,
    mig_capable   BOOLEAN NOT NULL DEFAULT FALSE,

    -- State
    status        TEXT NOT NULL DEFAULT 'available',
    strategy      TEXT NOT NULL DEFAULT 'idle',

    -- MIG instances (stored as JSON array — changes infrequently)
    mig_instances JSONB NOT NULL DEFAULT '[]',

    -- Active jobs (array of job IDs)
    active_job_ids JSONB NOT NULL DEFAULT '[]',

    UNIQUE(node_id, device_index)
);

CREATE INDEX idx_gpus_node_id ON gpus(node_id);
CREATE INDEX idx_gpus_status ON gpus(status);

-- ---------------------------------------------------------------------------
-- Placements
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS placements (
    id              TEXT PRIMARY KEY,
    job_id          TEXT NOT NULL REFERENCES jobs(id),
    node_id         TEXT NOT NULL REFERENCES nodes(id),

    -- GPU assignments (JSON array of {gpu_id, device_index, mig_instance_id})
    gpu_assignments JSONB NOT NULL DEFAULT '[]',
    strategy        TEXT NOT NULL,  -- full_gpu, mig, mps

    -- Prediction at decision time
    predicted_vram_bytes     BIGINT NOT NULL DEFAULT 0,
    predicted_utilization    FLOAT NOT NULL DEFAULT 0,
    predicted_runtime_secs   BIGINT NOT NULL DEFAULT 0,
    prediction_confidence    FLOAT NOT NULL DEFAULT 0,

    -- Scoring metadata
    node_score      FLOAT NOT NULL DEFAULT 0,
    reason          TEXT NOT NULL DEFAULT '',

    -- Lifecycle
    status          TEXT NOT NULL DEFAULT 'decided',
    decided_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    applied_at      TIMESTAMPTZ,
    released_at     TIMESTAMPTZ
);

CREATE INDEX idx_placements_job_id ON placements(job_id);
CREATE INDEX idx_placements_node_id ON placements(node_id);
CREATE INDEX idx_placements_status ON placements(status);

-- ---------------------------------------------------------------------------
-- Metrics (GPU telemetry history)
-- ---------------------------------------------------------------------------
-- This is the highest-volume table. Optimize for write throughput.
-- Reads are infrequent (post-job analysis by feedback controller).
-- Consider TimescaleDB hypertable in production for automatic partitioning.

CREATE TABLE IF NOT EXISTS metrics (
    gpu_id              TEXT NOT NULL,
    utilization         FLOAT NOT NULL,
    memory_used_bytes   BIGINT NOT NULL,
    power_draw_watts    FLOAT NOT NULL DEFAULT 0,
    temperature_celsius INT NOT NULL DEFAULT 0,
    pcie_throughput_mbps FLOAT NOT NULL DEFAULT 0,
    sm_occupancy        FLOAT NOT NULL DEFAULT 0,
    recorded_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Time-range queries for post-job analysis
CREATE INDEX idx_metrics_gpu_time ON metrics(gpu_id, recorded_at DESC);

-- ---------------------------------------------------------------------------
-- Policies
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS policies (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL UNIQUE,
    type       TEXT NOT NULL,  -- isolation, co_location, sla, quota, affinity
    priority   INT NOT NULL DEFAULT 0,
    enabled    BOOLEAN NOT NULL DEFAULT TRUE,

    -- Scope
    tenant     TEXT NOT NULL DEFAULT '',
    namespace  TEXT NOT NULL DEFAULT '',

    -- Rules (exactly one of these is set, based on type)
    -- Stored as JSON for flexibility — policy structure may evolve.
    rules      JSONB NOT NULL DEFAULT '{}',

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_policies_type ON policies(type);
CREATE INDEX idx_policies_enabled ON policies(enabled) WHERE enabled = TRUE;

-- ---------------------------------------------------------------------------
-- Predictions (feedback: predicted vs actual)
-- ---------------------------------------------------------------------------
-- Written by the feedback controller after each job completes.
-- Read for accuracy reports and prediction engine retraining.

CREATE TABLE IF NOT EXISTS predictions (
    id                    TEXT PRIMARY KEY,
    job_id                TEXT NOT NULL REFERENCES jobs(id),
    placement_id          TEXT NOT NULL REFERENCES placements(id),

    -- Predicted values (from placement decision time)
    predicted_vram_bytes  BIGINT NOT NULL,
    predicted_utilization FLOAT NOT NULL,
    predicted_runtime_secs BIGINT NOT NULL,

    -- Actual values (measured over job lifetime)
    actual_peak_vram      BIGINT NOT NULL,
    actual_avg_utilization FLOAT NOT NULL,
    actual_runtime_secs   BIGINT NOT NULL,

    -- Derived accuracy (0.0–1.0)
    vram_accuracy         FLOAT NOT NULL,
    utilization_accuracy  FLOAT NOT NULL,
    runtime_accuracy      FLOAT NOT NULL,

    -- Metadata
    workload_type         TEXT NOT NULL,
    framework             TEXT NOT NULL DEFAULT '',

    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_predictions_workload ON predictions(workload_type);
CREATE INDEX idx_predictions_created ON predictions(created_at DESC);
