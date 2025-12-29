#!/usr/bin/env bash
#
# Update agent CLI versions in nc-agent-runner Dockerfile
#
# This script helps check for and update the versions of AI CLI tools
# in the nc-agent-runner Dockerfile. It can query for latest versions
# and optionally update the Dockerfile.
#
# Usage:
#   ./scripts/update-agent-versions.sh [--check|--update]
#
# Options:
#   --check   Check current and latest versions (default)
#   --update  Update Dockerfile with latest versions (interactive)
#

set -euo pipefail

# Determine script and project directories
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
DOCKERFILE="${PROJECT_ROOT}/nc-agent-runner/Dockerfile"

# Parse command line arguments
MODE="${1:-check}"

#######################################
# Display script header
# Arguments:
#   None
#######################################
show_header() {
    echo "=========================================="
    echo "nc-agent-runner CLI Version Manager"
    echo "=========================================="
    echo "Mode: ${MODE}"
    echo "Dockerfile: ${DOCKERFILE}"
    echo ""
}

#######################################
# Check if a command exists
# Arguments:
#   $1 - Command name
# Returns:
#   0 if command exists, 1 otherwise
#######################################
command_exists() {
    command -v "$1" &> /dev/null
}

#######################################
# Get the latest version of kubectl
# Outputs:
#   Latest kubectl version string (e.g., "1.31.0")
#######################################
get_kubectl_latest() {
    curl -sL https://dl.k8s.io/release/stable.txt | sed 's/^v//'
}

#######################################
# Get the current kubectl version from Dockerfile
# Outputs:
#   Current kubectl version string from Dockerfile
#######################################
get_kubectl_current() {
    grep -o 'v1\.[0-9]*' "${DOCKERFILE}" | head -1 | sed 's/^v//'
}

#######################################
# Get the latest version of Helm
# Outputs:
#   Latest Helm version string (e.g., "3.14.0")
#######################################
get_helm_latest() {
    curl -sL https://api.github.com/repos/helm/helm/releases/latest | \
        jq -r '.tag_name' | sed 's/^v//'
}

#######################################
# Get the latest version of Goose
# Outputs:
#   Latest Goose version string (e.g., "1.18.0")
#######################################
get_goose_latest() {
    curl -sL https://api.github.com/repos/block/goose/releases/latest | \
        jq -r '.tag_name' | sed 's/^v//'
}

#######################################
# Get the current Goose version from Dockerfile
# Outputs:
#   Current Goose version string from Dockerfile
#######################################
get_goose_current() {
    grep -o 'goose/releases/download/v[0-9.]*' "${DOCKERFILE}" | \
        head -1 | sed 's|.*v||'
}

#######################################
# Check and display version information
# Globals:
#   DOCKERFILE
# Arguments:
#   None
#######################################
check_versions() {
    echo "Checking versions..."
    echo ""

    # Check prerequisites
    if ! command_exists curl; then
        echo "Error: curl is required but not found"
        exit 1
    fi

    if ! command_exists jq; then
        echo "Error: jq is required but not found"
        exit 1
    fi

    # kubectl version
    echo "kubectl:"
    local kubectl_current
    kubectl_current=$(get_kubectl_current)
    echo "  Current: ${kubectl_current}"

    local kubectl_latest
    kubectl_latest=$(get_kubectl_latest)
    echo "  Latest:  ${kubectl_latest}"

    if [[ "${kubectl_current}" != "${kubectl_latest}" ]]; then
        echo "  Status:  UPDATE AVAILABLE"
    else
        echo "  Status:  Up to date"
    fi
    echo ""

    # Helm version
    echo "Helm:"
    echo "  Current: (using install script - always latest)"
    local helm_latest
    helm_latest=$(get_helm_latest)
    echo "  Latest:  ${helm_latest}"
    echo "  Status:  Install script pulls latest version"
    echo ""

    # Goose version
    echo "Goose:"
    local goose_current
    goose_current=$(get_goose_current)
    echo "  Current: ${goose_current}"

    local goose_latest
    goose_latest=$(get_goose_latest)
    echo "  Latest:  ${goose_latest}"

    if [[ "${goose_current}" != "${goose_latest}" ]]; then
        echo "  Status:  UPDATE AVAILABLE"
    else
        echo "  Status:  Up to date"
    fi
    echo ""

    # AI CLI tools (npm packages)
    echo "AI CLI Tools (npm packages):"
    echo "  Claude Code CLI:  @anthropic-ai/claude-code@latest"
    echo "  OpenAI Codex CLI: @openai/codex@latest"
    echo "  Google Gemini CLI: @google/gemini-cli@latest"
    echo "  Status: Using @latest tag (always latest on build)"
    echo ""

    echo "Note: AI CLI tools are installed with @latest tag,"
    echo "      so they will always pull the latest version at build time."
    echo ""
}

#######################################
# Update kubectl version in Dockerfile
# Arguments:
#   $1 - New version string
#######################################
update_kubectl_version() {
    local new_version="$1"
    local major_minor
    major_minor=$(echo "${new_version}" | cut -d. -f1-2)

    echo "Updating kubectl to v${major_minor}..."

    # Update the version in the kubernetes repository URL
    sed -i.bak "s|v1\.[0-9]*/deb|v${major_minor}/deb|g" "${DOCKERFILE}"

    echo "  Updated kubernetes repository to v${major_minor}"
}

#######################################
# Update Goose version in Dockerfile
# Arguments:
#   $1 - New version string
#######################################
update_goose_version() {
    local new_version="$1"

    echo "Updating Goose to v${new_version}..."

    # Update the version in the download URL
    sed -i.bak "s|goose/releases/download/v[0-9.]*|goose/releases/download/v${new_version}|g" "${DOCKERFILE}"

    echo "  Updated Goose download URL to v${new_version}"
}

#######################################
# Interactive update mode
# Globals:
#   DOCKERFILE
# Arguments:
#   None
#######################################
update_versions() {
    echo "Checking for updates..."
    echo ""

    # Get current and latest versions
    local kubectl_current kubectl_latest
    kubectl_current=$(get_kubectl_current)
    kubectl_latest=$(get_kubectl_latest)

    local goose_current goose_latest
    goose_current=$(get_goose_current)
    goose_latest=$(get_goose_latest)

    local updates_needed=false

    # Check kubectl
    if [[ "${kubectl_current}" != "${kubectl_latest}" ]]; then
        echo "kubectl update available: ${kubectl_current} -> ${kubectl_latest}"
        read -rp "Update kubectl? (y/n): " response
        if [[ "${response}" =~ ^[Yy]$ ]]; then
            update_kubectl_version "${kubectl_latest}"
            updates_needed=true
        fi
        echo ""
    fi

    # Check Goose
    if [[ "${goose_current}" != "${goose_latest}" ]]; then
        echo "Goose update available: ${goose_current} -> ${goose_latest}"
        read -rp "Update Goose? (y/n): " response
        if [[ "${response}" =~ ^[Yy]$ ]]; then
            update_goose_version "${goose_latest}"
            updates_needed=true
        fi
        echo ""
    fi

    if [[ "${updates_needed}" == "true" ]]; then
        # Clean up backup file
        rm -f "${DOCKERFILE}.bak"

        echo "=========================================="
        echo "Updates applied!"
        echo "=========================================="
        echo ""
        echo "Next steps:"
        echo "1. Review the changes:"
        echo "   git diff ${DOCKERFILE}"
        echo ""
        echo "2. Rebuild the image:"
        echo "   ./scripts/build-agent-image.sh"
        echo ""
    else
        echo "No updates needed or applied."
        echo ""
    fi
}

#######################################
# Main execution
#######################################
main() {
    show_header

    # Verify Dockerfile exists
    if [[ ! -f "${DOCKERFILE}" ]]; then
        echo "Error: Dockerfile not found at ${DOCKERFILE}"
        exit 1
    fi

    case "${MODE}" in
        --check|check)
            check_versions
            ;;
        --update|update)
            update_versions
            ;;
        *)
            echo "Error: Unknown mode '${MODE}'"
            echo "Usage: $0 [--check|--update]"
            exit 1
            ;;
    esac
}

# Execute main function
main
