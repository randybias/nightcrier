// Package storage provides Azure Blob Storage implementation for incident artifacts.
package storage

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/sas"
)

// AzureStorage implements the Storage interface for Azure Blob Storage.
type AzureStorage struct {
	client      *azblob.Client
	accountName string
	accountKey  string
	container   string
	sasExpiry   time.Duration
}

// AzureStorageConfig holds configuration for Azure Blob Storage.
type AzureStorageConfig struct {
	// ConnectionString is the full Azure connection string (optional, alternative to AccountName+AccountKey)
	ConnectionString string
	// AccountName is the storage account name (required if ConnectionString not provided)
	AccountName string
	// AccountKey is the storage account access key (required if ConnectionString not provided)
	AccountKey string
	// Container is the blob container name (required)
	Container string
	// SASExpiry is the duration for SAS token expiration (default: 168h / 7 days)
	SASExpiry time.Duration
}

// NewAzureStorage creates a new Azure Blob Storage client.
// It supports both connection string and account+key authentication.
func NewAzureStorage(cfg *AzureStorageConfig) (*AzureStorage, error) {
	if cfg == nil {
		return nil, fmt.Errorf("azure storage configuration is required")
	}

	if cfg.Container == "" {
		return nil, fmt.Errorf("container name is required")
	}

	// Set default SAS expiry if not provided
	sasExpiry := cfg.SASExpiry
	if sasExpiry == 0 {
		sasExpiry = 168 * time.Hour // 7 days default
	}

	var client *azblob.Client
	var accountName, accountKey string
	var err error

	// Try connection string first
	if cfg.ConnectionString != "" {
		client, err = azblob.NewClientFromConnectionString(cfg.ConnectionString, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create Azure client from connection string: %w", err)
		}
		// Parse connection string to extract account name and key for SAS generation
		accountName, accountKey, err = parseConnectionString(cfg.ConnectionString)
		if err != nil {
			return nil, fmt.Errorf("failed to parse connection string: %w", err)
		}
	} else if cfg.AccountName != "" && cfg.AccountKey != "" {
		// Use account name and key
		accountName = cfg.AccountName
		accountKey = cfg.AccountKey
		credential, err := azblob.NewSharedKeyCredential(accountName, accountKey)
		if err != nil {
			return nil, fmt.Errorf("failed to create shared key credential: %w", err)
		}
		serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net/", accountName)
		client, err = azblob.NewClientWithSharedKeyCredential(serviceURL, credential, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create Azure client with shared key: %w", err)
		}
	} else {
		return nil, fmt.Errorf("either connection string or (account name + key) must be provided")
	}

	return &AzureStorage{
		client:      client,
		accountName: accountName,
		accountKey:  accountKey,
		container:   cfg.Container,
		sasExpiry:   sasExpiry,
	}, nil
}

// parseConnectionString extracts account name and key from a connection string.
func parseConnectionString(connStr string) (string, string, error) {
	// Simple parser for connection string format:
	// "DefaultEndpointsProtocol=https;AccountName=xxx;AccountKey=yyy;EndpointSuffix=core.windows.net"
	var accountName, accountKey string

	// Split by semicolon
	parts := map[string]string{}
	current := ""
	inValue := false
	key := ""

	for i := 0; i < len(connStr); i++ {
		if connStr[i] == '=' && !inValue {
			key = current
			current = ""
			inValue = true
		} else if connStr[i] == ';' && inValue {
			parts[key] = current
			current = ""
			key = ""
			inValue = false
		} else {
			current += string(connStr[i])
		}
	}
	// Add last part
	if key != "" && inValue {
		parts[key] = current
	}

	accountName = parts["AccountName"]
	accountKey = parts["AccountKey"]

	if accountName == "" || accountKey == "" {
		return "", "", fmt.Errorf("connection string must contain AccountName and AccountKey")
	}

	return accountName, accountKey, nil
}

// uploadBlob uploads data to a blob at the specified path with appropriate content-type.
func (a *AzureStorage) uploadBlob(ctx context.Context, blobPath string, data []byte) error {
	blobClient := a.client.ServiceClient().NewContainerClient(a.container).NewBlockBlobClient(blobPath)

	// Determine content-type based on file extension
	contentType := getContentType(blobPath)

	// Set HTTP headers for in-browser rendering
	httpHeaders := &blob.HTTPHeaders{
		BlobContentType:        &contentType,
		BlobContentDisposition: stringPtr("inline"), // Render in browser instead of download
	}

	_, err := blobClient.UploadBuffer(ctx, data, &azblob.UploadBufferOptions{
		HTTPHeaders: httpHeaders,
	})
	if err != nil {
		return fmt.Errorf("failed to upload blob %s: %w", blobPath, err)
	}

	return nil
}

// getContentType returns the appropriate MIME type for a file based on its extension.
func getContentType(filename string) string {
	if len(filename) == 0 {
		return "application/octet-stream"
	}

	// Get file extension
	ext := ""
	for i := len(filename) - 1; i >= 0; i-- {
		if filename[i] == '.' {
			ext = filename[i:]
			break
		}
	}

	switch ext {
	case ".md":
		return "text/markdown; charset=utf-8"
	case ".json":
		return "application/json; charset=utf-8"
	case ".txt":
		return "text/plain; charset=utf-8"
	case ".html":
		return "text/html; charset=utf-8"
	default:
		return "application/octet-stream"
	}
}

// stringPtr returns a pointer to a string value.
func stringPtr(s string) *string {
	return &s
}

// generateIndexHTML creates an HTML index page for browsing incident artifacts.
func generateIndexHTML(incidentID string, artifactURLs map[string]string, expiresAt time.Time) string {
	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Incident Report: %s</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            max-width: 900px;
            margin: 40px auto;
            padding: 20px;
            background-color: #f5f5f5;
        }
        .container {
            background: white;
            border-radius: 8px;
            padding: 30px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        h1 {
            color: #333;
            margin-top: 0;
            font-size: 28px;
        }
        .incident-id {
            color: #666;
            font-size: 14px;
            margin-bottom: 20px;
        }
        .file-list {
            list-style: none;
            padding: 0;
        }
        .file-item {
            padding: 15px;
            margin: 10px 0;
            background: #f8f9fa;
            border-radius: 4px;
            border-left: 4px solid #007bff;
            transition: background 0.2s;
        }
        .file-item:hover {
            background: #e9ecef;
        }
        .file-link {
            text-decoration: none;
            color: #007bff;
            font-weight: 500;
            font-size: 16px;
        }
        .file-link:hover {
            text-decoration: underline;
        }
        .file-description {
            color: #666;
            font-size: 14px;
            margin-top: 5px;
        }
        .expiry-notice {
            margin-top: 30px;
            padding: 15px;
            background: #fff3cd;
            border-left: 4px solid #ffc107;
            border-radius: 4px;
            color: #856404;
        }
        .badge {
            display: inline-block;
            padding: 3px 8px;
            font-size: 12px;
            border-radius: 3px;
            margin-left: 10px;
        }
        .badge-primary {
            background: #007bff;
            color: white;
        }
        .badge-success {
            background: #28a745;
            color: white;
        }
        .badge-secondary {
            background: #6c757d;
            color: white;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>🔍 Kubernetes Incident Report</h1>
        <div class="incident-id">Incident ID: <code>%s</code></div>

        <ul class="file-list">`, incidentID, incidentID)

	// Define file descriptions
	fileDescriptions := map[string]struct {
		name        string
		description string
		badge       string
	}{
		"investigation.html": {"Investigation Report", "Formatted HTML report with root cause analysis", "primary"},
		"investigation.md":   {"Investigation Report (Raw)", "Markdown source for programmatic access", "secondary"},
		"incident.json":      {"Incident Data", "Complete incident context including event, status, and result metadata", "success"},
	}

	// Sort files for consistent display
	orderedFiles := []string{"investigation.html", "investigation.md", "incident.json"}
	for _, filename := range orderedFiles {
		if url, exists := artifactURLs[filename]; exists {
			desc := fileDescriptions[filename]
			html += fmt.Sprintf(`
            <li class="file-item">
                <div>
                    <a href="%s" class="file-link" target="_blank">📄 %s</a>
                    <span class="badge badge-%s">%s</span>
                </div>
                <div class="file-description">%s</div>
            </li>`, url, desc.name, desc.badge, filename, desc.description)
		}
	}

	html += fmt.Sprintf(`
        </ul>

        <div class="expiry-notice">
            ⏰ <strong>Access Expiration:</strong> These links will expire on %s (UTC)
        </div>
    </div>
</body>
</html>`, expiresAt.UTC().Format("2006-01-02 15:04:05"))

	return html
}

// generateSASURL generates a Service SAS URL for the specified blob with expiration.
func (a *AzureStorage) generateSASURL(blobPath string, expiry time.Time) (string, error) {
	// Create shared key credential for SAS signing
	credential, err := azblob.NewSharedKeyCredential(a.accountName, a.accountKey)
	if err != nil {
		return "", fmt.Errorf("failed to create credential for SAS: %w", err)
	}

	// Create blob client for the specific blob
	blobClient := a.client.ServiceClient().NewContainerClient(a.container).NewBlobClient(blobPath)

	// Build SAS permissions
	permissions := sas.BlobPermissions{Read: true}

	// Build SAS query parameters for Service SAS
	sasQueryParams, err := sas.BlobSignatureValues{
		Protocol:      sas.ProtocolHTTPS,
		StartTime:     time.Now().UTC(),
		ExpiryTime:    expiry.UTC(),
		Permissions:   permissions.String(),
		ContainerName: a.container,
		BlobName:      blobPath,
	}.SignWithSharedKey(credential)
	if err != nil {
		return "", fmt.Errorf("failed to generate SAS token for %s: %w", blobPath, err)
	}

	// Construct the full URL with SAS token
	sasURL := fmt.Sprintf("%s?%s", blobClient.URL(), sasQueryParams.Encode())
	return sasURL, nil
}

// SaveIncident implements the Storage interface for Azure Blob Storage.
// It uploads all incident artifacts to Azure and returns SAS URLs for access.
func (a *AzureStorage) SaveIncident(ctx context.Context, incidentID string, artifacts *IncidentArtifacts) (*SaveResult, error) {
	if artifacts == nil {
		return nil, fmt.Errorf("artifacts cannot be nil")
	}

	// Calculate expiration time
	expiresAt := time.Now().Add(a.sasExpiry)

	// Define artifact mappings
	artifactFiles := map[string][]byte{
		"incident.json":        artifacts.IncidentJSON,
		"investigation.md":     artifacts.InvestigationMD,
		"investigation.html":   artifacts.InvestigationHTML,
	}

	result := &SaveResult{
		ArtifactURLs: make(map[string]string),
		ExpiresAt:    expiresAt,
	}

	// Upload each artifact and generate SAS URLs
	var lastError error
	fileList := []string{} // Track uploaded files for index generation

	for filename, data := range artifactFiles {
		if len(data) == 0 {
			log.Printf("Warning: skipping empty artifact %s for incident %s", filename, incidentID)
			continue
		}

		blobPath := fmt.Sprintf("%s/%s", incidentID, filename)

		// Upload the blob
		if err := a.uploadBlob(ctx, blobPath, data); err != nil {
			log.Printf("Error uploading %s for incident %s: %v", filename, incidentID, err)
			lastError = err
			continue // Continue with other artifacts
		}

		// Generate SAS URL
		sasURL, err := a.generateSASURL(blobPath, expiresAt)
		if err != nil {
			log.Printf("Error generating SAS URL for %s: %v", filename, err)
			lastError = err
			continue // Continue with other artifacts
		}

		result.ArtifactURLs[filename] = sasURL
		fileList = append(fileList, filename)
	}

	// Generate and upload index.html for browsing
	if len(fileList) > 0 {
		indexHTML := generateIndexHTML(incidentID, result.ArtifactURLs, expiresAt)
		indexPath := fmt.Sprintf("%s/index.html", incidentID)

		if err := a.uploadBlob(ctx, indexPath, []byte(indexHTML)); err != nil {
			log.Printf("Warning: failed to upload index.html for %s: %v", incidentID, err)
		} else {
			// Generate SAS URL for the index page - this becomes the ReportURL
			indexSASURL, err := a.generateSASURL(indexPath, expiresAt)
			if err != nil {
				log.Printf("Warning: failed to generate SAS URL for index.html: %v", err)
			} else {
				result.ReportURL = indexSASURL
				result.ArtifactURLs["index.html"] = indexSASURL
				log.Printf("INFO: Set ReportURL to index.html: %s", indexSASURL)
			}
		}
	}

	// If we failed to upload any artifacts, but got at least one success, return partial results
	if len(result.ArtifactURLs) == 0 {
		if lastError != nil {
			return nil, fmt.Errorf("failed to upload any artifacts: %w", lastError)
		}
		return nil, fmt.Errorf("no artifacts were uploaded")
	}

	return result, nil
}
