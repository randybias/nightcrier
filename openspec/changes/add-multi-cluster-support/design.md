# Design: Multi-Cluster MCP Server Support

## Component Architecture

### Package Structure

```
internal/
├── cluster/
│   ├── registry.go      # Cluster configuration registry
│   ├── connection.go    # Single cluster connection lifecycle
│   └── manager.go       # Connection manager (orchestrates all)
├── config/
│   └── config.go        # Extended with ClusterConfig slice
└── events/
    ├── client.go        # Existing MCP client (unchanged)
    ├── router.go        # Event fan-in and routing
    └── event.go         # Extended with cluster metadata
```

### Data Structures

#### ClusterConfig

```go
// ClusterConfig defines a single cluster's connection details
type ClusterConfig struct {
    Name        string            `mapstructure:"name" validate:"required"`
    MCPEndpoint string            `mapstructure:"mcp_endpoint" validate:"required,url"`
    Kubeconfig  string            `mapstructure:"kubeconfig" validate:"required,file"`
    Labels      map[string]string `mapstructure:"labels"`

    // Per-cluster overrides (optional)
    SeverityThreshold *string `mapstructure:"severity_threshold"`
    Enabled           *bool   `mapstructure:"enabled"` // default: true
}
```

#### ClusterConnection

```go
// ClusterConnection manages the lifecycle of a single MCP connection
type ClusterConnection struct {
    config     *ClusterConfig
    client     *events.Client
    status     ConnectionStatus
    lastEvent  time.Time
    lastError  error
    retryCount int

    mu sync.RWMutex
}

type ConnectionStatus string

const (
    StatusDisconnected ConnectionStatus = "disconnected"
    StatusConnecting   ConnectionStatus = "connecting"
    StatusConnected    ConnectionStatus = "connected"
    StatusSubscribing  ConnectionStatus = "subscribing"
    StatusActive       ConnectionStatus = "active"
    StatusFailed       ConnectionStatus = "failed"
)
```

#### ConnectionManager

```go
// ConnectionManager orchestrates multiple cluster connections
type ConnectionManager struct {
    clusters    map[string]*ClusterConnection
    eventChan   chan *ClusterEvent
    healthChan  chan HealthUpdate
    transport   *http.Transport  // Shared across all connections

    mu sync.RWMutex
}

// ClusterEvent wraps a FaultEvent with cluster context
type ClusterEvent struct {
    ClusterName string
    Kubeconfig  string
    Labels      map[string]string
    Event       *events.FaultEvent
}
```

## Connection Lifecycle

### State Machine

```
                    ┌──────────────┐
                    │ DISCONNECTED │◄────────────────┐
                    └──────┬───────┘                 │
                           │ Start()                 │
                           ▼                         │
                    ┌──────────────┐                 │
              ┌────►│  CONNECTING  │─────────────────┤
              │     └──────┬───────┘   (timeout/     │
              │            │            error)       │
              │            │ connected               │
              │            ▼                         │
              │     ┌──────────────┐                 │
              │     │  CONNECTED   │─────────────────┤
              │     └──────┬───────┘   (error)       │
              │            │                         │
              │            │ SetLoggingLevel         │
              │            ▼                         │
              │     ┌──────────────┐                 │
              │     │ SUBSCRIBING  │─────────────────┤
              │     └──────┬───────┘   (error)       │
              │            │                         │
              │            │ events_subscribe        │
              │            ▼                         │
        retry │     ┌──────────────┐                 │
      (backoff)     │    ACTIVE    │─────────────────┘
              │     └──────────────┘   (disconnect)
              │            │
              │            │ Stop()
              └────────────┴───────────────────────►┌──────────┐
                                                    │ STOPPED  │
                                                    └──────────┘
```

### Reconnection Strategy

```go
type ReconnectConfig struct {
    InitialBackoff time.Duration // Default: 1s
    MaxBackoff     time.Duration // Default: 60s
    Multiplier     float64       // Default: 2.0
    Jitter         float64       // Default: 0.1 (10%)
}

func (c *ClusterConnection) reconnect(ctx context.Context) {
    backoff := c.config.InitialBackoff

    for {
        select {
        case <-ctx.Done():
            return
        case <-time.After(backoff):
            if err := c.connect(ctx); err == nil {
                return // Success
            }
            // Calculate next backoff with jitter
            backoff = min(backoff*Multiplier, MaxBackoff)
            backoff = addJitter(backoff, Jitter)
        }
    }
}
```

## HTTP Transport Configuration

### Shared Transport

All MCP connections share a single `http.Transport` for efficient connection pooling:

```go
func NewConnectionManager(cfg *config.Config) *ConnectionManager {
    transport := &http.Transport{
        // Connection pool settings
        MaxIdleConns:        200,           // Total idle connections
        MaxIdleConnsPerHost: 2,             // Per-host idle connections
        MaxConnsPerHost:     10,            // Max connections per host
        IdleConnTimeout:     90 * time.Second,

        // Timeouts
        TLSHandshakeTimeout:   10 * time.Second,
        ResponseHeaderTimeout: 30 * time.Second,
        ExpectContinueTimeout: 1 * time.Second,

        // Keep-alive
        DisableKeepAlives: false,

        // Force HTTP/2 for multiplexing (if server supports)
        ForceAttemptHTTP2: true,
    }

    return &ConnectionManager{
        transport: transport,
        // ...
    }
}
```

### Per-Connection Client

Each cluster connection uses the shared transport:

```go
func (cm *ConnectionManager) createClient(cluster *ClusterConfig) *events.Client {
    httpClient := &http.Client{
        Transport: cm.transport,
        Timeout:   0, // No timeout for SSE/streaming
    }

    return events.NewClientWithHTTPClient(cluster.MCPEndpoint, httpClient)
}
```

## Event Routing

### Fan-In Architecture

```go
func (cm *ConnectionManager) Start(ctx context.Context) <-chan *ClusterEvent {
    cm.eventChan = make(chan *ClusterEvent, cm.cfg.GlobalQueueSize)

    for _, conn := range cm.clusters {
        go cm.runConnection(ctx, conn)
    }

    return cm.eventChan
}

func (cm *ConnectionManager) runConnection(ctx context.Context, conn *ClusterConnection) {
    for {
        select {
        case <-ctx.Done():
            return
        default:
            events, err := conn.Subscribe(ctx)
            if err != nil {
                conn.reconnect(ctx)
                continue
            }

            // Fan-in events to global channel
            for event := range events {
                select {
                case cm.eventChan <- &ClusterEvent{
                    ClusterName: conn.config.Name,
                    Kubeconfig:  conn.config.Kubeconfig,
                    Labels:      conn.config.Labels,
                    Event:       event,
                }:
                default:
                    // Queue full, apply overflow policy
                    slog.Warn("event queue full, dropping event",
                        "cluster", conn.config.Name)
                }
            }
        }
    }
}
```

## Health Monitoring

### Health Status Structure

```go
type ClusterHealth struct {
    Name       string           `json:"name"`
    Status     ConnectionStatus `json:"status"`
    LastEvent  *time.Time       `json:"last_event,omitempty"`
    LastError  string           `json:"error,omitempty"`
    RetryIn    string           `json:"retry_in,omitempty"`
    EventCount int64            `json:"event_count"`
    Labels     map[string]string `json:"labels,omitempty"`
}

type HealthSummary struct {
    Clusters []ClusterHealth `json:"clusters"`
    Summary  struct {
        Total     int `json:"total"`
        Active    int `json:"active"`
        Unhealthy int `json:"unhealthy"`
    } `json:"summary"`
}
```

### HTTP Health Endpoint

```go
// Optional: Add health server
func (cm *ConnectionManager) ServeHealth(addr string) error {
    mux := http.NewServeMux()

    mux.HandleFunc("/health", cm.handleHealthCheck)
    mux.HandleFunc("/health/clusters", cm.handleClusterHealth)

    return http.ListenAndServe(addr, mux)
}
```

## Configuration Loading

### Extended Config Structure

```go
type Config struct {
    // Single-cluster mode (backwards compatible)
    MCPEndpoint string `mapstructure:"mcp_endpoint"`

    // Multi-cluster mode
    Clusters []ClusterConfig `mapstructure:"clusters"`

    // Shared settings
    MaxConcurrentAgents int `mapstructure:"max_concurrent_agents"`
    GlobalQueueSize     int `mapstructure:"global_queue_size"`
    // ...
}

func (c *Config) Validate() error {
    // Mutual exclusivity check
    hasSingle := c.MCPEndpoint != ""
    hasMulti := len(c.Clusters) > 0

    if hasSingle && hasMulti {
        return fmt.Errorf("cannot specify both mcp_endpoint and clusters")
    }

    if !hasSingle && !hasMulti {
        return fmt.Errorf("must specify either mcp_endpoint or clusters")
    }

    // Validate each cluster config
    if hasMulti {
        names := make(map[string]bool)
        for i, cluster := range c.Clusters {
            if names[cluster.Name] {
                return fmt.Errorf("duplicate cluster name: %s", cluster.Name)
            }
            names[cluster.Name] = true

            if err := cluster.Validate(); err != nil {
                return fmt.Errorf("cluster[%d] %s: %w", i, cluster.Name, err)
            }
        }
    }

    return nil
}
```

## Agent Execution Context

### Kubeconfig Injection

The agent executor needs the cluster-specific kubeconfig:

```go
type ExecutorConfig struct {
    ScriptPath       string
    SystemPromptFile string
    AllowedTools     string
    Model            string
    Timeout          int
    Kubeconfig       string  // NEW: Cluster-specific kubeconfig
}

func (e *Executor) Execute(ctx context.Context, workspace, incidentID string) (int, error) {
    cmd := exec.CommandContext(ctx, e.config.ScriptPath,
        "-w", workspace,
        "-m", e.config.Model,
        "-k", e.config.Kubeconfig,  // Pass kubeconfig
        // ...
    )
    // ...
}
```

## Memory and Resource Estimates

### Per-Connection Overhead

| Component | Memory | Notes |
|-----------|--------|-------|
| HTTP Client | ~50 KB | Buffers, TLS state |
| MCP Session | ~100 KB | Protocol state, message buffers |
| Event Buffer | ~200 KB | 100 events @ ~2KB each |
| Metadata | ~10 KB | Config, status, timestamps |
| **Total** | **~400 KB** | Per cluster connection |

### Scale Projections

| Clusters | Connection Memory | Event Buffer | Total |
|----------|------------------|--------------|-------|
| 10 | 4 MB | 10 MB | ~15 MB |
| 50 | 20 MB | 50 MB | ~75 MB |
| 100 | 40 MB | 100 MB | ~150 MB |

Plus baseline application memory (~50 MB).

## Concurrency Model

### Goroutine Structure

```
main goroutine
├── ConnectionManager.Start()
│   ├── connection[0] goroutine (reconnect loop)
│   ├── connection[1] goroutine
│   ├── ...
│   └── connection[N] goroutine
├── Event processing loop (single)
│   └── processEvent() → spawns agent
└── Health server goroutine (optional)
```

### Synchronization Points

1. **Event Channel**: Buffered channel for fan-in (lock-free)
2. **Cluster Map**: RWMutex for dynamic add/remove
3. **Connection Status**: Per-connection mutex for status updates
4. **Agent Semaphore**: Existing circuit breaker unchanged

## Error Handling

### Per-Connection Failures

```go
func (c *ClusterConnection) handleError(err error) {
    c.mu.Lock()
    defer c.mu.Unlock()

    c.lastError = err
    c.status = StatusFailed
    c.retryCount++

    slog.Error("cluster connection failed",
        "cluster", c.config.Name,
        "error", err,
        "retry_count", c.retryCount)

    // Emit health update
    c.manager.emitHealthUpdate(c.healthSnapshot())
}
```

### Circuit Breaker (Future)

After N consecutive failures, mark cluster as "circuit open" and reduce retry frequency.
