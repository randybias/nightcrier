#!/usr/bin/env bash
#
# Build script for nc-agent-runner container image
#
# This script builds the nc-agent-runner Docker image with proper tagging.
# It must be run from the repository root to ensure proper build context.
#
# Usage:
#   ./scripts/build-agent-image.sh [VERSION]
#
# Examples:
#   ./scripts/build-agent-image.sh           # Build with 'latest' tag
#   ./scripts/build-agent-image.sh 1.0.0     # Build with version tag
#

set -euo pipefail

# Determine script and project directories
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Parse arguments
VERSION="${1:-latest}"
IMAGE_NAME="nc-agent-runner"
DOCKERFILE_PATH="nc-agent-runner/Dockerfile"

#######################################
# Display build information
# Globals:
#   VERSION
#   IMAGE_NAME
# Arguments:
#   None
#######################################
show_build_info() {
    echo "=========================================="
    echo "Building nc-agent-runner container image"
    echo "=========================================="
    echo "Version: ${VERSION}"
    echo "Image: ${IMAGE_NAME}:${VERSION}"
    echo "Build context: ${PROJECT_ROOT}"
    echo "Dockerfile: ${DOCKERFILE_PATH}"
    echo ""
}

#######################################
# Verify build prerequisites
# Globals:
#   PROJECT_ROOT
#   DOCKERFILE_PATH
# Arguments:
#   None
#######################################
verify_prerequisites() {
    # Check Docker is available
    if ! command -v docker &> /dev/null; then
        echo "Error: docker command not found. Please install Docker."
        exit 1
    fi

    # Check we're in the right directory
    if [[ ! -f "${PROJECT_ROOT}/${DOCKERFILE_PATH}" ]]; then
        echo "Error: Dockerfile not found at ${PROJECT_ROOT}/${DOCKERFILE_PATH}"
        echo "Please run this script from the repository root."
        exit 1
    fi

    # Check entrypoint script exists
    if [[ ! -f "${PROJECT_ROOT}/nc-agent-runner/entrypoint.sh" ]]; then
        echo "Error: entrypoint.sh not found at ${PROJECT_ROOT}/nc-agent-runner/entrypoint.sh"
        exit 1
    fi

# TODO: remove as this is no longer a requirement.  Skills are cloned into the image via git
#    # Check k8s-troubleshooter skill exists
#    if [[ ! -d "${PROJECT_ROOT}/internal/skills/agent-home/skills/k8s4agents/skills/k8s-troubleshooter" ]]; then
#        echo "Error: k8s-troubleshooter skill not found"
#        exit 1
#    fi

    echo "Prerequisites verified"
    echo ""
}

#######################################
# Build the Docker image
# Globals:
#   PROJECT_ROOT
#   DOCKERFILE_PATH
#   IMAGE_NAME
#   VERSION
# Arguments:
#   None
#######################################
build_image() {
    echo "Starting Docker build..."
    echo ""

    cd "${PROJECT_ROOT}"

    # Build with both version tag and latest tag
    docker build \
        -f "${DOCKERFILE_PATH}" \
        -t "${IMAGE_NAME}:${VERSION}" \
        -t "${IMAGE_NAME}:latest" \
        .

    echo ""
    echo "=========================================="
    echo "Build complete!"
    echo "=========================================="
    echo "Tagged images:"
    echo "  ${IMAGE_NAME}:${VERSION}"
    echo "  ${IMAGE_NAME}:latest"
    echo ""
}

#######################################
# Display post-build instructions
# Globals:
#   IMAGE_NAME
#   VERSION
# Arguments:
#   None
#######################################
show_post_build_info() {
    echo "Next steps:"
    echo ""
    echo "1. Test the image locally:"
    echo "   docker run --rm ${IMAGE_NAME}:${VERSION}"
    echo ""
    echo "2. Push to container registry:"
    echo "   docker tag ${IMAGE_NAME}:${VERSION} your-registry/${IMAGE_NAME}:${VERSION}"
    echo "   docker push your-registry/${IMAGE_NAME}:${VERSION}"
    echo ""
    echo "3. View image details:"
    echo "   docker images ${IMAGE_NAME}"
    echo "   docker inspect ${IMAGE_NAME}:${VERSION}"
    echo ""
}

#######################################
# Main execution
#######################################
main() {
    show_build_info
    verify_prerequisites
    build_image
    show_post_build_info
}

# Execute main function
main
