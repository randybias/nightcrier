# Design: Automate Kubernetes Bootstrap

## Overview

This change moves Kubernetes resource provisioning from `scripts/dev-setup.sh` into the core application startup sequence in `cmd/nightcrier/main.go`. The application will self-provision its runtime dependencies on every startup.

## Architecture

### Component: Bootstrap Manager

A new `internal/bootstrap` package will handle Kubernetes resource provisioning:

```
internal/bootstrap/
  ├── bootstrap.go       # Main bootstrap orchestrator
  ├── namespace.go       # Namespace creation
  ├── rbac.go            # ServiceAccount, Role, RoleBinding
  ├── secrets.go         # Secret creation (API keys, kubeconfigs)
  └── bootstrap_test.go  # Unit tests
```

### Startup Sequence

```
main.go startup:
1. Load configuration
2. Validate configuration (existing)
3. [NEW] Initialize Kubernetes client
4. [NEW] Run bootstrap (idempotent)
   - Check/create namespace
   - Check/create RBAC resources
   - Check/create ai-api-keys Secret
   - For each cluster:
     - Read triage.kubeconfig file
     - Check/create kubeconfig-{cluster} Secret
5. Initialize components (existing)
6. Start server
```

### Bootstrap Manager Interface

```go
type Bootstrapper interface {
    Bootstrap(ctx context.Context) error
}

type Config struct {
    // Kubernetes connection
    KubeconfigPath string
    Namespace      string

    // API keys (from main config)
    AnthropicAPIKey string
    OpenAIAPIKey    string
    GeminiAPIKey    string

    // Cluster configs (from main config)
    Clusters []ClusterConfig
}

type ClusterConfig struct {
    Name              string
    TriageKubeconfig  string  // Path to kubeconfig file
}
```

## Resource Management

### Namespace Creation

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: {config.Namespace}
  labels:
    app: nightcrier
    managed-by: nightcrier-bootstrap
```

**Logic:**
- Check if namespace exists via `GET /api/v1/namespaces/{name}`
- If 404, create via `POST /api/v1/namespaces`
- If exists, skip (idempotent)

### RBAC Creation

ServiceAccount, Role, and RoleBinding are created as defined in `deploy/dev/rbac.yaml`.

**Logic:**
- Check ServiceAccount: `GET /api/v1/namespaces/{ns}/serviceaccounts/nightcrier-executor`
- Check Role: `GET /apis/rbac.authorization.k8s.io/v1/namespaces/{ns}/roles/nightcrier-executor`
- Check RoleBinding: `GET /apis/rbac.authorization.k8s.io/v1/namespaces/{ns}/rolebindings/nightcrier-executor`
- Create missing resources (idempotent)

**Permissions required:**
The application (or user running it) needs:
- `create`, `get` on `namespaces`
- `create`, `get` on `serviceaccounts`, `roles`, `rolebindings`
- `create`, `get` on `secrets`

### Secret Creation

#### ai-api-keys Secret

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: ai-api-keys
  namespace: {config.Namespace}
type: Opaque
stringData:
  anthropic: {config.AnthropicAPIKey}
  openai:    {config.OpenAIAPIKey}
  gemini:    {config.GeminiAPIKey}
```

**Logic:**
- Check if Secret exists: `GET /api/v1/namespaces/{ns}/secrets/ai-api-keys`
- If exists, skip (never update existing secrets)
- If not exists:
  - Require at least one API key to be non-empty
  - Create Secret with all three keys (empty values allowed)

**Note:** We never update existing Secrets. If keys need updating, users must manually delete and recreate, or edit the Secret directly.

#### kubeconfig-{cluster} Secrets

For each cluster in configuration:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: kubeconfig-{cluster.Name}
  namespace: {config.Namespace}
  labels:
    app: nightcrier
    cluster: {cluster.Name}
type: Opaque
stringData:
  config: {contents of cluster.TriageKubeconfig file}
```

**Logic:**
- For each cluster where `triage.enabled == true`:
  - Validate `triage.kubeconfig` file exists and is readable
  - Read file contents
  - Check if Secret exists: `GET /api/v1/namespaces/{ns}/secrets/kubeconfig-{cluster.Name}`
  - If exists, skip
  - If not exists, create with file contents

**Error handling:**
- If kubeconfig file doesn't exist: Fatal error with clear message
- If file is not readable: Fatal error with permission guidance
- If cluster name is invalid (k8s label constraints): Fatal error

## Error Handling

### Permission Errors

If the application lacks Kubernetes permissions:

```
ERROR: Failed to bootstrap Kubernetes resources: forbidden: User "system:serviceaccount:default:default"
cannot create resource "namespaces" in API group "" at the cluster scope

Nightcrier requires permissions to create namespace, RBAC, and Secrets.

For local development (kind):
  kubectl apply -f deploy/dev/rbac.yaml

For remote clusters:
  Ensure your kubeconfig user has cluster-admin or appropriate permissions

For in-cluster deployment:
  Apply manifests in deploy/ directory before deploying Nightcrier
```

### File Errors

If kubeconfig file is missing:

```
ERROR: Cluster 'prod-us-east-1' requires triage kubeconfig but file not found:
  Path: /path/to/prod-kubeconfig.yaml

Please ensure:
  1. File exists at the specified path
  2. File is readable by Nightcrier process
  3. Path is absolute or relative to working directory
```

### Partial Failures

Bootstrap is atomic - if any step fails, the application exits. We don't attempt partial recovery or skip broken clusters.

## Configuration Integration

No new configuration fields required. Use existing fields:

```yaml
# Main config.yaml
kubeconfig_path: "/path/to/admin-kubeconfig.yaml"  # For bootstrap connection
k8s_namespace: "nightcrier"                        # Namespace to bootstrap

anthropic_api_key: "sk-ant-..."  # Used for ai-api-keys Secret
openai_api_key: "sk-..."
gemini_api_key: "..."

clusters:
  - name: prod-cluster
    triage:
      enabled: true
      kubeconfig: "/path/to/prod-readonly-kubeconfig.yaml"  # Used for Secret
```

**Key points:**
- `kubeconfig_path` - Admin kubeconfig for nightcrier to connect and bootstrap
- `triage.kubeconfig` - Read-only kubeconfig that agents will use (stored in Secret)
- These are typically different files with different permissions

## Impact on dev-setup.sh

After this change, `scripts/dev-setup.sh` should be simplified to:

```bash
# Only handle kind-specific setup
1. Create kind cluster
2. Load nc-agent-runner image
3. Done - let Nightcrier bootstrap itself
```

Remove:
- `apply_manifests()` - Nightcrier does this
- `create_secrets()` - Nightcrier does this
- `create_kubeconfig_secret()` - Nightcrier does this

The script becomes purely for local kind cluster creation, not Kubernetes resource setup.

## Testing Strategy

### Unit Tests
- `bootstrap_test.go` - Mock Kubernetes client, verify API calls
- Test idempotency (resources already exist)
- Test error handling (permission failures, missing files)

### Integration Tests
- Test against real kind cluster
- Verify namespace, RBAC, Secrets created correctly
- Test with multiple clusters in config
- Test with missing kubeconfig files (should fail)

### Manual Testing
- Local kind cluster (current workflow)
- Remote cluster deployment (new capability)
- In-cluster deployment as Pod (future)

## Deployment Impact

### Existing Deployments (kind)

No breaking changes. Existing kind clusters with manual setup will continue working:
- Bootstrap checks if resources exist
- Skips creation if already present
- No migration required

### New Deployments (remote)

Now possible without manual setup:
1. Configure kubeconfig_path pointing to remote cluster
2. Configure cluster triage.kubeconfig paths
3. Run nightcrier - it bootstraps itself
4. Start processing events

### Future: In-Cluster Deployment

When running as Pod in Kubernetes:
- Use in-cluster ServiceAccount (no kubeconfig_path needed)
- Bootstrap runs same as external deployment
- Requires pre-existing ServiceAccount with bootstrap permissions

## Security Considerations

1. **API key secrets** - Only created if they don't exist, never updated
   - Users must manually manage key rotation
   - Consider adding `--force-update-secrets` flag in future if needed

2. **Kubeconfig file permissions** - Bootstrap only reads files, doesn't modify
   - Files should have restricted permissions (0600)
   - Application process needs read access

3. **Kubernetes permissions** - Bootstrap requires elevated privileges
   - In production, pre-create namespace/RBAC via CI/CD
   - Run Nightcrier with limited ServiceAccount that only manages Jobs/ConfigMaps
   - Bootstrap will skip resource creation if running with limited permissions

4. **Kubeconfig in Secrets** - Sensitive credential storage
   - Kubernetes Secrets are base64-encoded, not encrypted by default
   - Consider enabling encryption-at-rest in production clusters
   - Use RBAC to limit Secret access

## Rollback Plan

If bootstrap causes issues:
1. Revert to previous version
2. Manually apply resources via kubectl (as before)
3. Application will skip bootstrap (resources exist)

No data loss risk - bootstrap only creates resources, never modifies existing data.
