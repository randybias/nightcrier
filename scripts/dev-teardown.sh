#!/usr/bin/env bash
#
# dev-teardown.sh - Clean up local development environment
#
# This script:
# 1. Deletes the kind cluster
# 2. Optionally removes local secrets file

set -euo pipefail

CLUSTER_NAME="nightcrier-dev"
DEPLOY_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../deploy/dev" && pwd)"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

info() {
    echo -e "${GREEN}[INFO]${NC} $*"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $*"
}

#######################################
# Delete kind cluster
#######################################
delete_cluster() {
    info "Checking for cluster '${CLUSTER_NAME}'..."

    if ! kind get clusters | grep -q "^${CLUSTER_NAME}$"; then
        warn "Cluster '${CLUSTER_NAME}' not found"
        return 0
    fi

    info "Deleting kind cluster '${CLUSTER_NAME}'..."
    kind delete cluster --name "${CLUSTER_NAME}"

    info "✓ Cluster deleted"
}

#######################################
# Clean up local secrets
#######################################
cleanup_secrets() {
    local secrets_file="${DEPLOY_DIR}/secrets.yaml"
    local kubeconfig_file="${DEPLOY_DIR}/kubeconfig-secret.yaml"

    if [[ -f "${secrets_file}" ]] || [[ -f "${kubeconfig_file}" ]]; then
        echo ""
        warn "Found local secrets files:"
        [[ -f "${secrets_file}" ]] && echo "  - ${secrets_file}"
        [[ -f "${kubeconfig_file}" ]] && echo "  - ${kubeconfig_file}"
        echo ""

        read -rp "Delete local secrets files? (y/N): " response
        if [[ "$response" =~ ^[Yy]$ ]]; then
            [[ -f "${secrets_file}" ]] && rm "${secrets_file}" && info "✓ Deleted ${secrets_file}"
            [[ -f "${kubeconfig_file}" ]] && rm "${kubeconfig_file}" && info "✓ Deleted ${kubeconfig_file}"
        else
            info "Keeping local secrets files"
        fi
    fi
}

#######################################
# Main
#######################################
main() {
    info "Cleaning up local development environment..."
    echo ""

    delete_cluster
    echo ""

    cleanup_secrets
    echo ""

    info "=========================================="
    info "Cleanup complete!"
    info "=========================================="
}

main "$@"
