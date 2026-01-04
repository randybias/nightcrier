// Package storage provides cluster storage interfaces and record types.
package storage

import (
	"context"
	"time"
)

// ClusterStorage defines the interface for cluster persistence operations.
type ClusterStorage interface {
	// Monitored cluster operations
	ListMonitoredClusters(ctx context.Context) ([]MonitoredClusterRecord, error)
	GetMonitoredCluster(ctx context.Context, name string) (*MonitoredClusterRecord, error)
	UpsertMonitoredCluster(ctx context.Context, cluster *MonitoredClusterRecord) error
	DeleteMonitoredCluster(ctx context.Context, name string) error

	// Execution cluster operations
	ListExecutionClusters(ctx context.Context) ([]ExecutionClusterRecord, error)
	GetExecutionCluster(ctx context.Context, name string) (*ExecutionClusterRecord, error)
	UpsertExecutionCluster(ctx context.Context, cluster *ExecutionClusterRecord) error
	DeleteExecutionCluster(ctx context.Context, name string) error

	// Sync operations for YAML-to-database synchronization
	SyncMonitoredClustersFromYAML(ctx context.Context, clusters []MonitoredClusterRecord) error
	SyncExecutionClustersFromYAML(ctx context.Context, clusters []ExecutionClusterRecord) error
}

// MonitoredClusterRecord represents a monitored cluster stored in the database.
type MonitoredClusterRecord struct {
	// Name is the unique identifier for this cluster
	Name string

	// Environment is an optional environment label (e.g., "production", "staging")
	Environment string

	// Labels are arbitrary key-value pairs for filtering/grouping
	Labels map[string]string

	// MCPEndpoint is the MCP server URL for this cluster
	MCPEndpoint string

	// MCPAPIKey is the optional API key for MCP authentication
	MCPAPIKey string

	// TriageEnabled indicates whether triage is enabled for this cluster
	TriageEnabled bool

	// TargetKubeconfig is the full kubeconfig YAML for triage agent access
	TargetKubeconfig string

	// AllowSecretsAccess indicates whether agents can read secrets
	AllowSecretsAccess bool

	// ExecutionCluster is the name of the execution cluster to use for Jobs
	// If empty, uses the first configured execution cluster
	ExecutionCluster string

	// CreatedAt is when this record was created
	CreatedAt time.Time

	// UpdatedAt is when this record was last updated
	UpdatedAt time.Time

	// Source indicates where this cluster config came from: "yaml" or "database"
	Source string
}

// ExecutionClusterRecord represents an execution cluster stored in the database.
type ExecutionClusterRecord struct {
	// Name is the unique identifier for this execution cluster
	Name string

	// Kubeconfig is the full kubeconfig YAML for accessing this cluster
	Kubeconfig string

	// Namespace where Jobs and ConfigMaps are created
	Namespace string

	// RunnerImage is the container image for the agent runner
	RunnerImage string

	// ImagePullPolicy: Always, Never, IfNotPresent
	ImagePullPolicy string

	// Timeout is the Job timeout in seconds
	Timeout int

	// MemoryLimit for Job containers (e.g., "2Gi")
	MemoryLimit string

	// CPULimit for Job containers (e.g., "1")
	CPULimit string

	// CleanupTTL is the TTL for Job cleanup after completion (seconds)
	CleanupTTL int

	// MaxConcurrentAgents is the maximum number of concurrent agents
	MaxConcurrentAgents int

	// CreatedAt is when this record was created
	CreatedAt time.Time

	// UpdatedAt is when this record was last updated
	UpdatedAt time.Time

	// Source indicates where this config came from: "yaml" or "database"
	Source string
}
