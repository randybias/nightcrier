# Proposal: Automate Kubernetes Bootstrap

## Problem

Nightcrier currently requires manual Kubernetes setup before it can run:
1. **Namespace creation** - `nightcrier` namespace must be manually created
2. **RBAC setup** - ServiceAccount, Role, and RoleBinding must be manually applied
3. **API keys secret** - `ai-api-keys` Secret must be manually created with LLM API keys
4. **Kubeconfig secrets** - Per-cluster `kubeconfig-{cluster-name}` Secrets must be manually created

All of this setup logic lives in `scripts/dev-setup.sh`, which is **only designed for local kind clusters**. This means:
- Remote cluster deployments are undocumented and broken
- Production deployments require manual kubectl commands
- The `triage.kubeconfig` path in configuration is validated but never used
- The application cannot self-provision its runtime dependencies

## Solution

Move all Kubernetes resource provisioning into the core application startup sequence. When Nightcrier starts, it should:

1. **Connect to the target Kubernetes cluster** using:
   - `KUBECONFIG` environment variable, OR
   - `kubeconfig_path` from configuration, OR
   - Default kubeconfig location (`~/.kube/config`)

2. **Bootstrap required resources** (if they don't exist):
   - Create `k8s_namespace` (default: `nightcrier`)
   - Apply RBAC: ServiceAccount, Role, RoleBinding
   - Create `ai-api-keys` Secret from API keys in config/environment
   - For each cluster in `clusters` array:
     - Read `triage.kubeconfig` file from disk
     - Create `kubeconfig-{cluster-name}` Secret from file contents

3. **Validate prerequisites**:
   - Check that kubeconfig files exist before creating Secrets
   - Verify Kubernetes permissions to create resources
   - Fail fast with clear error messages if setup cannot complete

4. **Support both local and remote clusters**:
   - Local kind: Same behavior as today
   - Remote cluster: Connect via kubeconfig, bootstrap there
   - In-cluster: Use ServiceAccount credentials when running as Pod

## Scope

### In Scope
- Bootstrap namespace, RBAC, and Secrets on startup
- Read `triage.kubeconfig` files and create corresponding Secrets
- Validate required files exist before attempting setup
- Clear error messages for permission failures
- Idempotent operations (skip if resources already exist)

### Out of Scope
- Creating ServiceAccounts for triage agents (remains manual/scripted)
- Automatic RBAC updates on config changes
- Secret rotation or credential management
- Kubernetes operator patterns (watch/reconcile)
- Migration path for existing deployments (manual cleanup acceptable)

## Benefits

1. **Self-contained deployment** - Application sets up its own prerequisites
2. **Remote cluster support** - Works with any Kubernetes cluster, not just local kind
3. **Configuration consistency** - `triage.kubeconfig` paths are actually used
4. **Production ready** - No manual kubectl commands required
5. **Fail fast** - Clear errors at startup if setup cannot complete

## Risks

1. **Increased startup time** - Kubernetes API calls add latency
   - Mitigation: Skip resources that already exist (idempotent checks)

2. **Permission failures** - Application may lack privileges to create resources
   - Mitigation: Clear error messages with remediation steps

3. **Kubeconfig file issues** - Invalid or missing files
   - Mitigation: Validate files exist and are readable before creating Secrets

## Alternatives Considered

### Manual setup documentation
- Document exact kubectl commands for production deployment
- Rejected: Error-prone, doesn't leverage configuration file

### Helm chart or operator
- Package everything as Helm chart with hooks
- Rejected: Too complex for current needs, requires Helm knowledge

### Init container pattern
- Run bootstrap in init container when deployed as Pod
- Rejected: Doesn't help with external deployment, still need core logic

## Open Questions

None - implementation approach is clear.
