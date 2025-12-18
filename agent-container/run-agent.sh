#!/usr/bin/env bash
#
# run-agent.sh - Run AI triage agent in Docker container for Kubernetes incident investigation
#
# Usage: ./run-agent.sh [options] [prompt]
#
# Supports multiple AI CLIs: claude (default), codex, goose, gemini
# All parameters can be set via environment variables or command-line flags.
# Command-line flags override environment variables.
#

set -euo pipefail

# =============================================================================
# Default Configuration (override via environment or flags)
# =============================================================================

# Docker image
AGENT_IMAGE="${AGENT_IMAGE:-k8s-triage-agent:latest}"

# Agent selection (claude, codex, goose, gemini)
AGENT_CLI="${AGENT_CLI:-claude}"

# API Authentication (set the appropriate one for your agent)
ANTHROPIC_API_KEY="${ANTHROPIC_API_KEY:-}"
OPENAI_API_KEY="${OPENAI_API_KEY:-}"
GEMINI_API_KEY="${GEMINI_API_KEY:-}"
GOOGLE_API_KEY="${GOOGLE_API_KEY:-}"

# Workspace configuration
# IMPORTANT: Workspace should be an INCIDENT directory, not source code!
# The container will have read/write access to this directory.
WORKSPACE_DIR="${WORKSPACE_DIR:-}"  # Required - must be specified
CONTAINER_WORKSPACE="${CONTAINER_WORKSPACE:-/workspace}"

# Output configuration
OUTPUT_DIR="${OUTPUT_DIR:-}"  # If empty, uses WORKSPACE_DIR/output
OUTPUT_FILE="${OUTPUT_FILE:-}"  # If empty, auto-generates based on timestamp

# Kubernetes configuration
KUBECONFIG_PATH="${KUBECONFIG_PATH:-${HOME}/.kube/config}"
KUBERNETES_CONTEXT="${KUBERNETES_CONTEXT:-}"

# Claude CLI options (default agent)
CLAUDE_MODEL="${CLAUDE_MODEL:-sonnet}"
CLAUDE_OUTPUT_FORMAT="${CLAUDE_OUTPUT_FORMAT:-text}"
CLAUDE_ALLOWED_TOOLS="${CLAUDE_ALLOWED_TOOLS:-Read,Grep,Glob,Bash}"
CLAUDE_SYSTEM_PROMPT="${CLAUDE_SYSTEM_PROMPT:-}"
CLAUDE_SYSTEM_PROMPT_FILE="${CLAUDE_SYSTEM_PROMPT_FILE:-}"
CLAUDE_VERBOSE="${CLAUDE_VERBOSE:-false}"
CLAUDE_MAX_TURNS="${CLAUDE_MAX_TURNS:-}"

# Container options
CONTAINER_TIMEOUT="${CONTAINER_TIMEOUT:-600}"
CONTAINER_MEMORY="${CONTAINER_MEMORY:-2g}"
CONTAINER_CPUS="${CONTAINER_CPUS:-}"
CONTAINER_NETWORK="${CONTAINER_NETWORK:-host}"
CONTAINER_USER="${CONTAINER_USER:-}"

# Skills
SKILLS_DIR="${SKILLS_DIR:-}"

# Debug mode
DEBUG="${DEBUG:-false}"

# =============================================================================
# Help
# =============================================================================

show_help() {
    cat << 'EOF'
Usage: run-agent.sh [OPTIONS] [PROMPT]

Run AI triage agent in Docker container for Kubernetes incident investigation.

AGENT SELECTION:
  -a, --agent AGENT             AI CLI to use: claude, codex, goose, gemini (default: claude)

API Authentication (set for your chosen agent):
  --anthropic-key KEY           Anthropic API key for Claude (or ANTHROPIC_API_KEY)
  --openai-key KEY              OpenAI API key for Codex (or OPENAI_API_KEY)
  --gemini-key KEY              Gemini API key (or GEMINI_API_KEY or GOOGLE_API_KEY)

OPTIONS:
  -h, --help                    Show this help message
  -d, --debug                   Enable debug output

Workspace:
  -w, --workspace DIR           Host workspace directory (default: current dir)
  --container-workspace PATH    Container workspace path (default: /workspace)

Output:
  --output-dir DIR              Directory for output files (default: workspace/output)
  --output-file FILE            Output filename (default: auto-generated)

Kubernetes:
  --kubeconfig PATH             Path to kubeconfig file (default: ~/.kube/config)
  --context NAME                Kubernetes context to use

Claude CLI Options (when --agent claude):
  -m, --model MODEL             Claude model: opus, sonnet, haiku (default: sonnet)
  -o, --output-format FORMAT    Output format: text, json, stream-json (default: text)
  -t, --allowed-tools TOOLS     Comma-separated allowed tools (default: Read,Grep,Glob,Bash)
  -s, --system-prompt TEXT      System prompt to append
  --system-prompt-file PATH     File containing system prompt (host path)
  -v, --verbose                 Enable verbose Claude output
  --max-turns N                 Maximum conversation turns

Container Options:
  -i, --image IMAGE             Docker image (default: k8s-triage-agent:latest)
  --timeout SECONDS             Container timeout (default: 600)
  --memory LIMIT                Memory limit (default: 2g)
  --cpus LIMIT                  CPU limit (e.g., 1.5)
  --network MODE                Network mode (default: host)
  --user UID:GID                Run as specific user

Skills:
  --skills-dir DIR              Directory containing skills to mount

ENVIRONMENT VARIABLES:
  AGENT_CLI                     AI CLI to use (claude, codex, goose, gemini)
  ANTHROPIC_API_KEY             API key for Claude
  OPENAI_API_KEY                API key for Codex
  GEMINI_API_KEY                API key for Gemini
  AGENT_IMAGE                   Docker image name
  WORKSPACE_DIR                 Host workspace directory
  OUTPUT_DIR                    Output directory
  KUBECONFIG_PATH               Path to kubeconfig
  KUBERNETES_CONTEXT            Kubernetes context
  CLAUDE_MODEL                  Claude model (default: sonnet)
  CLAUDE_OUTPUT_FORMAT          Output format
  CLAUDE_ALLOWED_TOOLS          Allowed tools
  CLAUDE_SYSTEM_PROMPT          System prompt text
  CLAUDE_VERBOSE                Enable verbose (true/false)
  CLAUDE_MAX_TURNS              Max conversation turns
  CONTAINER_TIMEOUT             Timeout in seconds
  CONTAINER_MEMORY              Memory limit
  CONTAINER_CPUS                CPU limit
  CONTAINER_NETWORK             Network mode
  SKILLS_DIR                    Skills directory

EXAMPLES:
  # Basic usage with Claude (default)
  ./run-agent.sh "Investigate the failing pod in namespace default"

  # Use Codex instead of Claude
  ./run-agent.sh -a codex "Analyze the CrashLoopBackOff issue"

  # Use Gemini
  ./run-agent.sh -a gemini "Check cluster health"

  # Claude with specific model and tools
  ./run-agent.sh -a claude -m opus -t "Read,Grep,Glob,Bash" "Deep analysis of event.json"

  # With specific workspace and output directory
  ./run-agent.sh -w ./incidents/abc123 --output-dir ./reports "Analyze event.json"

  # Read-only kubectl investigation
  ./run-agent.sh -t "Read,Grep,Glob,Bash" \
    -s "Only use kubectl get, describe, and logs commands. Never modify cluster state." \
    "Investigate why pod my-app is in CrashLoopBackOff"

EOF
}

# =============================================================================
# Argument Parsing
# =============================================================================

PROMPT=""

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_help
            exit 0
            ;;
        -d|--debug)
            DEBUG="true"
            shift
            ;;
        -a|--agent)
            AGENT_CLI="$2"
            shift 2
            ;;
        --anthropic-key)
            ANTHROPIC_API_KEY="$2"
            shift 2
            ;;
        --openai-key)
            OPENAI_API_KEY="$2"
            shift 2
            ;;
        --gemini-key)
            GEMINI_API_KEY="$2"
            GOOGLE_API_KEY="$2"
            shift 2
            ;;
        -w|--workspace)
            WORKSPACE_DIR="$2"
            shift 2
            ;;
        --container-workspace)
            CONTAINER_WORKSPACE="$2"
            shift 2
            ;;
        --output-dir)
            OUTPUT_DIR="$2"
            shift 2
            ;;
        --output-file)
            OUTPUT_FILE="$2"
            shift 2
            ;;
        --kubeconfig)
            KUBECONFIG_PATH="$2"
            shift 2
            ;;
        --context)
            KUBERNETES_CONTEXT="$2"
            shift 2
            ;;
        -m|--model)
            CLAUDE_MODEL="$2"
            shift 2
            ;;
        -o|--output-format)
            CLAUDE_OUTPUT_FORMAT="$2"
            shift 2
            ;;
        -t|--allowed-tools)
            CLAUDE_ALLOWED_TOOLS="$2"
            shift 2
            ;;
        -s|--system-prompt)
            CLAUDE_SYSTEM_PROMPT="$2"
            shift 2
            ;;
        --system-prompt-file)
            CLAUDE_SYSTEM_PROMPT_FILE="$2"
            shift 2
            ;;
        -v|--verbose)
            CLAUDE_VERBOSE="true"
            shift
            ;;
        --max-turns)
            CLAUDE_MAX_TURNS="$2"
            shift 2
            ;;
        -i|--image)
            AGENT_IMAGE="$2"
            shift 2
            ;;
        --timeout)
            CONTAINER_TIMEOUT="$2"
            shift 2
            ;;
        --memory)
            CONTAINER_MEMORY="$2"
            shift 2
            ;;
        --cpus)
            CONTAINER_CPUS="$2"
            shift 2
            ;;
        --network)
            CONTAINER_NETWORK="$2"
            shift 2
            ;;
        --user)
            CONTAINER_USER="$2"
            shift 2
            ;;
        --skills-dir)
            SKILLS_DIR="$2"
            shift 2
            ;;
        -*|--*)
            echo "Error: Unknown option: $1" >&2
            echo "Use --help for usage information" >&2
            exit 1
            ;;
        *)
            # Remaining arguments are the prompt
            PROMPT="$*"
            break
            ;;
    esac
done

# =============================================================================
# Validation
# =============================================================================

# Validate agent selection
case "$AGENT_CLI" in
    claude|codex|goose|gemini)
        ;;
    *)
        echo "Error: Invalid agent '$AGENT_CLI'. Must be one of: claude, codex, goose, gemini" >&2
        exit 1
        ;;
esac

# Validate API key for selected agent
validate_api_key() {
    case "$AGENT_CLI" in
        claude)
            if [[ -z "$ANTHROPIC_API_KEY" ]]; then
                echo "Error: ANTHROPIC_API_KEY is required for Claude" >&2
                echo "Set via --anthropic-key flag or ANTHROPIC_API_KEY environment variable" >&2
                exit 1
            fi
            ;;
        codex)
            if [[ -z "$OPENAI_API_KEY" ]]; then
                echo "Error: OPENAI_API_KEY is required for Codex" >&2
                echo "Set via --openai-key flag or OPENAI_API_KEY environment variable" >&2
                exit 1
            fi
            ;;
        gemini)
            if [[ -z "$GEMINI_API_KEY" && -z "$GOOGLE_API_KEY" ]]; then
                echo "Error: GEMINI_API_KEY or GOOGLE_API_KEY is required for Gemini" >&2
                echo "Set via --gemini-key flag or GEMINI_API_KEY environment variable" >&2
                exit 1
            fi
            ;;
        goose)
            # Goose can use various providers, check common ones
            if [[ -z "$ANTHROPIC_API_KEY" && -z "$OPENAI_API_KEY" && -z "$GEMINI_API_KEY" ]]; then
                echo "Warning: No API key found for Goose. Set ANTHROPIC_API_KEY, OPENAI_API_KEY, or GEMINI_API_KEY" >&2
            fi
            ;;
    esac
}

validate_api_key

if [[ -z "$PROMPT" ]]; then
    echo "Error: Prompt is required" >&2
    echo "Use --help for usage information" >&2
    exit 1
fi

if [[ -z "$WORKSPACE_DIR" ]]; then
    echo "Error: Workspace directory is required (-w flag or WORKSPACE_DIR env)" >&2
    echo "The workspace should be an INCIDENT directory, not source code." >&2
    echo "Example: ./run-agent.sh -w ./incidents/inc-123 \"Investigate event.json\"" >&2
    exit 1
fi

if [[ ! -d "$WORKSPACE_DIR" ]]; then
    echo "Error: Workspace directory does not exist: $WORKSPACE_DIR" >&2
    exit 1
fi

# Convert to absolute path
WORKSPACE_DIR="$(cd "$WORKSPACE_DIR" && pwd)"

# Setup output directory and file
if [[ -z "$OUTPUT_DIR" ]]; then
    OUTPUT_DIR="${WORKSPACE_DIR}/output"
fi
mkdir -p "$OUTPUT_DIR"

if [[ -z "$OUTPUT_FILE" ]]; then
    TIMESTAMP=$(date +%Y%m%d_%H%M%S)
    OUTPUT_FILE="triage_${AGENT_CLI}_${TIMESTAMP}.log"
fi
OUTPUT_PATH="${OUTPUT_DIR}/${OUTPUT_FILE}"

# =============================================================================
# Build Docker Command
# =============================================================================

DOCKER_ARGS=(
    "run"
    "--rm"
)

# Timeout via timeout command wrapper
if [[ -n "$CONTAINER_TIMEOUT" ]]; then
    DOCKER_ARGS+=("--stop-timeout" "$CONTAINER_TIMEOUT")
fi

# Memory limit
if [[ -n "$CONTAINER_MEMORY" ]]; then
    DOCKER_ARGS+=("--memory" "$CONTAINER_MEMORY")
fi

# CPU limit
if [[ -n "$CONTAINER_CPUS" ]]; then
    DOCKER_ARGS+=("--cpus" "$CONTAINER_CPUS")
fi

# Network mode
if [[ -n "$CONTAINER_NETWORK" ]]; then
    DOCKER_ARGS+=("--network" "$CONTAINER_NETWORK")
fi

# User
if [[ -n "$CONTAINER_USER" ]]; then
    DOCKER_ARGS+=("--user" "$CONTAINER_USER")
fi

# Environment variables based on agent
if [[ -n "$ANTHROPIC_API_KEY" ]]; then
    DOCKER_ARGS+=("-e" "ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}")
fi
if [[ -n "$OPENAI_API_KEY" ]]; then
    DOCKER_ARGS+=("-e" "OPENAI_API_KEY=${OPENAI_API_KEY}")
fi
if [[ -n "$GEMINI_API_KEY" ]]; then
    DOCKER_ARGS+=("-e" "GEMINI_API_KEY=${GEMINI_API_KEY}")
fi
if [[ -n "$GOOGLE_API_KEY" ]]; then
    DOCKER_ARGS+=("-e" "GOOGLE_API_KEY=${GOOGLE_API_KEY}")
fi

if [[ -n "$KUBERNETES_CONTEXT" ]]; then
    DOCKER_ARGS+=("-e" "KUBERNETES_CONTEXT=${KUBERNETES_CONTEXT}")
fi

# Volume mounts
DOCKER_ARGS+=("-v" "${WORKSPACE_DIR}:${CONTAINER_WORKSPACE}")
DOCKER_ARGS+=("-v" "${OUTPUT_DIR}:/output")

# Mount kubeconfig if it exists (to non-root user's home directory)
if [[ -f "$KUBECONFIG_PATH" ]]; then
    DOCKER_ARGS+=("-v" "${KUBECONFIG_PATH}:/home/agent/.kube/config:ro")
fi

# Mount skills directory if specified (overrides built-in skills, convert to absolute path)
if [[ -n "$SKILLS_DIR" && -d "$SKILLS_DIR" ]]; then
    SKILLS_DIR_ABS="$(cd "$SKILLS_DIR" && pwd)"
    DOCKER_ARGS+=("-v" "${SKILLS_DIR_ABS}:/home/agent/.claude/skills:ro")
fi

# Mount system prompt file if specified (convert to absolute path for Docker)
if [[ -n "$CLAUDE_SYSTEM_PROMPT_FILE" && -f "$CLAUDE_SYSTEM_PROMPT_FILE" ]]; then
    CLAUDE_SYSTEM_PROMPT_FILE_ABS="$(cd "$(dirname "$CLAUDE_SYSTEM_PROMPT_FILE")" && pwd)/$(basename "$CLAUDE_SYSTEM_PROMPT_FILE")"
    DOCKER_ARGS+=("-v" "${CLAUDE_SYSTEM_PROMPT_FILE_ABS}:/tmp/system-prompt.txt:ro")
fi

# Working directory
DOCKER_ARGS+=("-w" "$CONTAINER_WORKSPACE")

# Image
DOCKER_ARGS+=("$AGENT_IMAGE")

# =============================================================================
# Build Agent CLI Command
# =============================================================================

build_agent_command() {
    local cmd=""

    case "$AGENT_CLI" in
        claude)
            cmd="claude -p '${PROMPT//\'/\'\\\'\'}'"

            # Model
            if [[ -n "$CLAUDE_MODEL" ]]; then
                cmd+=" --model $CLAUDE_MODEL"
            fi

            # Output format
            if [[ -n "$CLAUDE_OUTPUT_FORMAT" ]]; then
                cmd+=" --output-format $CLAUDE_OUTPUT_FORMAT"
            fi

            # Allowed tools
            if [[ -n "$CLAUDE_ALLOWED_TOOLS" ]]; then
                cmd+=" --allowedTools $CLAUDE_ALLOWED_TOOLS"
            fi

            # System prompt (inline)
            if [[ -n "$CLAUDE_SYSTEM_PROMPT" ]]; then
                cmd+=" --append-system-prompt '${CLAUDE_SYSTEM_PROMPT//\'/\'\\\'\'}'"
            fi

            # System prompt (file)
            if [[ -n "$CLAUDE_SYSTEM_PROMPT_FILE" ]]; then
                cmd+=" --append-system-prompt-file /tmp/system-prompt.txt"
            fi

            # Verbose
            if [[ "$CLAUDE_VERBOSE" == "true" ]]; then
                cmd+=" --verbose"
            fi

            # Max turns
            if [[ -n "$CLAUDE_MAX_TURNS" ]]; then
                cmd+=" --max-turns $CLAUDE_MAX_TURNS"
            fi
            ;;

        codex)
            # Use 'codex exec' for non-interactive/headless mode
            # Must login with API key first (codex doesn't auto-use OPENAI_API_KEY)
            # --skip-git-repo-check needed when not in a git repo
            cmd="echo -n \"\${OPENAI_API_KEY}\" > /tmp/.codex-key && codex login --with-api-key < /tmp/.codex-key && rm -f /tmp/.codex-key && codex exec --skip-git-repo-check '${PROMPT//\'/\'\\\'\'}'"
            ;;

        goose)
            cmd="goose run '${PROMPT//\'/\'\\\'\'}'"
            ;;

        gemini)
            cmd="gemini -p '${PROMPT//\'/\'\\\'\'}'"
            ;;
    esac

    # Tee output to file
    cmd+=" 2>&1 | tee /output/${OUTPUT_FILE}"

    echo "$cmd"
}

AGENT_CMD=$(build_agent_command)

# =============================================================================
# Execute
# =============================================================================

if [[ "$DEBUG" == "true" ]]; then
    echo "=== Debug Information ===" >&2
    echo "Agent: $AGENT_CLI" >&2
    echo "Workspace: $WORKSPACE_DIR" >&2
    echo "Output: $OUTPUT_PATH" >&2
    echo "Image: $AGENT_IMAGE" >&2
    echo "Docker args: ${DOCKER_ARGS[*]}" >&2
    echo "Agent command: $AGENT_CMD" >&2
    echo "=========================" >&2
fi

echo "Starting $AGENT_CLI triage agent..." >&2
echo "Output will be saved to: $OUTPUT_PATH" >&2
echo "" >&2

# Run with timeout wrapper
if [[ -n "$CONTAINER_TIMEOUT" ]]; then
    timeout "$CONTAINER_TIMEOUT" docker "${DOCKER_ARGS[@]}" "$AGENT_CMD"
    EXIT_CODE=$?
else
    docker "${DOCKER_ARGS[@]}" "$AGENT_CMD"
    EXIT_CODE=$?
fi

echo "" >&2
echo "Agent completed with exit code: $EXIT_CODE" >&2
echo "Output saved to: $OUTPUT_PATH" >&2

exit $EXIT_CODE
