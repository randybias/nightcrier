// Package storage provides object storage implementation using Go CDK.
package storage

import (
	"context"
	"fmt"
	"log"
	"time"
)

// ObjectStoreStorage implements the Storage interface using Go CDK object storage.
type ObjectStoreStorage struct {
	store *ObjectStore
}

// NewObjectStoreStorage creates a new ObjectStoreStorage backed by an ObjectStore.
func NewObjectStoreStorage(store *ObjectStore) *ObjectStoreStorage {
	return &ObjectStoreStorage{
		store: store,
	}
}

// getContentType returns the appropriate MIME type for a file based on its extension.
func getContentTypeFromFilename(filename string) string {
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
	case ".log":
		return "text/plain; charset=utf-8"
	case ".gz":
		return "application/gzip"
	default:
		return "application/octet-stream"
	}
}

// generateIndexHTML creates an HTML index page for browsing incident artifacts.
func generateIndexHTMLWithSignedURLs(incidentID string, artifactURLs map[string]string, expiresAt time.Time) string {
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
        <h1>Kubernetes Incident Report</h1>
        <div class="incident-id">Incident ID: <code>%s</code></div>

        <ul class="file-list">`, incidentID, incidentID)

	// Define file descriptions
	fileDescriptions := map[string]struct {
		name        string
		description string
		badge       string
	}{
		"investigation.html":                {"Investigation Report", "Formatted HTML report with root cause analysis", "primary"},
		"investigation.md":                  {"Investigation Report (Raw)", "Markdown source for programmatic access", "secondary"},
		"incident.json":                     {"Incident Data", "Complete incident context including event, status, and result metadata", "success"},
		"incident_cluster_permissions.json": {"Cluster Permissions", "Validated Kubernetes permissions the agent had during investigation", "success"},
		"prompt-sent.md":                    {"Prompt Sent to Agent", "Full system prompt and additional context sent to the agent for audit", "secondary"},
		"agent-stdout.log":                  {"Agent Standard Output", "Agent's final output and results (DEBUG mode only)", "secondary"},
		"agent-stderr.log":                  {"Agent Standard Error", "Agent's diagnostic output and errors (DEBUG mode only)", "secondary"},
		"agent-full.log":                    {"Agent Combined Log", "Complete timestamped agent execution log (DEBUG mode only)", "secondary"},
		"agent-commands-executed.log":       {"Agent Commands Executed", "Bash commands run by the agent during investigation (DEBUG mode only)", "secondary"},
		"agent-session.tar.gz":              {"Agent Session Archive", "Complete agent session with turn history and internal logs (DEBUG mode only)", "secondary"},
	}

	// Sort files for consistent display - logs and session archive last since operators only need them for troubleshooting
	orderedFiles := []string{"investigation.html", "investigation.md", "incident.json", "incident_cluster_permissions.json", "prompt-sent.md", "agent-stdout.log", "agent-stderr.log", "agent-full.log", "agent-commands-executed.log", "agent-session.tar.gz"}
	for _, filename := range orderedFiles {
		if url, exists := artifactURLs[filename]; exists {
			desc := fileDescriptions[filename]
			html += fmt.Sprintf(`
            <li class="file-item">
                <div>
                    <a href="%s" class="file-link" target="_blank">%s</a>
                    <span class="badge badge-%s">%s</span>
                </div>
                <div class="file-description">%s</div>
            </li>`, url, desc.name, desc.badge, filename, desc.description)
		}
	}

	html += fmt.Sprintf(`
        </ul>

        <div class="expiry-notice">
            <strong>Access Expiration:</strong> These links will expire on %s (UTC)
        </div>
    </div>
</body>
</html>`, expiresAt.UTC().Format("2006-01-02 15:04:05"))

	return html
}

// SaveIncident implements the Storage interface for object storage.
// It uploads all incident artifacts to object storage and returns both signed and canonical URLs.
func (o *ObjectStoreStorage) SaveIncident(ctx context.Context, incidentID string, artifacts *IncidentArtifacts) (*SaveResult, error) {
	if artifacts == nil {
		return nil, fmt.Errorf("artifacts cannot be nil")
	}

	// Get expiration time for signed URLs
	var expiresAt time.Time

	// Define artifact mappings
	artifactFiles := map[string][]byte{
		"incident.json":                     artifacts.IncidentJSON,
		"investigation.md":                  artifacts.InvestigationMD,
		"investigation.html":                artifacts.InvestigationHTML,
		"incident_cluster_permissions.json": artifacts.ClusterPermissionsJSON,
		"prompt-sent.md":                    artifacts.PromptSent,
	}

	result := &SaveResult{
		ArtifactURLs:  make(map[string]string),
		CanonicalURLs: make(map[string]string),
		LogURLs:       make(map[string]string),
	}

	// Upload each artifact and generate URLs
	var lastError error
	fileList := []string{} // Track uploaded files for index generation

	for filename, data := range artifactFiles {
		if len(data) == 0 {
			log.Printf("Warning: skipping empty artifact %s for incident %s", filename, incidentID)
			continue
		}

		key := fmt.Sprintf("%s/%s", incidentID, filename)
		contentType := getContentTypeFromFilename(filename)

		// Upload the artifact
		if err := o.store.Upload(ctx, key, data, contentType); err != nil {
			log.Printf("Error uploading %s for incident %s: %v", filename, incidentID, err)
			lastError = err
			continue // Continue with other artifacts
		}

		// Generate signed URL for immediate clickability
		signedURL, expiry, err := o.store.SignedURL(ctx, key)
		if err != nil {
			log.Printf("Error generating signed URL for %s: %v", filename, err)
			lastError = err
			continue // Continue with other artifacts
		}

		// Generate canonical URL for long-term storage
		canonicalURL := o.store.CanonicalURL(key)

		result.ArtifactURLs[filename] = signedURL
		result.CanonicalURLs[filename] = canonicalURL
		expiresAt = expiry
		fileList = append(fileList, filename)
	}

	// Upload agent logs if present (DEBUG mode only)
	logFiles := map[string][]byte{
		"agent-stdout.log":            artifacts.AgentLogs.Stdout,
		"agent-stderr.log":            artifacts.AgentLogs.Stderr,
		"agent-full.log":              artifacts.AgentLogs.Combined,
		"agent-commands-executed.log": artifacts.AgentLogs.CommandsExecuted,
	}

	for filename, data := range logFiles {
		if len(data) == 0 {
			log.Printf("Info: skipping empty log file %s for incident %s", filename, incidentID)
			continue
		}

		key := fmt.Sprintf("%s/logs/%s", incidentID, filename)
		contentType := getContentTypeFromFilename(filename)

		// Upload the log file
		if err := o.store.Upload(ctx, key, data, contentType); err != nil {
			log.Printf("Error uploading %s for incident %s: %v", filename, incidentID, err)
			lastError = err
			continue // Continue with other logs
		}

		// Generate signed URL
		signedURL, expiry, err := o.store.SignedURL(ctx, key)
		if err != nil {
			log.Printf("Error generating signed URL for %s: %v", filename, err)
			lastError = err
			continue // Continue with other logs
		}

		// Generate canonical URL
		canonicalURL := o.store.CanonicalURL(key)

		result.LogURLs[filename] = signedURL
		result.CanonicalURLs[filename] = canonicalURL
		expiresAt = expiry
	}

	// Upload agent session archive if present (DEBUG mode only)
	if len(artifacts.AgentSessionArchive) > 0 {
		filename := "agent-session.tar.gz"
		key := fmt.Sprintf("%s/logs/%s", incidentID, filename)
		contentType := getContentTypeFromFilename(filename)

		if err := o.store.Upload(ctx, key, artifacts.AgentSessionArchive, contentType); err != nil {
			log.Printf("Error uploading agent session archive for incident %s: %v", incidentID, err)
			lastError = err
		} else {
			// Generate signed URL for session archive
			signedURL, expiry, err := o.store.SignedURL(ctx, key)
			if err != nil {
				log.Printf("Error generating signed URL for agent session archive: %v", err)
				lastError = err
			} else {
				// Generate canonical URL
				canonicalURL := o.store.CanonicalURL(key)

				result.LogURLs[filename] = signedURL
				result.CanonicalURLs[filename] = canonicalURL
				expiresAt = expiry
			}
		}
	}

	// Set expiration time
	result.ExpiresAt = expiresAt

	// Generate and upload index.html for browsing
	if len(fileList) > 0 {
		// Merge log URLs into artifact URLs for index.html generation
		allURLs := make(map[string]string)
		for k, v := range result.ArtifactURLs {
			allURLs[k] = v
		}
		for k, v := range result.LogURLs {
			allURLs[k] = v
		}

		indexHTML := generateIndexHTMLWithSignedURLs(incidentID, allURLs, expiresAt)
		indexKey := fmt.Sprintf("%s/index.html", incidentID)
		contentType := getContentTypeFromFilename("index.html")

		if err := o.store.Upload(ctx, indexKey, []byte(indexHTML), contentType); err != nil {
			log.Printf("Warning: failed to upload index.html for %s: %v", incidentID, err)
		} else {
			// Generate signed URL for the index page - this becomes the ReportURL
			indexSignedURL, _, err := o.store.SignedURL(ctx, indexKey)
			if err != nil {
				log.Printf("Warning: failed to generate signed URL for index.html: %v", err)
			} else {
				result.ReportURL = indexSignedURL
				result.ArtifactURLs["index.html"] = indexSignedURL
				// Generate canonical URL for index.html
				result.CanonicalURLs["index.html"] = o.store.CanonicalURL(indexKey)
				log.Printf("INFO: Set ReportURL to index.html: %s", indexSignedURL)
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
