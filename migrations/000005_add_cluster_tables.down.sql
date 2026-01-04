-- Migration: 000005_add_cluster_tables.down.sql
-- Purpose: Remove monitored_clusters and execution_clusters tables

DROP INDEX IF EXISTS idx_execution_clusters_source;
DROP INDEX IF EXISTS idx_monitored_clusters_source;
DROP INDEX IF EXISTS idx_monitored_clusters_environment;

DROP TABLE IF EXISTS execution_clusters;
DROP TABLE IF EXISTS monitored_clusters;
