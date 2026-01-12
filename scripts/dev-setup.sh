#!/usr/bin/env bash
#
# dev-setup.sh - Setup local kind cluster for development
#
# This script:
# 1. Checks prerequisites (kind, kubectl, docker)
# 2. Creates kind cluster with config
# 3. Loads nc-agent-runner image into kind
#
# Note: Kubernetes resources (namespace, RBAC, secrets) are automatically
# bootstrapped by nightcrier on startup. This script only handles the kind
# cluster infrastructure.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
DEPLOY_DIR="${REPO_ROOT}/deploy/dev"
CLUSTER_NAME="nightcrier-dev"
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
# Export kubeconfig for execution cluster
#######################################
export_kubeconfig() {
    local kubeconfig_dir="${REPO_ROOT}/kubeconfigs"
    local kubeconfig_file="${kubeconfig_dir}/exec-cluster.yaml"

    info "Exporting kubeconfig for execution cluster..."

    # Create kubeconfigs directory if it doesn't exist
    if [[ ! -d "${kubeconfig_dir}" ]]; then
        info "Creating kubeconfigs directory..."
        mkdir -p "${kubeconfig_dir}"
    fi

    # Export kubeconfig
    kind export kubeconfig --name "${CLUSTER_NAME}" --kubeconfig "${kubeconfig_file}"

    info "✓ Kubeconfig exported to ${kubeconfig_file}"
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

    # Check image in kind
    if ! docker exec "${CLUSTER_NAME}-control-plane" crictl images | grep -q "nc-agent-runner"; then
        warn "Image 'nc-agent-runner' not found in kind cluster"
        errors=$((errors + 1))
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
    info "Kind Cluster Setup Complete!"
    info "=========================================="
    echo ""
    echo "Cluster: ${CLUSTER_NAME}"
    echo "Image: ${IMAGE_NAME}"
    echo "Kubeconfig: ./kubeconfigs/exec-cluster.yaml"
    echo ""
    echo "Next steps:"
    echo "  1. Copy configs/config.example.yaml to configs/config.yaml"
    echo "  2. Configure API keys in config.yaml or environment variables"
    echo "  3. Run Nightcrier: ./bin/nightcrier --config configs/config.yaml"
    echo "     (This will automatically bootstrap namespace, RBAC, and secrets)"
    echo "  4. Create test incident: see deploy/dev/test-incident.json"
    echo ""
    echo "Useful commands:"
    echo "  kubectl get all -n nightcrier"
    echo "  kubectl logs -n nightcrier -l app=nc-agent-runner -f"
    echo "  kind delete cluster --name ${CLUSTER_NAME}"
    echo ""
}

#######################################
# Main
#######################################
main() {
    info "Starting kind cluster setup..."
    echo ""

    check_prerequisites
    echo ""

    create_cluster
    echo ""

    export_kubeconfig
    echo ""

    load_image
    echo ""

    verify_setup
    echo ""

    print_summary
}

main "$@"
