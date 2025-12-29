// Package k8s provides Kubernetes client initialization and utilities for the Nightcrier agent.
package k8s

import (
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// ClientConfig holds configuration for creating a Kubernetes client.
type ClientConfig struct {
	// Kubeconfig is the path to the kubeconfig file for out-of-cluster access.
	// If empty, in-cluster configuration will be attempted.
	Kubeconfig string
	// Context is the kubeconfig context to use.
	// If empty, the current context from the kubeconfig file will be used.
	Context string
}

// Client wraps a Kubernetes clientset with additional context.
type Client struct {
	clientset kubernetes.Interface
	config    *rest.Config
	inCluster bool
}

// NewClient creates a new Kubernetes client.
// It automatically detects whether to use in-cluster or out-of-cluster configuration:
// - If Kubeconfig path is provided, uses that file
// - If KUBECONFIG env var is set, uses that file
// - If running in a pod (service account token exists), uses in-cluster config
// - Otherwise, tries the default kubeconfig location (~/.kube/config)
func NewClient(cfg ClientConfig) (*Client, error) {
	config, inCluster, err := buildConfig(cfg.Kubeconfig, cfg.Context)
	if err != nil {
		return nil, fmt.Errorf("failed to build kubernetes config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes clientset: %w", err)
	}

	return &Client{
		clientset: clientset,
		config:    config,
		inCluster: inCluster,
	}, nil
}

// Clientset returns the underlying Kubernetes clientset.
func (c *Client) Clientset() kubernetes.Interface {
	return c.clientset
}

// Config returns the Kubernetes REST config.
func (c *Client) Config() *rest.Config {
	return c.config
}

// IsInCluster returns true if the client is using in-cluster configuration.
func (c *Client) IsInCluster() bool {
	return c.inCluster
}

// buildConfig constructs a Kubernetes REST config, attempting multiple strategies
// in order of precedence:
// 1. Explicit kubeconfig path provided
// 2. KUBECONFIG environment variable
// 3. In-cluster config (if service account token exists)
// 4. Default kubeconfig location (~/.kube/config)
//
// If context is specified, it will be used; otherwise the current context is used.
func buildConfig(kubeconfigPath string, context string) (*rest.Config, bool, error) {
	// Strategy 1: Explicit kubeconfig path
	if kubeconfigPath != "" {
		config, err := buildConfigWithContext(kubeconfigPath, context)
		if err != nil {
			return nil, false, fmt.Errorf("failed to load kubeconfig from %s: %w", kubeconfigPath, err)
		}
		return config, false, nil
	}

	// Strategy 2: KUBECONFIG environment variable
	if envPath := os.Getenv("KUBECONFIG"); envPath != "" {
		config, err := buildConfigWithContext(envPath, context)
		if err != nil {
			return nil, false, fmt.Errorf("failed to load kubeconfig from KUBECONFIG env (%s): %w", envPath, err)
		}
		return config, false, nil
	}

	// Strategy 3: In-cluster config
	// Check if we're running inside a pod by looking for the service account token
	if _, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount/token"); err == nil {
		config, err := rest.InClusterConfig()
		if err != nil {
			return nil, false, fmt.Errorf("failed to load in-cluster config: %w", err)
		}
		return config, true, nil
	}

	// Strategy 4: Default kubeconfig location
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, false, fmt.Errorf("failed to get user home directory: %w", err)
	}

	defaultPath := filepath.Join(homeDir, ".kube", "config")
	if _, err := os.Stat(defaultPath); err == nil {
		config, err := buildConfigWithContext(defaultPath, context)
		if err != nil {
			return nil, false, fmt.Errorf("failed to load kubeconfig from default location (%s): %w", defaultPath, err)
		}
		return config, false, nil
	}

	return nil, false, fmt.Errorf("no kubeconfig found: provide explicit path, set KUBECONFIG env, run in-cluster, or create ~/.kube/config")
}

// buildConfigWithContext loads a kubeconfig and uses the specified context.
// If context is empty, uses the current context from the kubeconfig.
func buildConfigWithContext(kubeconfigPath string, context string) (*rest.Config, error) {
	// Load the kubeconfig file
	loadingRules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath}
	configOverrides := &clientcmd.ConfigOverrides{}

	// Override the context if specified
	if context != "" {
		configOverrides.CurrentContext = context
	}

	// Build the config with the specified context
	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
	config, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	return config, nil
}
