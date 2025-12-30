#!/usr/bin/env bash
#
# get-mcp-endpoint.sh - Discover MCP server endpoint from a Kubernetes cluster
#
# Usage: ./get-mcp-endpoint.sh <kubeconfig-file>

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

error() {
    echo -e "${RED}[ERROR]${NC} $*" >&2
}

info() {
    echo -e "${GREEN}[INFO]${NC} $*" >&2
}

# Check arguments
if [[ $# -ne 1 ]]; then
    error "Usage: $0 <kubeconfig-file>"
    exit 1
fi

KUBECONFIG_FILE="$1"

# Verify kubeconfig exists
if [[ ! -f "$KUBECONFIG_FILE" ]]; then
    error "Kubeconfig file not found: $KUBECONFIG_FILE"
    exit 1
fi

# Set kubectl with kubeconfig
KUBECTL="kubectl --kubeconfig=$KUBECONFIG_FILE"

# Find MCP server pod in mcp-system namespace
info "Looking for kubernetes-mcp-server pod in mcp-system namespace..."
POD_NAME=$($KUBECTL get pods -n mcp-system -l app=kubernetes-mcp-server -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)

if [[ -z "$POD_NAME" ]]; then
    # Try finding by name pattern if label doesn't work
    POD_NAME=$($KUBECTL get pods -n mcp-system -o jsonpath='{.items[?(@.metadata.name contains "kubernetes-mcp-server")].metadata.name}' 2>/dev/null | awk '{print $1}' || true)
fi

if [[ -z "$POD_NAME" ]]; then
    error "No kubernetes-mcp-server pod found in mcp-system namespace"
    exit 1
fi

info "Found pod: $POD_NAME"

# Find the service (likely NodePort)
info "Looking for MCP service..."
SERVICE_NAME=$($KUBECTL get svc -n mcp-system -l app=kubernetes-mcp-server -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)

if [[ -z "$SERVICE_NAME" ]]; then
    # Try finding service by name pattern
    SERVICE_NAME=$($KUBECTL get svc -n mcp-system -o jsonpath='{.items[?(@.metadata.name contains "mcp")].metadata.name}' 2>/dev/null | awk '{print $1}' || true)
fi

if [[ -z "$SERVICE_NAME" ]]; then
    error "No MCP service found in mcp-system namespace"
    exit 1
fi

info "Found service: $SERVICE_NAME"

# Get service type and port
SERVICE_TYPE=$($KUBECTL get svc "$SERVICE_NAME" -n mcp-system -o jsonpath='{.spec.type}')
info "Service type: $SERVICE_TYPE"

# Get NodePort
if [[ "$SERVICE_TYPE" == "NodePort" ]]; then
    NODE_PORT=$($KUBECTL get svc "$SERVICE_NAME" -n mcp-system -o jsonpath='{.spec.ports[0].nodePort}')
elif [[ "$SERVICE_TYPE" == "LoadBalancer" ]]; then
    NODE_PORT=$($KUBECTL get svc "$SERVICE_NAME" -n mcp-system -o jsonpath='{.spec.ports[0].port}')
else
    error "Unsupported service type: $SERVICE_TYPE (expected NodePort or LoadBalancer)"
    exit 1
fi

info "Port: $NODE_PORT"

# Get external IP (cluster node IP or LoadBalancer IP)
EXTERNAL_IP=""
if [[ "$SERVICE_TYPE" == "LoadBalancer" ]]; then
    # Try LoadBalancer external IP first
    EXTERNAL_IP=$($KUBECTL get svc "$SERVICE_NAME" -n mcp-system -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || true)
    if [[ -z "$EXTERNAL_IP" ]]; then
        # Try hostname (for ELB/ALB)
        EXTERNAL_IP=$($KUBECTL get svc "$SERVICE_NAME" -n mcp-system -o jsonpath='{.status.loadBalancer.ingress[0].hostname}' 2>/dev/null || true)
    fi
elif [[ "$SERVICE_TYPE" == "NodePort" ]]; then
    # For NodePort, use the Kubernetes API server IP from kubeconfig
    # This is needed when accessing via VPN where cluster IPs are not routed
    API_SERVER=$($KUBECTL config view --minify -o jsonpath='{.clusters[0].cluster.server}')
    # Extract IP/hostname from URL (remove https:// and :port)
    EXTERNAL_IP=$(echo "$API_SERVER" | sed -E 's|https?://([^:/]+).*|\1|')
    info "Using Kubernetes API server IP from kubeconfig"
fi

if [[ -z "$EXTERNAL_IP" ]]; then
    error "Could not determine cluster IP address"
    exit 1
fi

info "Cluster IP: $EXTERNAL_IP"

# Output the MCP endpoint URL
MCP_ENDPOINT="http://${EXTERNAL_IP}:${NODE_PORT}/mcp"
echo "$MCP_ENDPOINT"
