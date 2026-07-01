CREATE TABLE IF NOT EXISTS repos (
    id TEXT PRIMARY KEY,
    path TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS memories (
    id TEXT PRIMARY KEY,
    repo_id TEXT NOT NULL,
    kind TEXT NOT NULL,
    content TEXT NOT NULL,
    source TEXT NOT NULL DEFAULT 'mcp',
    metadata_json TEXT,
    session_id TEXT,
    summary TEXT,
    deleted INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    access_count INTEGER DEFAULT 0,
    last_accessed TEXT,
    relevance_score REAL DEFAULT 0.0,
    quality_tier TEXT DEFAULT 'unrated',
    FOREIGN KEY (repo_id) REFERENCES repos(id)
);

CREATE INDEX IF NOT EXISTS idx_memories_repo_kind
    ON memories(repo_id, kind, deleted, created_at DESC);

CREATE TABLE IF NOT EXISTS memory_embeddings (
    memory_id TEXT PRIMARY KEY,
    embedding_json TEXT NOT NULL,
    FOREIGN KEY (memory_id) REFERENCES memories(id)
);

CREATE TABLE IF NOT EXISTS failure_signatures (
    repo_id TEXT NOT NULL,
    signature TEXT NOT NULL,
    count INTEGER NOT NULL DEFAULT 1,
    first_seen TEXT NOT NULL,
    last_seen TEXT NOT NULL,
    resolved INTEGER NOT NULL DEFAULT 0,
    memory_id TEXT,
    PRIMARY KEY (repo_id, signature)
);

CREATE TABLE IF NOT EXISTS memory_access_log (
    id TEXT PRIMARY KEY,
    memory_id TEXT NOT NULL,
    access_type TEXT NOT NULL,
    query_text TEXT,
    session_id TEXT,
    host_ide TEXT,
    was_useful INTEGER,
    created_at TEXT NOT NULL,
    FOREIGN KEY (memory_id) REFERENCES memories(id)
);

CREATE INDEX IF NOT EXISTS idx_access_log_memory
    ON memory_access_log(memory_id, created_at DESC);

CREATE VIRTUAL TABLE IF NOT EXISTS memories_fts USING fts5(
    content, kind, repo_id
);
