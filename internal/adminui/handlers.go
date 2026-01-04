package adminui

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
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

// Server serves the admin UI.
type Server struct {
	store        *Store
	tmpl         *template.Template
	objectSigner ObjectSigner
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
}

// NewServer creates a new admin UI server.
func NewServer(cfg Config) (*Server, error) {
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
