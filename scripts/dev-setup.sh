#!/usr/bin/env bash
#
# dev-setup.sh - Bootstrap local development environment with kind
#
# This script:
# 1. Checks prerequisites (kind, kubectl, docker)
# 2. Creates kind cluster with config
# 3. Loads nc-agent-runner image into kind
# 4. Applies namespace and RBAC manifests
# 5. Prompts for API keys and creates secrets
# 6. Creates sample kubeconfig secret for self-triage
# 7. Verifies everything is ready

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
DEPLOY_DIR="${REPO_ROOT}/deploy/dev"
CLUSTER_NAME="nightcrier-dev"
NAMESPACE="nightcrier"
IMAGE_NAME="nc-agent-runner:latest"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

#######################################
# Print colored message
#######################################
info() {
    echo -e "${GREEN}[INFO]${NC} $*"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $*"
}

error() {
    echo -e "${RED}[ERROR]${NC} $*"
}

#######################################
# Check if command exists
#######################################
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

#######################################
# Check prerequisites
#######################################
check_prerequisites() {
    info "Checking prerequisites..."

    local missing=0

    if ! command_exists kind; then
        error "kind not found. Install: https://kind.sigs.k8s.io/docs/user/quick-start/#installation"
        missing=1
    else
        info "✓ kind $(kind version | cut -d' ' -f2)"
    fi

    if ! command_exists kubectl; then
        error "kubectl not found. Install: https://kubernetes.io/docs/tasks/tools/"
        missing=1
    else
        info "✓ kubectl $(kubectl version --client --short 2>/dev/null | cut -d' ' -f3)"
    fi

    if ! command_exists docker; then
        error "docker not found. Install: https://docs.docker.com/get-docker/"
        missing=1
    else
        info "✓ docker $(docker --version | cut -d' ' -f3 | tr -d ',')"
    fi

    if [[ $missing -eq 1 ]]; then
        error "Missing required tools. Please install them and try again."
        exit 1
    fi

    info "All prerequisites found"
}

#######################################
# Create kind cluster
#######################################
create_cluster() {
    info "Checking if cluster '${CLUSTER_NAME}' exists..."

    if kind get clusters | grep -q "^${CLUSTER_NAME}$"; then
        warn "Cluster '${CLUSTER_NAME}' already exists"
        read -rp "Delete and recreate? (y/N): " response
        if [[ "$response" =~ ^[Yy]$ ]]; then
            info "Deleting existing cluster..."
            kind delete cluster --name "${CLUSTER_NAME}"
        else
            info "Using existing cluster"
            return 0
        fi
    fi

    info "Creating kind cluster '${CLUSTER_NAME}'..."
    kind create cluster --config "${DEPLOY_DIR}/kind-config.yaml" --name "${CLUSTER_NAME}"

    info "Waiting for cluster to be ready..."
    kubectl wait --for=condition=Ready nodes --all --timeout=60s

    info "✓ Cluster created successfully"
}

#######################################
# Load image into kind
#######################################
load_image() {
    info "Checking if image '${IMAGE_NAME}' exists locally..."

    if ! docker image inspect "${IMAGE_NAME}" >/dev/null 2>&1; then
        warn "Image '${IMAGE_NAME}' not found locally"
        info "Building image..."
        (cd "${REPO_ROOT}/nc-agent-runner" && make build)
    fi

    info "Loading image into kind cluster..."
    kind load docker-image "${IMAGE_NAME}" --name "${CLUSTER_NAME}"

    info "✓ Image loaded successfully"
}

#######################################
# Apply K8s manifests
#######################################
apply_manifests() {
    info "Applying K8s manifests..."

    # Apply namespace
    kubectl apply -f "${DEPLOY_DIR}/namespace.yaml"

    # Apply RBAC
    kubectl apply -f "${DEPLOY_DIR}/rbac.yaml"

    info "✓ Manifests applied successfully"
}

#######################################
# Create secrets
#######################################
create_secrets() {
    info "Setting up secrets..."

    # Check if secrets already exist
    if kubectl get secret agent-api-keys -n "${NAMESPACE}" >/dev/null 2>&1; then
        warn "Secret 'agent-api-keys' already exists"
        read -rp "Recreate? (y/N): " response
        if [[ "$response" =~ ^[Yy]$ ]]; then
            kubectl delete secret agent-api-keys -n "${NAMESPACE}"
        else
            info "Using existing secrets"
            return 0
        fi
    fi

    echo ""
    info "Please provide API keys for AI agents (or press Enter to skip):"
    echo ""

    # Prompt for API keys
    read -rp "Anthropic API key (ANTHROPIC_API_KEY): " anthropic_key
    read -rp "OpenAI API key (OPENAI_API_KEY): " openai_key
    read -rp "Gemini API key (GEMINI_API_KEY): " gemini_key

    # Create secret
    kubectl create secret generic agent-api-keys \
        --from-literal=ANTHROPIC_API_KEY="${anthropic_key:-changeme}" \
        --from-literal=OPENAI_API_KEY="${openai_key:-changeme}" \
        --from-literal=GEMINI_API_KEY="${gemini_key:-changeme}" \
        -n "${NAMESPACE}"

    info "✓ API keys secret created"
}

#######################################
# Create kubeconfig secret for self-triage
#######################################
create_kubeconfig_secret() {
    info "Creating kubeconfig secret for self-triage..."

    # Check if secret already exists
    if kubectl get secret "kubeconfig-${CLUSTER_NAME}" -n "${NAMESPACE}" >/dev/null 2>&1; then
        warn "Secret 'kubeconfig-${CLUSTER_NAME}' already exists"
        return 0
    fi

    # For self-triage, agents need read-only access to the cluster
    # Create a ServiceAccount with limited permissions
    kubectl create serviceaccount agent-reader -n "${NAMESPACE}" --dry-run=client -o yaml | kubectl apply -f -

    # Create Role with read-only permissions
    cat <<EOF | kubectl apply -f -
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: agent-reader
rules:
  - apiGroups: [""]
    resources: ["pods", "services", "endpoints", "namespaces", "events", "persistentvolumeclaims"]
    verbs: ["get", "list"]
  - apiGroups: [""]
    resources: ["pods/log"]
    verbs: ["get"]
  - apiGroups: ["apps"]
    resources: ["deployments", "replicasets", "statefulsets", "daemonsets"]
    verbs: ["get", "list"]
  - apiGroups: ["batch"]
    resources: ["jobs", "cronjobs"]
    verbs: ["get", "list"]
EOF

    # Create ClusterRoleBinding
    kubectl create clusterrolebinding agent-reader \
        --clusterrole=agent-reader \
        --serviceaccount="${NAMESPACE}:agent-reader" \
        --dry-run=client -o yaml | kubectl apply -f -

    # Get service account token
    local token
    token=$(kubectl create token agent-reader -n "${NAMESPACE}" --duration=87600h)  # 10 years

    # Get cluster CA and server
    local ca_data
    local server
    ca_data=$(kubectl config view --raw -o jsonpath='{.clusters[0].cluster.certificate-authority-data}')
    server="https://kubernetes.default.svc"

    # Create kubeconfig
    local kubeconfig
    kubeconfig=$(cat <<EOF
apiVersion: v1
kind: Config
clusters:
  - cluster:
      certificate-authority-data: ${ca_data}
      server: ${server}
    name: ${CLUSTER_NAME}
contexts:
  - context:
      cluster: ${CLUSTER_NAME}
      user: agent-reader
    name: ${CLUSTER_NAME}
current-context: ${CLUSTER_NAME}
users:
  - name: agent-reader
    user:
      token: ${token}
EOF
)

    # Create secret
    kubectl create secret generic "kubeconfig-${CLUSTER_NAME}" \
        --from-literal=config="${kubeconfig}" \
        -n "${NAMESPACE}"

    info "✓ Kubeconfig secret created for self-triage"
}

#######################################
# Verify setup
#######################################
verify_setup() {
    info "Verifying setup..."

    local errors=0

    # Check cluster
    if ! kubectl cluster-info >/dev/null 2>&1; then
        error "Cluster not accessible"
        errors=$((errors + 1))
    else
        info "✓ Cluster accessible"
    fi

    # Check namespace
    if ! kubectl get namespace "${NAMESPACE}" >/dev/null 2>&1; then
        error "Namespace '${NAMESPACE}' not found"
        errors=$((errors + 1))
    else
        info "✓ Namespace exists"
    fi

    # Check RBAC
    if ! kubectl get serviceaccount nightcrier-executor -n "${NAMESPACE}" >/dev/null 2>&1; then
        error "ServiceAccount 'nightcrier-executor' not found"
        errors=$((errors + 1))
    else
        info "✓ RBAC configured"
    fi

    # Check secrets
    if ! kubectl get secret agent-api-keys -n "${NAMESPACE}" >/dev/null 2>&1; then
        warn "Secret 'agent-api-keys' not found"
    else
        info "✓ API keys secret exists"
    fi

    if ! kubectl get secret "kubeconfig-${CLUSTER_NAME}" -n "${NAMESPACE}" >/dev/null 2>&1; then
        warn "Secret 'kubeconfig-${CLUSTER_NAME}' not found"
    else
        info "✓ Kubeconfig secret exists"
    fi

    # Check image in kind
    if ! docker exec "${CLUSTER_NAME}-control-plane" crictl images | grep -q "nc-agent-runner"; then
        warn "Image 'nc-agent-runner' not found in kind cluster"
    else
        info "✓ Image loaded in kind"
    fi

    if [[ $errors -gt 0 ]]; then
        error "Setup verification failed with $errors errors"
        exit 1
    fi

    info "✓ All checks passed"
}

#######################################
# Print summary
#######################################
print_summary() {
    echo ""
    info "=========================================="
    info "Local Development Environment Ready!"
    info "=========================================="
    echo ""
    echo "Cluster: ${CLUSTER_NAME}"
    echo "Namespace: ${NAMESPACE}"
    echo "Image: ${IMAGE_NAME}"
    echo ""
    echo "Next steps:"
    echo "  1. Update API keys: kubectl edit secret agent-api-keys -n ${NAMESPACE}"
    echo "  2. Run Nightcrier: ./nightcrier server"
    echo "  3. Create test incident: see deploy/dev/test-incident.json"
    echo ""
    echo "Useful commands:"
    echo "  kubectl get all -n ${NAMESPACE}"
    echo "  kubectl logs -n ${NAMESPACE} -l app=nc-agent-runner -f"
    echo "  kind delete cluster --name ${CLUSTER_NAME}"
    echo ""
}

#######################################
# Main
#######################################
main() {
    info "Starting local development environment setup..."
    echo ""

    check_prerequisites
    echo ""

    create_cluster
    echo ""

    load_image
    echo ""

    apply_manifests
    echo ""

    create_secrets
    echo ""

    create_kubeconfig_secret
    echo ""

    verify_setup
    echo ""

    print_summary
}

main "$@"
