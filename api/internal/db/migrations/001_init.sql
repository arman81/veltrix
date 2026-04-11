CREATE TABLE IF NOT EXISTS metrics (
    id SERIAL PRIMARY KEY,
    gpu_id TEXT,
    utilization FLOAT,
    memory FLOAT,
    created_at TIMESTAMP DEFAULT NOW()
);