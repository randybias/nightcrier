// Package k8s provides Kubernetes client initialization and utilities for the Nightcrier agent.
package k8s

import (
	"context"
	"fmt"
	"time"

	"github.com/rbias/nightcrier/internal/storage"
)

// OutputURLs contains presigned PUT URLs for agent outputs with expiration times.
// These URLs are passed to the agent container as environment variables.
type OutputURLs struct {
	// Report is the presigned PUT URL for report.md
	Report string
	// ReportExpiry is when the Report URL expires
	ReportExpiry time.Time

	// Log is the presigned PUT URL for agent.log
	Log string
	// LogExpiry is when the Log URL expires
	LogExpiry time.Time

	// Session is the presigned PUT URL for session.tar.gz
	Session string
	// SessionExpiry is when the Session URL expires
	SessionExpiry time.Time

	// Result is the presigned PUT URL for result.json
	Result string
	// ResultExpiry is when the Result URL expires
	ResultExpiry time.Time

	// Commands is the presigned PUT URL for commands-executed.log
	Commands string
	// CommandsExpiry is when the Commands URL expires
	CommandsExpiry time.Time
}

// ToPresignedURLs converts OutputURLs to PresignedURLs for use in JobConfig.
// The expiration times are discarded as they're not needed by the Job container.
func (o *OutputURLs) ToPresignedURLs() PresignedURLs {
	return PresignedURLs{
		Report:   o.Report,
		Log:      o.Log,
		Session:  o.Session,
		Result:   o.Result,
		Commands: o.Commands,
	}
}

// GenerateOutputURLs generates presigned PUT URLs for all agent outputs.
// The URLs are scoped to the incident: incidents/{incident-id}/results/{filename}
// URL expiration is set to jobTimeout + 5 minute buffer to allow time for uploads.
func GenerateOutputURLs(ctx context.Context, store *storage.ObjectStore, incidentID string, jobTimeout time.Duration) (*OutputURLs, error) {
	if store == nil {
		return nil, fmt.Errorf("object store cannot be nil")
	}
	if incidentID == "" {
		return nil, fmt.Errorf("incident ID cannot be empty")
	}
	if jobTimeout == 0 {
		return nil, fmt.Errorf("job timeout must be greater than zero")
	}

	// Set URL expiration to job timeout + 5 minute buffer
	// This ensures the container has time to upload outputs before URLs expire
	urlExpiry := jobTimeout + (5 * time.Minute)

	urls := &OutputURLs{}

	// Generate presigned PUT URL for report.md
	reportKey := fmt.Sprintf("incidents/%s/results/report.md", incidentID)
	reportURL, reportExpiry, err := store.SignedPutURL(ctx, reportKey, urlExpiry)
	if err != nil {
		return nil, fmt.Errorf("failed to generate report URL: %w", err)
	}
	urls.Report = reportURL
	urls.ReportExpiry = reportExpiry

	// Generate presigned PUT URL for agent.log
	logKey := fmt.Sprintf("incidents/%s/results/agent.log", incidentID)
	logURL, logExpiry, err := store.SignedPutURL(ctx, logKey, urlExpiry)
	if err != nil {
		return nil, fmt.Errorf("failed to generate log URL: %w", err)
	}
	urls.Log = logURL
	urls.LogExpiry = logExpiry

	// Generate presigned PUT URL for session.tar.gz
	sessionKey := fmt.Sprintf("incidents/%s/results/session.tar.gz", incidentID)
	sessionURL, sessionExpiry, err := store.SignedPutURL(ctx, sessionKey, urlExpiry)
	if err != nil {
		return nil, fmt.Errorf("failed to generate session URL: %w", err)
	}
	urls.Session = sessionURL
	urls.SessionExpiry = sessionExpiry

	// Generate presigned PUT URL for result.json
	resultKey := fmt.Sprintf("incidents/%s/results/result.json", incidentID)
	resultURL, resultExpiry, err := store.SignedPutURL(ctx, resultKey, urlExpiry)
	if err != nil {
		return nil, fmt.Errorf("failed to generate result URL: %w", err)
	}
	urls.Result = resultURL
	urls.ResultExpiry = resultExpiry

	// Generate presigned PUT URL for commands-executed.log
	commandsKey := fmt.Sprintf("incidents/%s/results/commands-executed.log", incidentID)
	commandsURL, commandsExpiry, err := store.SignedPutURL(ctx, commandsKey, urlExpiry)
	if err != nil {
		return nil, fmt.Errorf("failed to generate commands URL: %w", err)
	}
	urls.Commands = commandsURL
	urls.CommandsExpiry = commandsExpiry

	return urls, nil
}
