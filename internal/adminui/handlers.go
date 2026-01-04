package adminui

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"html/template"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	_ "embed"
)

//go:embed templates/*.html
var templatesFS embed.FS

//go:embed assets/nightcrier-headshot.jpeg
var logoImage []byte

// ObjectSigner can generate signed URLs for objects.
// SignedURL returns (signedURL, expiryTime, error) - we ignore expiryTime.
type ObjectSigner interface {
	SignedURL(ctx context.Context, key string) (string, time.Time, error)
}

// JobCanceller can cancel running K8s jobs.
type JobCanceller interface {
	CancelJob(ctx context.Context, namespace, jobName string) error
}

// ClusterInfo holds display information about a monitored cluster.
type ClusterInfo struct {
	Name          string
	Environment   string
	MCPEndpoint   string
	TriageEnabled bool
}

// Server serves the admin UI.
type Server struct {
	store        *Store
	tmpl         *template.Template
	objectSigner ObjectSigner
	jobCanceller JobCanceller
	namespace    string
	clusters     []ClusterInfo
	server       *http.Server
}

// Config holds configuration for the admin server.
type Config struct {
	// DB is the database connection
	DB *sql.DB
	// ListenAddr is the address to listen on (e.g., "127.0.0.1:8847")
	ListenAddr string
	// ObjectSigner generates signed URLs for artifacts (optional)
	ObjectSigner ObjectSigner
	// JobCanceller cancels running K8s jobs (optional, required for cancel action)
	JobCanceller JobCanceller
	// Namespace is the K8s namespace where triage jobs run (required for cancel action)
	Namespace string
	// Clusters is the list of monitored clusters to display
	Clusters []ClusterInfo
}

// NewServer creates a new admin UI server.
// The server only binds to loopback addresses (127.0.0.1 or localhost) for security.
func NewServer(cfg Config) (*Server, error) {
	// Validate that the listen address is loopback only (security requirement)
	host, _, err := net.SplitHostPort(cfg.ListenAddr)
	if err != nil {
		return nil, fmt.Errorf("invalid listen address %q: %w", cfg.ListenAddr, err)
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		return nil, fmt.Errorf("admin UI must bind to loopback address (127.0.0.1 or localhost), not %q", host)
	}
	if !strings.HasPrefix(host, "127.") && host != "localhost" && host != "::1" {
		return nil, fmt.Errorf("admin UI must bind to loopback address (127.0.0.1 or localhost), not %q", host)
	}

	// Parse templates with custom functions
	funcMap := template.FuncMap{
		"formatTime": func(t time.Time) string {
			return t.Format("2006-01-02 15:04:05")
		},
		"formatDuration": func(d time.Duration) string {
			if d < time.Minute {
				return fmt.Sprintf("%ds", int(d.Seconds()))
			}
			if d < time.Hour {
				return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
			}
			return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
		},
		"formatTimePtr": func(t *time.Time) string {
			if t == nil {
				return "-"
			}
			return t.Format("2006-01-02 15:04:05")
		},
		"truncate": func(s string, maxLen int) string {
			if len(s) <= maxLen {
				return s
			}
			return s[:maxLen-3] + "..."
		},
	}

	tmpl, err := template.New("").Funcs(funcMap).ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	store := NewStore(cfg.DB)

	mux := http.NewServeMux()
	s := &Server{
		store:        store,
		tmpl:         tmpl,
		objectSigner: cfg.ObjectSigner,
		jobCanceller: cfg.JobCanceller,
		namespace:    cfg.Namespace,
		clusters:     cfg.Clusters,
		server: &http.Server{
			Addr:         cfg.ListenAddr,
			Handler:      mux,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
		},
	}

	// Register routes
	mux.HandleFunc("/admin", s.handleAdmin)
	mux.HandleFunc("/admin/logo", s.handleLogo)
	mux.HandleFunc("/admin/incidents/", s.handleDeleteIncident)
	mux.HandleFunc("/admin/triages/", s.handleCancelTriage)
	mux.HandleFunc("/", s.handleRedirect)

	return s, nil
}

// handleLogo serves the logo image.
func (s *Server) handleLogo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Write(logoImage)
}

// Start starts the admin server (blocking).
func (s *Server) Start() error {
	slog.Info("starting admin UI server", "addr", s.server.Addr)
	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

// handleRedirect redirects / to /admin.
func (s *Server) handleRedirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/admin", http.StatusFound)
}

// incidentView wraps Incident with a signed URL.
type incidentView struct {
	Incident
	ViewURL string
}

// adminData holds data for the admin template.
type adminData struct {
	Clusters       []ClusterInfo
	RunningTriages []RunningTriage
	Incidents      []incidentView
	RefreshTime    time.Time
}

// handleAdmin serves the main admin dashboard.
func (s *Server) handleAdmin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Fetch data
	triages, err := s.store.GetRunningTriages(ctx)
	if err != nil {
		slog.Error("failed to get running triages", "error", err)
		http.Error(w, "Failed to load data", http.StatusInternalServerError)
		return
	}

	incidents, err := s.store.GetIncidents(ctx, 100)
	if err != nil {
		slog.Error("failed to get incidents", "error", err)
		http.Error(w, "Failed to load data", http.StatusInternalServerError)
		return
	}

	// Build incident views with signed URLs
	incidentViews := make([]incidentView, len(incidents))
	for i, inc := range incidents {
		incidentViews[i] = incidentView{Incident: inc}
		if s.objectSigner != nil {
			// Generate signed URL for the incident's index.html
			key := fmt.Sprintf("%s/index.html", inc.IncidentID)
			signedURL, _, err := s.objectSigner.SignedURL(ctx, key)
			if err != nil {
				slog.Debug("failed to sign URL", "incident_id", inc.IncidentID, "error", err)
			} else {
				incidentViews[i].ViewURL = signedURL
			}
		}
	}

	data := adminData{
		Clusters:       s.clusters,
		RunningTriages: triages,
		Incidents:      incidentViews,
		RefreshTime:    time.Now(),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, "admin.html", data); err != nil {
		slog.Error("failed to render template", "error", err)
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		return
	}
}

// handleDeleteIncident handles POST /admin/incidents/{id}/delete
func (s *Server) handleDeleteIncident(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract incident ID from path: /admin/incidents/{id}/delete
	path := strings.TrimPrefix(r.URL.Path, "/admin/incidents/")
	incidentID := strings.TrimSuffix(path, "/delete")
	if incidentID == "" || incidentID == path {
		http.Error(w, "Invalid incident ID", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	if err := s.store.DeleteIncident(ctx, incidentID); err != nil {
		slog.Error("failed to delete incident", "incident_id", incidentID, "error", err)
		http.Error(w, "Failed to delete incident", http.StatusInternalServerError)
		return
	}

	slog.Info("incident deleted", "incident_id", incidentID)
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

// handleCancelTriage handles POST /admin/triages/{id}/cancel
func (s *Server) handleCancelTriage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract execution ID from path: /admin/triages/{id}/cancel
	path := strings.TrimPrefix(r.URL.Path, "/admin/triages/")
	executionID := strings.TrimSuffix(path, "/cancel")
	if executionID == "" || executionID == path {
		http.Error(w, "Invalid execution ID", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// Cancel the K8s job if canceller is available
	if s.jobCanceller != nil {
		// Look up the running triage to get the incident ID for the job name
		triage, err := s.store.GetRunningTriageByExecutionID(ctx, executionID)
		if err != nil {
			slog.Warn("failed to look up triage for cancellation", "execution_id", executionID, "error", err)
		} else if triage != nil {
			// Job name is triage-{incidentID}
			jobName := fmt.Sprintf("triage-%s", triage.IncidentID)
			if err := s.jobCanceller.CancelJob(ctx, s.namespace, jobName); err != nil {
				slog.Warn("failed to cancel K8s job", "execution_id", executionID, "job_name", jobName, "error", err)
				// Continue to mark as cancelled even if job deletion fails
			}
		}
	}

	// Mark execution as cancelled in database
	if err := s.store.CancelExecution(ctx, executionID); err != nil {
		slog.Error("failed to cancel execution", "execution_id", executionID, "error", err)
		http.Error(w, "Failed to cancel execution", http.StatusInternalServerError)
		return
	}

	slog.Info("triage cancelled", "execution_id", executionID)
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}
