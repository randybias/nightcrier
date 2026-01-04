-- Migration: 000005_add_cluster_tables.up.sql
-- Purpose: Add monitored_clusters and execution_clusters tables for database-backed cluster management

-- Monitored clusters: where fault events are detected
CREATE TABLE IF NOT EXISTS monitored_clusters (
    name TEXT PRIMARY KEY,
    environment TEXT,
    labels TEXT,  -- JSON object stored as TEXT for SQLite compatibility
    mcp_endpoint TEXT NOT NULL,
    mcp_api_key TEXT,
    triage_enabled INTEGER NOT NULL DEFAULT 0,  -- boolean as integer for SQLite
    target_kubeconfig TEXT,  -- Full kubeconfig YAML content
    allow_secrets_access INTEGER NOT NULL DEFAULT 0,  -- boolean as integer for SQLite
    execution_cluster TEXT,  -- FK to execution_clusters.name (not enforced for flexibility)
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    source TEXT NOT NULL DEFAULT 'database'  -- 'yaml' or 'database'
);

-- Execution clusters: where agent Jobs run
CREATE TABLE IF NOT EXISTS execution_clusters (
    name TEXT PRIMARY KEY,
    kubeconfig TEXT NOT NULL,  -- Full kubeconfig YAML content
    namespace TEXT NOT NULL DEFAULT 'nightcrier',
    runner_image TEXT NOT NULL DEFAULT 'nc-agent-runner:latest',
    image_pull_policy TEXT NOT NULL DEFAULT 'IfNotPresent',
    timeout INTEGER NOT NULL DEFAULT 600,
    memory_limit TEXT NOT NULL DEFAULT '2Gi',
    cpu_limit TEXT NOT NULL DEFAULT '1',
    cleanup_ttl INTEGER NOT NULL DEFAULT 3600,
    max_concurrent_agents INTEGER NOT NULL DEFAULT 10,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    source TEXT NOT NULL DEFAULT 'database'  -- 'yaml' or 'database'
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_monitored_clusters_environment ON monitored_clusters(environment);
CREATE INDEX IF NOT EXISTS idx_monitored_clusters_source ON monitored_clusters(source);
CREATE INDEX IF NOT EXISTS idx_execution_clusters_source ON execution_clusters(source);
