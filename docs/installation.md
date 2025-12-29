# Installation

## Prerequisites

- Go 1.23 or later
- Kubernetes cluster (for agent execution via Jobs)
- kubectl and kind (for local development)
- Object storage: Azure Blob Storage, AWS S3, or S3-compatible (MinIO, RustFS)
- Slack webhook (optional, for notifications)
- At least one LLM API key (Anthropic, OpenAI, or Google)

## Build from Source

```bash
# Clone the repository
git clone https://github.com/rbias/nightcrier.git
cd nightcrier

# Build the Nightcrier binary
make build

# Build the agent container image
cd nc-agent-runner
make build

# For local development with kind, load the image
make load-kind
```

## Kubeconfig Setup

Triage agents require read-only cluster access to investigate incidents. Follow these steps to create appropriate credentials.

### 1. Create Read-Only ServiceAccount

Create a dedicated ServiceAccount with minimal permissions:

```yaml
# kubernetes-triage-readonly-sa.yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: kubernetes-triage-readonly
  namespace: kube-system
---
apiVersion: v1
kind: Secret
metadata:
  name: kubernetes-triage-readonly-token
  namespace: kube-system
  annotations:
    kubernetes.io/service-account.name: kubernetes-triage-readonly
type: kubernetes.io/service-account-token
```

Apply:
```bash
kubectl apply -f kubernetes-triage-readonly-sa.yaml
```

### 2. Grant RBAC Permissions

**Minimum Permissions (Required)**:

Bind the built-in `view` ClusterRole for basic read access:

```yaml
# kubernetes-triage-readonly-rbac.yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: kubernetes-triage-readonly-view
subjects:
  - kind: ServiceAccount
    name: kubernetes-triage-readonly
    namespace: kube-system
roleRef:
  kind: ClusterRole
  name: view
  apiGroup: rbac.authorization.k8s.io
```

This provides access to:
- Pods (get, list, watch)
- Pod logs (pods/log subresource)
- Events (get, list, watch)
- Deployments, ReplicaSets, StatefulSets
- Services, Endpoints
- ConfigMaps (but NOT secrets)

**Optional: Node Access**:

For cluster-wide visibility (node resource usage, taints, etc.):

```yaml
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: kubernetes-triage-nodes-readonly
rules:
  - apiGroups: [""]
    resources: ["nodes"]
    verbs: ["get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: kubernetes-triage-nodes-readonly
subjects:
  - kind: ServiceAccount
    name: kubernetes-triage-readonly
    namespace: kube-system
roleRef:
  kind: ClusterRole
  name: kubernetes-triage-nodes-readonly
  apiGroup: rbac.authorization.k8s.io
```

**Optional: Helm Debugging Permissions**:

WARNING: This allows reading secrets, which may contain sensitive data.

```yaml
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: kubernetes-triage-helm-readonly
rules:
  - apiGroups: [""]
    resources: ["secrets", "configmaps"]
    verbs: ["get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: kubernetes-triage-helm-readonly
subjects:
  - kind: ServiceAccount
    name: kubernetes-triage-readonly
    namespace: kube-system
roleRef:
  kind: ClusterRole
  name: kubernetes-triage-helm-readonly
  apiGroup: rbac.authorization.k8s.io
```

If you grant secrets access, set `triage.allow_secrets_access: true` in config.

Apply RBAC:
```bash
kubectl apply -f kubernetes-triage-readonly-rbac.yaml
```

### 3. Extract Kubeconfig

Extract the ServiceAccount token and generate a kubeconfig:

```bash
#!/bin/bash
# extract-triage-kubeconfig.sh

CLUSTER_NAME="prod-us-east-1"
SA_NAME="kubernetes-triage-readonly"
SA_NAMESPACE="kube-system"
OUTPUT_FILE="./kubeconfigs/${CLUSTER_NAME}-readonly.yaml"

# Get cluster info
CLUSTER_SERVER=$(kubectl config view --minify -o jsonpath='{.clusters[0].cluster.server}')
CLUSTER_CA=$(kubectl config view --minify --raw -o jsonpath='{.clusters[0].cluster.certificate-authority-data}')

# Get ServiceAccount token
SA_TOKEN=$(kubectl get secret -n ${SA_NAMESPACE} kubernetes-triage-readonly-token -o jsonpath='{.data.token}' | base64 -d)

# Create kubeconfig
cat > ${OUTPUT_FILE} <<EOF
apiVersion: v1
kind: Config
clusters:
- cluster:
    certificate-authority-data: ${CLUSTER_CA}
    server: ${CLUSTER_SERVER}
  name: ${CLUSTER_NAME}
contexts:
- context:
    cluster: ${CLUSTER_NAME}
    user: ${SA_NAME}
  name: ${CLUSTER_NAME}
current-context: ${CLUSTER_NAME}
users:
- name: ${SA_NAME}
  user:
    token: ${SA_TOKEN}
EOF

echo "Kubeconfig written to ${OUTPUT_FILE}"

# Test the kubeconfig
kubectl --kubeconfig=${OUTPUT_FILE} auth can-i --list
```

Run:
```bash
chmod +x extract-triage-kubeconfig.sh
./extract-triage-kubeconfig.sh
```

### 4. Verify Permissions

Test the generated kubeconfig:

```bash
KUBECONFIG_FILE="./kubeconfigs/prod-us-east-1-readonly.yaml"

# Test basic access
kubectl --kubeconfig=${KUBECONFIG_FILE} get pods --all-namespaces

# Verify can-i permissions
kubectl --kubeconfig=${KUBECONFIG_FILE} auth can-i get pods
kubectl --kubeconfig=${KUBECONFIG_FILE} auth can-i get pods/log
kubectl --kubeconfig=${KUBECONFIG_FILE} auth can-i get events
kubectl --kubeconfig=${KUBECONFIG_FILE} auth can-i get nodes

# Verify cannot mutate
kubectl --kubeconfig=${KUBECONFIG_FILE} auth can-i delete pods    # Should return "no"
kubectl --kubeconfig=${KUBECONFIG_FILE} auth can-i create pods    # Should return "no"
```

### 5. Configure Nightcrier

Update `configs/config.yaml` to reference the kubeconfig:

```yaml
clusters:
  - name: prod-us-east-1
    mcp:
      endpoint: http://kubernetes-mcp-server:8080/mcp
    triage:
      enabled: true
      kubeconfig: ./kubeconfigs/prod-us-east-1-readonly.yaml
      allow_secrets_access: false  # or true if Helm debugging needed
```

### Startup Permission Validation

When Nightcrier starts, it automatically validates cluster permissions:

```
level=INFO msg="initializing connection manager - validating permissions"
level=INFO msg="validating cluster permissions" cluster=prod-us-east-1 kubeconfig=./kubeconfigs/prod-us-east-1-readonly.yaml
level=INFO msg="cluster permissions validated successfully" cluster=prod-us-east-1 minimum_met=true helm_access=false
```

If permissions are insufficient:
```
level=WARN msg="cluster has permission warnings" cluster=prod-us-east-1 warnings="cannot get nodes (cluster-wide visibility limited)"
```

Permission validation results are written to `incident_cluster_permissions.json` in each incident workspace, allowing the AI agent to understand what actions are available.
