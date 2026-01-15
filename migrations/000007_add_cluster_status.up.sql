-- Migration: 000007_add_cluster_status.up.sql
-- Purpose: Add reachability columns to monitored_clusters for connection tracking

-- Connection status: tracks MCP connection state
-- Values: disconnected, connecting, connected, subscribing, active, failed
ALTER TABLE monitored_clusters ADD COLUMN connection_status TEXT NOT NULL DEFAULT 'disconnected';

-- Unreachable flag: indicates Nightcrier cannot reach the cluster
-- (MCP connection failed or permission validation failed)
ALTER TABLE monitored_clusters ADD COLUMN unreachable INTEGER NOT NULL DEFAULT 0;

-- Unreachable reason: explains why the cluster is unreachable
ALTER TABLE monitored_clusters ADD COLUMN unreachable_reason TEXT;

-- Last status check timestamp
ALTER TABLE monitored_clusters ADD COLUMN last_status_check TIMESTAMP;

-- Last error message (for display in admin UI)
ALTER TABLE monitored_clusters ADD COLUMN last_error TEXT;
