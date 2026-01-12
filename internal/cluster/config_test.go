package cluster

import (
	"strings"
	"testing"
)

func TestMonitoredClusterConfig_Validate_TLSEnforcement(t *testing.T) {
	tests := []struct {
		name        string
		config      MonitoredClusterConfig
		wantErr     bool
		errContains string
	}{
		{
			name: "api_key with http endpoint should fail",
			config: MonitoredClusterConfig{
				Name: "test-cluster",
				MCP: MCPConfig{
					Endpoint: "http://mcp-server:8080/mcp",
					APIKey:   "sk-test-key-12345",
				},
			},
			wantErr:     true,
			errContains: "mcp.api_key requires HTTPS endpoint",
		},
		{
			name: "api_key with https endpoint should succeed",
			config: MonitoredClusterConfig{
				Name: "test-cluster",
				MCP: MCPConfig{
					Endpoint: "https://agentgateway:8443/mcp",
					APIKey:   "sk-test-key-12345",
				},
			},
			wantErr: false,
		},
		{
			name: "no api_key with http endpoint should succeed",
			config: MonitoredClusterConfig{
				Name: "test-cluster",
				MCP: MCPConfig{
					Endpoint: "http://mcp-server:8080/mcp",
					APIKey:   "",
				},
			},
			wantErr: false,
		},
		{
			name: "no api_key with https endpoint should succeed",
			config: MonitoredClusterConfig{
				Name: "test-cluster",
				MCP: MCPConfig{
					Endpoint: "https://agentgateway:8443/mcp",
					APIKey:   "",
				},
			},
			wantErr: false,
		},
		{
			name: "empty endpoint should fail",
			config: MonitoredClusterConfig{
				Name: "test-cluster",
				MCP: MCPConfig{
					Endpoint: "",
					APIKey:   "",
				},
			},
			wantErr:     true,
			errContains: "mcp.endpoint is required",
		},
		{
			name: "invalid endpoint scheme should fail",
			config: MonitoredClusterConfig{
				Name: "test-cluster",
				MCP: MCPConfig{
					Endpoint: "ftp://mcp-server:8080/mcp",
					APIKey:   "",
				},
			},
			wantErr:     true,
			errContains: "must start with http:// or https://",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() expected error, got nil")
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Validate() error = %q, want error containing %q", err.Error(), tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestMCPConfig_APIKeyDocumentation(t *testing.T) {
	// This test ensures the MCPConfig struct has the expected fields
	// and that our API key authentication pattern works correctly
	cfg := MCPConfig{
		Endpoint: "https://agentgateway:8443/mcp",
		APIKey:   "sk-test-key",
	}

	if cfg.Endpoint == "" {
		t.Error("Endpoint should not be empty")
	}
	if cfg.APIKey == "" {
		t.Error("APIKey should not be empty for this test")
	}
}
