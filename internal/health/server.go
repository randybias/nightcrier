package health

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/rbias/nightcrier/internal/cluster"
)

// ClusterHealth represents the health status of a single cluster connection.
// Design reference: design.md lines 551-561
type ClusterHealth struct {
	Name          string                       `json:"name"`
	Status        cluster.ConnectionStatus     `json:"status"`
	LastEvent     *time.Time                   `json:"last_event,omitempty"`
	LastError     string                       `json:"error,omitempty"`
	RetryIn       string                       `json:"retry_in,omitempty"`
	EventCount    int64                        `json:"event_count"`
	TriageEnabled bool                         `json:"triage_enabled"`
	Permissions   *cluster.ClusterPermissions  `json:"permissions,omitempty"`
	Labels        map[string]string            `json:"labels,omitempty"`
}

// HealthSummary is the top-level response structure for the health endpoint.
// Design reference: design.md lines 563-571
type HealthSummary struct {
	Clusters []ClusterHealth `json:"clusters"`
	Summary  struct {
		Total         int `json:"total"`
		Active        int `json:"active"`
		Unhealthy     int `json:"unhealthy"`
		TriageEnabled int `json:"triage_enabled"`
	} `json:"summary"`
}

// ConnectionManagerHealth defines the interface for accessing cluster health data.
// This allows the health server to work with the ConnectionManager without
// importing it directly (avoiding potential circular dependencies).
// Note: GetHealth() returns interface{} to avoid circular dependency issues.
type ConnectionManagerHealth interface {
	GetHealth() interface{}
}

// Server provides HTTP health monitoring endpoints for cluster connections.
type Server struct {
	manager ConnectionManagerHealth
	addr    string
}

// NewServer creates a new health monitoring server.
//
// Parameters:
//   - manager: The ConnectionManager to query for health status
//   - port: The port to listen on (default: 8080)
//
// Returns a new Server instance ready to be started.
func NewServer(manager ConnectionManagerHealth, port int) *Server {
	if port == 0 {
		port = 8080
	}

	return &Server{
		manager: manager,
		addr:    fmt.Sprintf(":%d", port),
	}
}

// Start begins serving health monitoring endpoints.
// This is a blocking call that should be run in a goroutine.
//
// Available endpoints:
//   - GET /health/clusters - Returns detailed cluster health status
//
// Parameters:
//   - ctx: Context for shutdown coordination (currently unused, for future graceful shutdown)
func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/health/clusters", s.handleClustersHealth)

	slog.Info("starting health server", "address", s.addr)
	return http.ListenAndServe(s.addr, mux)
}

// handleClustersHealth handles GET /health/clusters requests.
// Returns JSON with per-cluster health status and summary statistics.
func (s *Server) handleClustersHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get health summary from connection manager (returns interface{} due to import constraints)
	health := s.manager.GetHealth()

	// Set response headers
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Encode response - the interface{} will be marshaled as JSON
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(health); err != nil {
		slog.Error("failed to encode health response", "error", err)
	}
}
