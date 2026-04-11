CREATE TABLE IF NOT EXISTS recommendations (
    id SERIAL PRIMARY KEY,
    gpu_id TEXT,
    action TEXT,
    reason TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);