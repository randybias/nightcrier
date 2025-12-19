// Package storage provides interfaces and implementations for persisting incident artifacts.
package storage

import (
	"context"
	"fmt"
	"time"
)

// Storage defines the interface for persisting incident artifacts to local or cloud storage.
type Storage interface {
	// SaveIncident uploads all artifacts for an incident to storage.
	// It returns URLs to access the artifacts and metadata about the storage operation.
	SaveIncident(ctx context.Context, incidentID string, artifacts *IncidentArtifacts) (*SaveResult, error)
}

// IncidentArtifacts contains all files generated during incident investigation.
type IncidentArtifacts struct {
	// EventJSON is the serialized event that triggered the investigation
	EventJSON []byte
	// ResultJSON is the JSON result from the agent investigation
	ResultJSON []byte
	// InvestigationMD is the markdown investigation report
	InvestigationMD []byte
	// InvestigationHTML is the HTML-rendered version of the investigation report
	InvestigationHTML []byte
}

// SaveResult contains the results of a storage operation, including URLs to access artifacts.
type SaveResult struct {
	// ReportURL is the authenticated URL to the investigation report (investigation.md)
	ReportURL string
	// ArtifactURLs maps artifact names to their authenticated URLs
	// Common keys: "event.json", "result.json", "investigation.md"
	ArtifactURLs map[string]string
	// ExpiresAt is when the URLs expire (relevant for cloud storage with SAS tokens)
	ExpiresAt time.Time
}

// StorageConfig represents the configuration needed to initialize storage backends.
// This interface allows us to accept different config types without importing
// the concrete config package (avoiding circular dependencies).
type StorageConfig interface {
	// IsAzureStorageEnabled returns true if Azure storage should be used
	IsAzureStorageEnabled() bool
	// GetWorkspaceRoot returns the filesystem workspace root directory
	GetWorkspaceRoot() string
}

// AzureConfig provides Azure-specific configuration needed to initialize AzureStorage.
type AzureConfig interface {
	StorageConfig
	GetAzureConnectionString() string
	GetAzureAccount() string
	GetAzureKey() string
	GetAzureContainer() string
	GetAzureSASExpiry() time.Duration
}

// NewStorage creates and returns a Storage implementation based on the provided configuration.
// It detects the storage mode (Azure, filesystem, etc.) from the configuration.
// If AZURE_STORAGE_ACCOUNT or AZURE_STORAGE_CONNECTION_STRING is set, Azure storage is used.
// Otherwise, filesystem storage is used as the fallback.
func NewStorage(cfg StorageConfig) (Storage, error) {
	if cfg == nil {
		return nil, fmt.Errorf("storage configuration is required")
	}

	// Detect storage mode based on configuration
	if cfg.IsAzureStorageEnabled() {
		// Try to cast to AzureConfig interface
		azureCfg, ok := cfg.(AzureConfig)
		if !ok {
			return nil, fmt.Errorf("Azure storage enabled but config doesn't implement AzureConfig interface")
		}

		// Create Azure storage backend
		azureStorage, err := NewAzureStorage(&AzureStorageConfig{
			ConnectionString: azureCfg.GetAzureConnectionString(),
			AccountName:      azureCfg.GetAzureAccount(),
			AccountKey:       azureCfg.GetAzureKey(),
			Container:        azureCfg.GetAzureContainer(),
			SASExpiry:        azureCfg.GetAzureSASExpiry(),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to initialize Azure storage: %w", err)
		}
		return azureStorage, nil
	}

	// Use filesystem storage as fallback
	return NewFilesystemStorage(cfg.GetWorkspaceRoot()), nil
}
