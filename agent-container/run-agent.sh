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
# Environment Variable Configuration
# =============================================================================
#
# ALL configuration must be provided via environment variables or command-line flags.
# No default values are provided in this script.
# When invoked by nightcrier (Go), all values are set via environment variables.
# When invoked manually, use command-line flags or set environment variables.
#
# Required environment variables (when invoked by nightcrier):
#   AGENT_IMAGE          - Docker image to use
#   AGENT_CLI            - AI CLI (claude, codex, goose, gemini)
#   LLM_MODEL            - Model to use (agent-agnostic)
#   AGENT_ALLOWED_TOOLS  - Allowed tools for agent (agent-agnostic)
#   CONTAINER_TIMEOUT    - Timeout in seconds
#
# Optional environment variables:
#   ANTHROPIC_API_KEY, OPENAI_API_KEY, GEMINI_API_KEY, GOOGLE_API_KEY
#   WORKSPACE_DIR, OUTPUT_DIR, OUTPUT_FILE
#   KUBECONFIG_PATH, KUBERNETES_CONTEXT
#   OUTPUT_FORMAT        - Output format (agent-agnostic)
#   SYSTEM_PROMPT        - System prompt text (agent-agnostic)
#   SYSTEM_PROMPT_FILE   - System prompt file path (agent-agnostic)
#   AGENT_VERBOSE        - Enable verbose output
#   AGENT_MAX_TURNS      - Maximum conversation turns
#   CONTAINER_MEMORY, CONTAINER_CPUS, CONTAINER_NETWORK, CONTAINER_USER
#   SKILLS_DIR, DEBUG
#
# Legacy Claude-specific variables (still supported for backward compatibility):
#   CLAUDE_MODEL, CLAUDE_ALLOWED_TOOLS, CLAUDE_OUTPUT_FORMAT, CLAUDE_SYSTEM_PROMPT_FILE
#
# =============================================================================

# Initialize variables from environment (no defaults)
AGENT_IMAGE="${AGENT_IMAGE:-}"
AGENT_CLI="${AGENT_CLI:-}"
ANTHROPIC_API_KEY="${ANTHROPIC_API_KEY:-}"
OPENAI_API_KEY="${OPENAI_API_KEY:-}"
GEMINI_API_KEY="${GEMINI_API_KEY:-}"
GOOGLE_API_KEY="${GOOGLE_API_KEY:-}"
WORKSPACE_DIR="${WORKSPACE_DIR:-}"
OUTPUT_DIR="${OUTPUT_DIR:-}"
OUTPUT_FILE="${OUTPUT_FILE:-}"
KUBECONFIG_PATH="${KUBECONFIG_PATH:-}"
KUBERNETES_CONTEXT="${KUBERNETES_CONTEXT:-}"

# Generic agent-agnostic variables (preferred)
LLM_MODEL="${LLM_MODEL:-}"
AGENT_ALLOWED_TOOLS="${AGENT_ALLOWED_TOOLS:-}"
OUTPUT_FORMAT="${OUTPUT_FORMAT:-}"
SYSTEM_PROMPT="${SYSTEM_PROMPT:-}"
SYSTEM_PROMPT_FILE="${SYSTEM_PROMPT_FILE:-}"
AGENT_VERBOSE="${AGENT_VERBOSE:-}"
AGENT_MAX_TURNS="${AGENT_MAX_TURNS:-}"

# Legacy Claude-specific variables (for backward compatibility)
# Fall back to these if generic ones not set
[[ -z "$LLM_MODEL" && -n "${CLAUDE_MODEL:-}" ]] && LLM_MODEL="$CLAUDE_MODEL"
[[ -z "$AGENT_ALLOWED_TOOLS" && -n "${CLAUDE_ALLOWED_TOOLS:-}" ]] && AGENT_ALLOWED_TOOLS="$CLAUDE_ALLOWED_TOOLS"
[[ -z "$OUTPUT_FORMAT" && -n "${CLAUDE_OUTPUT_FORMAT:-}" ]] && OUTPUT_FORMAT="$CLAUDE_OUTPUT_FORMAT"
[[ -z "$SYSTEM_PROMPT" && -n "${CLAUDE_SYSTEM_PROMPT:-}" ]] && SYSTEM_PROMPT="$CLAUDE_SYSTEM_PROMPT"
[[ -z "$SYSTEM_PROMPT_FILE" && -n "${CLAUDE_SYSTEM_PROMPT_FILE:-}" ]] && SYSTEM_PROMPT_FILE="$CLAUDE_SYSTEM_PROMPT_FILE"
[[ -z "$AGENT_VERBOSE" && -n "${CLAUDE_VERBOSE:-}" ]] && AGENT_VERBOSE="$CLAUDE_VERBOSE"
[[ -z "$AGENT_MAX_TURNS" && -n "${CLAUDE_MAX_TURNS:-}" ]] && AGENT_MAX_TURNS="$CLAUDE_MAX_TURNS"

CONTAINER_TIMEOUT="${CONTAINER_TIMEOUT:-}"
CONTAINER_MEMORY="${CONTAINER_MEMORY:-}"
CONTAINER_CPUS="${CONTAINER_CPUS:-}"
CONTAINER_NETWORK="${CONTAINER_NETWORK:-}"
CONTAINER_USER="${CONTAINER_USER:-}"
SKILLS_DIR="${SKILLS_DIR:-}"
DEBUG="${DEBUG:-}"

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
  -w, --workspace DIR           Host incident directory containing incident.json

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

Codex CLI Options (when --agent codex):
  -m, --model MODEL             Model mapping: opus->gpt-5-codex, sonnet->gpt-5.2, haiku->gpt-4o
                                Or specify directly: gpt-5-codex, gpt-5.2, gpt-4o (default: gpt-5.2)
  Automatically enabled flags:
    --enable skills             Load SKILL.md from ~/.codex/skills/
    --dangerously-bypass-approvals-and-sandbox
                                Bypass sandbox (required for Docker without Landlock)

Container Options:
  -i, --image IMAGE             Docker image (default: nightcrier-agent:latest)
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
  ./run-agent.sh -a claude -m opus -t "Read,Grep,Glob,Bash" "Deep analysis of incident.json"

  # With specific workspace and output directory
  ./run-agent.sh -w ./incidents/abc123 --output-dir ./reports "Analyze incident.json"

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
            LLM_MODEL="$2"
            shift 2
            ;;
        -o|--output-format)
            OUTPUT_FORMAT="$2"
            shift 2
            ;;
        -t|--allowed-tools)
            AGENT_ALLOWED_TOOLS="$2"
            shift 2
            ;;
        -s|--system-prompt)
            SYSTEM_PROMPT="$2"
            shift 2
            ;;
        --system-prompt-file)
            SYSTEM_PROMPT_FILE="$2"
            shift 2
            ;;
        -v|--verbose)
            AGENT_VERBOSE="true"
            shift
            ;;
        --max-turns)
            AGENT_MAX_TURNS="$2"
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

# Validate required environment variables when invoked programmatically
# (These can be overridden by command-line flags for manual usage)
validate_required_vars() {
    local missing=()

    # Check required vars that should always be set by nightcrier
    [[ -z "$AGENT_CLI" ]] && missing+=("AGENT_CLI")
    [[ -z "$AGENT_IMAGE" ]] && missing+=("AGENT_IMAGE")
    [[ -z "$LLM_MODEL" ]] && missing+=("LLM_MODEL")
    [[ -z "$AGENT_ALLOWED_TOOLS" ]] && missing+=("AGENT_ALLOWED_TOOLS")
    [[ -z "$CONTAINER_TIMEOUT" ]] && missing+=("CONTAINER_TIMEOUT")

    if [[ ${#missing[@]} -gt 0 ]]; then
        echo "Error: Required environment variables not set: ${missing[*]}" >&2
        echo "These must be set by the calling application (nightcrier) or via environment." >&2
        echo "For manual usage, use command-line flags instead (see --help)." >&2
        return 1
    fi
    return 0
}

# Only validate required vars if they're not being set by flags
# (detect if we're being called programmatically vs manually)
if [[ $# -eq 0 ]]; then
    # No arguments - must be called programmatically with env vars
    validate_required_vars || exit 1
fi

# Validate agent selection
if [[ -n "$AGENT_CLI" ]]; then
    case "$AGENT_CLI" in
        claude|codex|goose|gemini)
            ;;
        *)
            echo "Error: Invalid agent '$AGENT_CLI'. Must be one of: claude, codex, goose, gemini" >&2
            exit 1
            ;;
    esac
fi

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
    echo "Example: ./run-agent.sh -w ./incidents/inc-123 \"Investigate incident.json\"" >&2
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
)

# Remove containers automatically in production mode
# In DEBUG mode, keep the container so we can extract the Claude session
if [[ "$DEBUG" != "true" ]]; then
    DOCKER_ARGS+=("--rm")
fi

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

# Volume mounts - everything goes into /home/agent (the agent's home AND workspace)
# This keeps skills, config files, and incident data all in one place
AGENT_HOME="/home/agent"

# Mount incident.json directly into agent home
if [[ -f "${WORKSPACE_DIR}/incident.json" ]]; then
    DOCKER_ARGS+=("-v" "${WORKSPACE_DIR}/incident.json:${AGENT_HOME}/incident.json:ro")
fi

# Mount incident_cluster_permissions.json (Phase 3: multi-cluster support)
if [[ -f "${WORKSPACE_DIR}/incident_cluster_permissions.json" ]]; then
    DOCKER_ARGS+=("-v" "${WORKSPACE_DIR}/incident_cluster_permissions.json:${AGENT_HOME}/incident_cluster_permissions.json:ro")
fi

# Mount output directory into agent home (read-write for agent to write results)
mkdir -p "${OUTPUT_DIR}"
DOCKER_ARGS+=("-v" "${OUTPUT_DIR}:${AGENT_HOME}/output")

# Mount kubeconfig
if [[ -f "$KUBECONFIG_PATH" ]]; then
    DOCKER_ARGS+=("-v" "${KUBECONFIG_PATH}:${AGENT_HOME}/.kube/config:ro")
fi

# Mount skills directory if specified (overrides built-in skills)
if [[ -n "$SKILLS_DIR" && -d "$SKILLS_DIR" ]]; then
    SKILLS_DIR_ABS="$(cd "$SKILLS_DIR" && pwd)"
    DOCKER_ARGS+=("-v" "${SKILLS_DIR_ABS}:${AGENT_HOME}/.claude/skills:ro")
    DOCKER_ARGS+=("-v" "${SKILLS_DIR_ABS}:${AGENT_HOME}/.codex/skills:ro")
fi

# Mount system prompt file if specified
if [[ -n "$SYSTEM_PROMPT_FILE" && -f "$SYSTEM_PROMPT_FILE" ]]; then
    SYSTEM_PROMPT_FILE_ABS="$(cd "$(dirname "$SYSTEM_PROMPT_FILE")" && pwd)/$(basename "$SYSTEM_PROMPT_FILE")"
    DOCKER_ARGS+=("-v" "${SYSTEM_PROMPT_FILE_ABS}:/tmp/system-prompt.txt:ro")
fi

# Working directory IS the agent home - everything in one place
DOCKER_ARGS+=("-w" "${AGENT_HOME}")

# Container name for reliable session archive extraction
if [[ -n "$INCIDENT_ID" ]]; then
    DOCKER_ARGS+=("--name" "nightcrier-agent-${INCIDENT_ID}")
fi

# Image
DOCKER_ARGS+=("$AGENT_IMAGE")

# =============================================================================
# Build Agent CLI Command
# =============================================================================

build_agent_command() {
    local cmd=""

    # Add environment verification if debug mode (runs inside container)
    if [[ "$DEBUG" == "true" ]]; then
        cmd="echo '=== Container Environment ===' >&2 && "
        cmd+="echo \"PWD: \$(pwd)\" >&2 && "
        cmd+="echo \"USER: \$(whoami)\" >&2 && "
        cmd+="echo \"HOME: \$HOME\" >&2 && "
        cmd+="echo \"ANTHROPIC_API_KEY: \${ANTHROPIC_API_KEY:+SET}\" >&2 && "
        cmd+="echo \"Kubeconfig exists: \$(test -f \$HOME/.kube/config && echo YES || echo NO)\" >&2 && "
        cmd+="echo \"incident.json exists: \$(test -f incident.json && echo YES || echo NO)\" >&2 && "
        cmd+="echo \"Files in workspace: \$(ls -la | wc -l) files\" >&2 && "
        cmd+="echo '==============================' >&2 && "
    fi

    case "$AGENT_CLI" in
        claude)
            cmd+="claude -p '${PROMPT//\'/\'\\\'\'}'"

            # Model
            if [[ -n "$LLM_MODEL" ]]; then
                cmd+=" --model $LLM_MODEL"
            fi

            # Output format
            if [[ -n "$OUTPUT_FORMAT" ]]; then
                cmd+=" --output-format $OUTPUT_FORMAT"
            fi

            # Allowed tools
            if [[ -n "$AGENT_ALLOWED_TOOLS" ]]; then
                cmd+=" --allowedTools $AGENT_ALLOWED_TOOLS"
            fi

            # System prompt (inline)
            if [[ -n "$SYSTEM_PROMPT" ]]; then
                cmd+=" --append-system-prompt '${SYSTEM_PROMPT//\'/\'\\\'\'}'"
            fi

            # System prompt (file)
            if [[ -n "$SYSTEM_PROMPT_FILE" ]]; then
                cmd+=" --append-system-prompt-file /tmp/system-prompt.txt"
            fi

            # Verbose
            if [[ "$AGENT_VERBOSE" == "true" ]]; then
                cmd+=" --verbose"
            fi

            # Max turns
            if [[ -n "$AGENT_MAX_TURNS" ]]; then
                cmd+=" --max-turns $AGENT_MAX_TURNS"
            fi
            ;;

        codex)
            # Use 'codex exec' for non-interactive/headless mode
            # Must login with API key first (codex doesn't auto-use OPENAI_API_KEY)
            # --skip-git-repo-check needed when not in a git repo
            # --enable skills to enable SKILL.md loading from ~/.codex/skills
            # In Docker containers without Landlock, we need --dangerously-bypass-approvals-and-sandbox
            # AGENTS.md is in /home/agent (the working directory) so Codex finds it automatically
            cmd="echo -n \"\${OPENAI_API_KEY}\" > /tmp/.codex-key && codex login --with-api-key < /tmp/.codex-key && rm -f /tmp/.codex-key && codex exec --skip-git-repo-check --enable skills --dangerously-bypass-approvals-and-sandbox"

            # Model mapping for Codex
            # Codex uses OpenAI model names, but we support friendly aliases
            if [[ -n "$LLM_MODEL" ]]; then
                case "$LLM_MODEL" in
                    opus|gpt-5-codex)
                        cmd+=" -m gpt-5-codex"
                        ;;
                    sonnet|gpt-5.2)
                        cmd+=" -m gpt-5.2"
                        ;;
                    haiku|gpt-4o)
                        cmd+=" -m gpt-4o"
                        ;;
                    *)
                        # Pass through custom model name
                        cmd+=" -m $LLM_MODEL"
                        ;;
                esac
            fi

            cmd+=" '${PROMPT//\'/\'\\\'\'}'"
            ;;

        goose)
            cmd="goose run"

            # Model for Goose (supports various providers)
            if [[ -n "$LLM_MODEL" ]]; then
                cmd+=" --model $LLM_MODEL"
            fi

            cmd+=" '${PROMPT//\'/\'\\\'\'}'"
            ;;

        gemini)
            cmd="gemini -p '${PROMPT//\'/\'\\\'\'}'"

            # Model for Gemini
            if [[ -n "$LLM_MODEL" ]]; then
                cmd+=" --model $LLM_MODEL"
            fi
            ;;
    esac

    # Tee output to file
    cmd+=" 2>&1 | tee ${AGENT_HOME}/output/${OUTPUT_FILE}"

    echo "$cmd"
}

AGENT_CMD=$(build_agent_command)

# =============================================================================
# Execute
# =============================================================================

if [[ "$DEBUG" == "true" ]]; then
    echo "=== Debug Information ===" >&2
    echo "Agent: $AGENT_CLI" >&2
    echo "Model: $LLM_MODEL" >&2
    echo "Verbose: $AGENT_VERBOSE" >&2
    echo "Workspace: $WORKSPACE_DIR" >&2
    echo "Output: $OUTPUT_PATH" >&2
    echo "Image: $AGENT_IMAGE" >&2
    echo "Kubeconfig: $KUBECONFIG_PATH" >&2
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

# =============================================================================
# Post-Run Hooks
# =============================================================================
# This section handles tasks that run after the agent completes.
# Add new post-run tasks as separate functions below.

# Post-run hook: Extract executed commands from Claude session JSONL files
# Creates agent-commands-executed.log with all Bash commands run by the agent
extract_agent_commands() {
    local session_dir="$1"
    local output_file="$WORKSPACE_DIR/logs/agent-commands-executed.log"

    # Find the most recent session JSONL file
    local jsonl_file
    jsonl_file=$(find "$session_dir/projects" -name "*.jsonl" -type f 2>/dev/null | \
                 xargs ls -t 2>/dev/null | head -1)

    if [[ -z "$jsonl_file" || ! -f "$jsonl_file" ]]; then
        echo "DEBUG: No session JSONL file found for command extraction" >&2
        return 0
    fi

    echo "DEBUG: Extracting commands from: $jsonl_file" >&2

    # Extract Bash tool calls using jq
    # Format: timestamp | command | description
    {
        echo "# Agent Commands Executed"
        echo "# Generated: $(date -u +"%Y-%m-%dT%H:%M:%SZ")"
        echo "# Incident: ${INCIDENT_ID:-unknown}"
        echo "# Session: $(basename "$jsonl_file")"
        echo "#"
        echo ""

        # Parse JSONL and extract Bash commands
        jq -r '
            select(.type == "assistant") |
            .message.content[]? |
            select(.type == "tool_use" and .name == "Bash") |
            "$ " + .input.command + (if .input.description then " # " + .input.description else "" end)
        ' "$jsonl_file" 2>/dev/null

    } > "$output_file"

    local cmd_count
    cmd_count=$(grep -c '^\$' "$output_file" 2>/dev/null || echo "0")
    echo "DEBUG: Extracted $cmd_count commands to agent-commands-executed.log" >&2
}

# Post-run hook: Extract Claude session archive (DEBUG mode only)
post_run_extract_claude_session() {
    if [[ "$DEBUG" != "true" ]]; then
        return 0
    fi

    CONTAINER_NAME="nightcrier-agent-${INCIDENT_ID}"
    if [[ -z "$INCIDENT_ID" ]]; then
        return 0
    fi

    echo "DEBUG: Post-run: Extracting Claude session from container: $CONTAINER_NAME" >&2

    # Extract the session directory from the container
    if docker cp "$CONTAINER_NAME:/home/agent/.claude" "$WORKSPACE_DIR/claude-session-src" 2>/dev/null; then
        mkdir -p "$WORKSPACE_DIR/logs"

        # Extract commands before archiving
        extract_agent_commands "$WORKSPACE_DIR/claude-session-src"

        cd "$WORKSPACE_DIR"
        tar -czf "$WORKSPACE_DIR/logs/claude-session.tar.gz" -C "$WORKSPACE_DIR" claude-session-src
        echo "DEBUG: Post-run: Claude session archive saved to $WORKSPACE_DIR/logs/claude-session.tar.gz" >&2
        rm -rf "$WORKSPACE_DIR/claude-session-src"
        return 0
    else
        echo "DEBUG: Post-run: Could not extract Claude session (session may not exist)" >&2
        return 0
    fi
}

# Execute all post-run hooks
# Add new hooks here:
post_run_extract_claude_session

exit $EXIT_CODE
