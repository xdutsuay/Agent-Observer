CREATE TABLE IF NOT EXISTS usage_sessions (
    id TEXT PRIMARY KEY,
    client_name TEXT,
    client_version TEXT,
    host_ide TEXT,
    transport TEXT,
    connected_at TEXT,
    last_seen_at TEXT
);

CREATE TABLE IF NOT EXISTS usage_interactions (
    id TEXT PRIMARY KEY,
    session_id TEXT,
    transport TEXT NOT NULL,
    method TEXT NOT NULL,
    client_name TEXT,
    client_version TEXT,
    host_ide TEXT,
    query_summary TEXT,
    query_json TEXT,
    response_preview TEXT,
    duration_ms REAL,
    ok INTEGER DEFAULT 1,
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_usage_interactions_created
    ON usage_interactions(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_usage_interactions_host
    ON usage_interactions(host_ide);
