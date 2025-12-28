// Package storage provides interfaces and implementations for persisting incident artifacts.
package storage

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"
)

// Storage defines the interface for persisting incident artifacts to local or cloud storage.
type Storage interface {
	// SaveIncident uploads all artifacts for an incident to storage.
	// It returns URLs to access the artifacts and metadata about the storage operation.
	SaveIncident(ctx context.Context, incidentID string, artifacts *IncidentArtifacts) (*SaveResult, error)
}

// AgentLogs contains the captured log output from the agent's execution.
// These logs are read from the workspace logs/ directory after the agent completes.
type AgentLogs struct {
	// Stdout contains the standard output from the agent
	Stdout []byte
	// Stderr contains the standard error output from the agent
	Stderr []byte
	// Combined contains the combined stdout and stderr in chronological order
	Combined []byte
	// CommandsExecuted contains the extracted Bash commands from the agent session (DEBUG mode only)
	CommandsExecuted []byte
}

// IncidentArtifacts contains all files generated during incident investigation.
type IncidentArtifacts struct {
	// IncidentJSON is the serialized incident (combines event context and result)
	IncidentJSON []byte
	// InvestigationMD is the markdown investigation report
	InvestigationMD []byte
	// InvestigationHTML is the HTML-rendered version of the investigation report
	InvestigationHTML []byte
	// ClusterPermissionsJSON contains the validated cluster permissions for the triage agent
	ClusterPermissionsJSON []byte
	// AgentLogs contains the captured log output from the agent's execution (DEBUG mode only)
	AgentLogs AgentLogs
	// ClaudeSessionArchive contains the tar.gz archive of ~/.claude from the agent container (DEBUG mode only)
	ClaudeSessionArchive []byte
	// PromptSent is the captured prompt sent to the agent (system + additional)
	PromptSent []byte
}

// SaveResult contains the results of a storage operation, including URLs to access artifacts.
type SaveResult struct {
	// ReportURL is the authenticated URL to the investigation report (investigation.md)
	ReportURL string
	// ArtifactURLs maps artifact names to their authenticated URLs
	// Common keys: "incident.json", "investigation.md"
	ArtifactURLs map[string]string
	// CanonicalURLs maps artifact names to their canonical URLs (no authentication)
	// These URLs are stable and can be stored long-term, but may require re-signing for access
	CanonicalURLs map[string]string
	// LogURLs maps agent log file names to their presigned URLs
	// Common keys: "stdout.log", "stderr.log", "combined.log"
	LogURLs map[string]string
	// ExpiresAt is when the URLs expire (relevant for cloud storage with SAS tokens)
	ExpiresAt time.Time
}

// StorageConfig represents the configuration needed to initialize storage backends.
// This interface allows us to accept different config types without importing
// the concrete config package (avoiding circular dependencies).
type StorageConfig interface {
	// GetWorkspaceRoot returns the filesystem workspace root directory
	GetWorkspaceRoot() string
	// GetObjectStorageURL returns the Go CDK storage URL (empty if not configured)
	GetObjectStorageURL() string
	// GetObjectStorageExpiry returns the signed URL expiry duration
	GetObjectStorageExpiry() time.Duration
	// GetAzureStorageAccount returns the Azure storage account name (for Azure provider)
	GetAzureStorageAccount() string
	// GetAzureStorageKey returns the Azure storage key (for Azure provider)
	GetAzureStorageKey() string
	// GetAWSAccessKeyID returns the AWS access key ID (for S3 provider)
	GetAWSAccessKeyID() string
	// GetAWSSecretAccessKey returns the AWS secret access key (for S3 provider)
	GetAWSSecretAccessKey() string
}

// setProviderEnvironment sets provider-specific environment variables required by Go CDK.
// Environment variables take precedence over config file values (12-factor app principle).
// Azure: AZURE_STORAGE_ACCOUNT, AZURE_STORAGE_KEY
// S3: AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY
func setProviderEnvironment(storageURL string, cfg StorageConfig) error {
	// Parse URL to detect provider
	if strings.HasPrefix(storageURL, "azblob://") {
		// Azure Blob Storage requires AZURE_STORAGE_ACCOUNT and AZURE_STORAGE_KEY
		// Only set from config if not already in environment
		if os.Getenv("AZURE_STORAGE_ACCOUNT") == "" {
			account := cfg.GetAzureStorageAccount()
			if account == "" {
				return fmt.Errorf("Azure storage account not configured (set AZURE_STORAGE_ACCOUNT env var or config file)")
			}
			os.Setenv("AZURE_STORAGE_ACCOUNT", account)
		}
		if os.Getenv("AZURE_STORAGE_KEY") == "" {
			key := cfg.GetAzureStorageKey()
			if key == "" {
				return fmt.Errorf("Azure storage key not configured (set AZURE_STORAGE_KEY env var or config file)")
			}
			os.Setenv("AZURE_STORAGE_KEY", key)
		}
	} else if strings.HasPrefix(storageURL, "s3://") {
		// S3 requires AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY
		// Only set from config if not already in environment
		// Note: For IAM roles, these may not be needed
		if os.Getenv("AWS_ACCESS_KEY_ID") == "" {
			accessKey := cfg.GetAWSAccessKeyID()
			if accessKey != "" {
				os.Setenv("AWS_ACCESS_KEY_ID", accessKey)
			}
		}
		if os.Getenv("AWS_SECRET_ACCESS_KEY") == "" {
			secretKey := cfg.GetAWSSecretAccessKey()
			if secretKey != "" {
				os.Setenv("AWS_SECRET_ACCESS_KEY", secretKey)
			}
		}
	}
	// mem:// doesn't need credentials
	return nil
}

// NewStorage creates and returns a Storage implementation based on the provided configuration.
// It detects the storage mode based on the OBJECT_STORAGE_URL configuration.
// If OBJECT_STORAGE_URL is set, object storage (Azure, S3, or in-memory) is used.
// Otherwise, filesystem storage is used as the fallback.
func NewStorage(ctx context.Context, cfg StorageConfig) (Storage, error) {
	if cfg == nil {
		return nil, fmt.Errorf("storage configuration is required")
	}

	// Check if object storage is configured
	storageURL := cfg.GetObjectStorageURL()
	if storageURL != "" {
		// Create ObjectStore-based storage
		expiry := cfg.GetObjectStorageExpiry()
		if expiry == 0 {
			expiry = 168 * time.Hour // Default 7 days
		}

		// Set provider-specific environment variables that Go CDK expects
		// These must be set before calling blob.OpenBucket
		if err := setProviderEnvironment(storageURL, cfg); err != nil {
			return nil, fmt.Errorf("failed to set provider environment: %w", err)
		}

		objectStore, err := NewObjectStore(ctx, storageURL, expiry)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize object storage: %w", err)
		}

		// For Azure, set the account name if provided
		// This is needed for canonical URL generation
		azureAccount := cfg.GetAzureStorageAccount()
		if azureAccount != "" {
			objectStore.SetAzureAccount(azureAccount)
		}

		return NewObjectStoreStorage(objectStore), nil
	}

	// Use filesystem storage as fallback
	return NewFilesystemStorage(cfg.GetWorkspaceRoot()), nil
}
