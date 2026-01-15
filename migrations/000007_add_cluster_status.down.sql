-- Migration: 000007_add_cluster_status.down.sql
-- Purpose: Remove reachability columns from monitored_clusters

-- SQLite doesn't support DROP COLUMN directly, so we need to recreate the table
-- This is handled by the migration tool's SQLite-specific logic

-- For PostgreSQL:
ALTER TABLE monitored_clusters DROP COLUMN IF EXISTS connection_status;
ALTER TABLE monitored_clusters DROP COLUMN IF EXISTS unreachable;
ALTER TABLE monitored_clusters DROP COLUMN IF EXISTS unreachable_reason;
ALTER TABLE monitored_clusters DROP COLUMN IF EXISTS last_status_check;
ALTER TABLE monitored_clusters DROP COLUMN IF EXISTS last_error;
