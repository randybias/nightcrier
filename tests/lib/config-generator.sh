#!/usr/bin/env bash
# config-generator.sh - Generate nightcrier configs from templates
#
# Usage: config-generator.sh <agent> [--debug]
#   agent: claude, codex, or gemini
#   --debug: Enable agent debug mode
#
# Requires:
#   - ~/dev-secrets/nightcrier-secrets.env with required secrets
#   - Template file at tests/config-templates/test-<agent>.yaml.tmpl
#
# Generates:
#   - configs/config-test-<agent>.yaml

set -euo pipefail

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# Paths
SECRETS_FILE="${HOME}/dev-secrets/nightcrier-secrets.env"
TEMPLATES_DIR="${REPO_ROOT}/tests/config-templates"
CONFIGS_DIR="${REPO_ROOT}/configs"

# Usage information
usage() {
    cat >&2 <<EOF
Usage: $0 <agent> [--debug]

Arguments:
  agent     Agent type: claude, codex, or gemini
  --debug   Enable agent debug mode (optional)

Example:
  $0 claude
  $0 codex --debug

Requires:
  - Secrets file at ${SECRETS_FILE}
  - Template at ${TEMPLATES_DIR}/test-<agent>.yaml.tmpl
EOF
    exit 1
}

# Validate required variables
check_required_var() {
    local var_name="$1"
    if [[ -z "${!var_name:-}" ]]; then
        echo "ERROR: Required secret '${var_name}' is not set" >&2
        echo "Please add it to ${SECRETS_FILE}" >&2
        return 1
    fi
}

# Main function
generate_config() {
    local agent="$1"
    local debug_mode="${2:-false}"

    # Validate agent
    case "${agent}" in
        claude|codex|gemini)
            ;;
        *)
            echo "ERROR: Invalid agent '${agent}'" >&2
            echo "Valid agents: claude, codex, gemini" >&2
            usage
            ;;
    esac

    # Load secrets
    if [[ ! -f "${SECRETS_FILE}" ]]; then
        echo "ERROR: Secrets file not found at ${SECRETS_FILE}" >&2
        echo "Please create it with required secrets" >&2
        exit 1
    fi

    # Source secrets file (set -a exports all variables)
    # shellcheck disable=SC1090
    set -a
    source "${SECRETS_FILE}"
    set +a

    # Check common required variables
    local errors=0
    check_required_var "MCP_ENDPOINT" || ((errors++))
    check_required_var "MCP_API_KEY" || ((errors++))
    check_required_var "CLUSTER_NAME" || ((errors++))
    check_required_var "KUBECONFIG_PATH" || ((errors++))

    # Check agent-specific API key
    case "${agent}" in
        claude)
            check_required_var "ANTHROPIC_API_KEY" || ((errors++))
            ;;
        codex)
            check_required_var "OPENAI_API_KEY" || ((errors++))
            ;;
        gemini)
            check_required_var "GEMINI_API_KEY" || ((errors++))
            ;;
    esac

    # Exit if any required variables are missing
    if [[ ${errors} -gt 0 ]]; then
        echo "ERROR: ${errors} required secret(s) missing" >&2
        exit 1
    fi

    # Set defaults for optional variables
    export LOG_LEVEL="${LOG_LEVEL:-debug}"
    export AGENT_VERBOSE="${AGENT_VERBOSE:-true}"
    export AGENT_DEBUG="${debug_mode}"
    export SLACK_WEBHOOK_URL="${SLACK_WEBHOOK_URL:-}"
    export AZURE_STORAGE_ACCOUNT="${AZURE_STORAGE_ACCOUNT:-}"
    export AZURE_STORAGE_KEY="${AZURE_STORAGE_KEY:-}"
    export AZURE_STORAGE_CONTAINER="${AZURE_STORAGE_CONTAINER:-}"

    # Template and output paths
    local template_file="${TEMPLATES_DIR}/test-${agent}.yaml.tmpl"
    local output_file="${CONFIGS_DIR}/config-test-${agent}.yaml"

    # Validate template exists
    if [[ ! -f "${template_file}" ]]; then
        echo "ERROR: Template not found at ${template_file}" >&2
        exit 1
    fi

    # Generate config
    echo "Generating config from template..."
    echo "  Template: ${template_file}"
    echo "  Output:   ${output_file}"
    echo "  Agent:    ${agent}"
    echo "  Debug:    ${debug_mode}"

    # Use envsubst to interpolate variables
    envsubst < "${template_file}" > "${output_file}"

    # Verify output was created
    if [[ ! -f "${output_file}" ]]; then
        echo "ERROR: Failed to generate config file" >&2
        exit 1
    fi

    echo "Config generated successfully: ${output_file}"
}

# Parse arguments
main() {
    if [[ $# -eq 0 ]]; then
        usage
    fi

    local agent="$1"
    local debug_mode="false"

    # Check for --debug flag
    if [[ $# -gt 1 ]] && [[ "$2" == "--debug" ]]; then
        debug_mode="true"
    fi

    generate_config "${agent}" "${debug_mode}"
}

main "$@"
