#!/usr/bin/env bash
#
# nc-agent-runner entrypoint script
#
# This script orchestrates the execution of AI agents (Claude, Codex, Gemini, Goose)
# for Kubernetes incident triage. It handles:
# - Agent-specific skill path setup
# - Agent invocation with real-time logging
# - Command extraction from agent session data
# - Artifact upload to object storage
#

set -euo pipefail

# Environment variables (set by K8s Job spec)
# - AGENT_CLI: claude|codex|gemini|goose
# - LLM_MODEL: AI model to use
# - INCIDENT_ID: Unique incident identifier
# - PROMPT: Triage prompt for the agent
# - OUTPUT_URL_REPORT: Presigned PUT URL for report.md
# - OUTPUT_URL_LOG: Presigned PUT URL for agent.log
# - OUTPUT_URL_SESSION: Presigned PUT URL for session.tar.gz
# - OUTPUT_URL_RESULT: Presigned PUT URL for result.json
# - OUTPUT_URL_COMMANDS: Presigned PUT URL for commands-executed.log
# - ANTHROPIC_API_KEY: API key for Claude (optional)
# - OPENAI_API_KEY: API key for Codex (optional)
# - GEMINI_API_KEY: API key for Gemini (optional)

# Global variables
EXIT_CODE=0
SESSION_DIR=""

#######################################
# Clone k8s4agents skill repository if not present
# Globals:
#   None
# Arguments:
#   None
#######################################
clone_skills() {
    local skills_base="/home/agent/skills"
    local k8s_skill_path="${skills_base}/k8s4agents"

    echo "Checking for k8s4agents skill repository..."

    if [[ -d "$k8s_skill_path" && -f "$k8s_skill_path/skills/k8s-troubleshooter/SKILL.md" ]]; then
        echo "k8s4agents skill already exists at: $k8s_skill_path"
        return 0
    fi

    echo "Cloning k8s4agents skill repository from GitHub..."
    mkdir -p "$skills_base"

    if git clone --depth 1 https://github.com/randybias/k8s4agents "$k8s_skill_path"; then
        echo "Successfully cloned k8s4agents skill"
        # Verify the skill structure
        if [[ -f "$k8s_skill_path/skills/k8s-troubleshooter/SKILL.md" ]]; then
            echo "Verified k8s-troubleshooter skill structure"
        else
            echo "Warning: k8s-troubleshooter skill structure not as expected"
        fi
    else
        echo "Error: Failed to clone k8s4agents repository"
        exit 1
    fi
}

#######################################
# Setup agent-specific skill paths and session directories
# Globals:
#   AGENT_CLI
#   SESSION_DIR
# Arguments:
#   None
#######################################
setup_agent_paths() {
    echo "Setting up paths for agent: $AGENT_CLI"

    case "$AGENT_CLI" in
        claude)
            mkdir -p ~/.claude
            ln -sf /home/agent/skills/k8s4agents/skills ~/.claude/skills
            SESSION_DIR=~/.claude
            echo "Claude: Created symlink ~/.claude/skills -> /home/agent/skills/k8s4agents/skills"
            ;;
        codex)
            mkdir -p ~/.codex
            ln -sf /home/agent/skills/k8s4agents/skills ~/.codex/skills
            SESSION_DIR=~/.codex
            echo "Codex: Created symlink ~/.codex/skills -> /home/agent/skills/k8s4agents/skills"

            # Codex requires login with API key (must pipe via stdin)
            if [[ -n "${OPENAI_API_KEY:-}" ]]; then
                echo "Codex: Logging in with API key..."
                printenv OPENAI_API_KEY | codex login --with-api-key || {
                    echo "Warning: Codex login failed, continuing anyway"
                }
            fi
            ;;
        gemini)
            # Gemini uses GEMINI.md context file, should already exist in /home/agent
            SESSION_DIR=~/.gemini
            if [[ -f /home/agent/GEMINI.md ]]; then
                echo "Gemini: Found GEMINI.md context file"
            else
                echo "Warning: GEMINI.md not found, agent may not load context properly"
            fi
            ;;
        goose)
            # Goose searches multiple skill locations - create symlinks for both
            mkdir -p ~/.config/agents
            ln -sf /home/agent/skills/k8s4agents/skills ~/.config/agents/skills
            mkdir -p ~/.config/goose
            ln -sf /home/agent/skills/k8s4agents/skills ~/.config/goose/skills

            # Disable keyring to prevent interactive prompts
            export GOOSE_DISABLE_KEYRING=1

            SESSION_DIR=~/.config/goose
            echo "Goose: Created symlinks:"
            echo "  ~/.config/agents/skills -> /home/agent/skills/k8s4agents/skills"
            echo "  ~/.config/goose/skills -> /home/agent/skills/k8s4agents/skills"
            echo "Goose: Disabled keyring (GOOSE_DISABLE_KEYRING=1)"
            ;;
        *)
            echo "Error: Unknown AGENT_CLI: $AGENT_CLI"
            exit 1
            ;;
    esac

    export SESSION_DIR
    echo "Session directory: $SESSION_DIR"
}

#######################################
# Build the complete triage prompt
# This is the unified prompt passed to all agents as the last positional argument
# Combines:
#   1. Base triage prompt (from mounted ConfigMap)
#   2. Incident data (JSON)
#   3. Cluster permissions (JSON)
# Globals:
#   None
# Arguments:
#   None
# Outputs:
#   Complete triage prompt to stdout
#######################################
build_triage_prompt() {
    local prompt=""

    # 1. Base triage prompt (from mounted ConfigMap)
    if [[ -f /home/agent/base-triage-prompt.md ]]; then
        prompt+="$(cat /home/agent/base-triage-prompt.md)"
        prompt+=$'\n\n'
    fi

    # 2. Incident Context (from mounted ConfigMap)
    if [[ -f /home/agent/incident.json ]]; then
        prompt+="## Incident Data"$'\n\n'
        prompt+="<incident>"$'\n'
        prompt+="$(cat /home/agent/incident.json)"
        prompt+=$'\n'"</incident>"$'\n\n'
    fi

    # 3. Cluster Permissions (from mounted ConfigMap)
    if [[ -f /home/agent/incident_cluster_permissions.json ]]; then
        prompt+="## Cluster Access Permissions"$'\n\n'
        prompt+="<kubernetes_cluster_access_permissions>"$'\n'
        prompt+="$(cat /home/agent/incident_cluster_permissions.json)"
        prompt+=$'\n'"</kubernetes_cluster_access_permissions>"$'\n'
    fi

    echo "$prompt"
}

#######################################
# Capture the exact prompt sent to the agent
# This is the source of truth for what was actually sent
# Globals:
#   AGENT_CLI
#   LLM_MODEL
#   INCIDENT_ID
# Arguments:
#   $1 - The complete prompt text
# Outputs:
#   Creates /tmp/prompt-sent.md with metadata and full prompt
#######################################
capture_prompt_sent() {
    local prompt_text="$1"
    local output_file="/tmp/prompt-sent.md"
    local timestamp
    timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    cat > "$output_file" <<EOF
# Prompt Sent to Agent

## Metadata
- Timestamp: ${timestamp}
- Incident ID: ${INCIDENT_ID}
- Agent CLI: ${AGENT_CLI}
- Model: ${LLM_MODEL}
- Execution Mode: Kubernetes Job

## Complete Prompt

${prompt_text}
EOF

    echo "Captured actual prompt to ${output_file}"
}

#######################################
# Run the AI agent with real-time logging
# Captures stdout/stderr to both console and log file using tee
# Globals:
#   AGENT_CLI
#   LLM_MODEL
#   EXIT_CODE
#   AGENT_VERBOSE
#   DEBUG
# Arguments:
#   None
#######################################
run_agent() {
    local log_file="/home/agent/logs/agent.log"
    mkdir -p /home/agent/logs

    echo "Starting agent: $AGENT_CLI with model: $LLM_MODEL"
    echo "Incident ID: $INCIDENT_ID"
    echo "Log file: $log_file"
    echo "=========================================="

    # Build the complete triage prompt (same for all agents)
    local triage_prompt
    triage_prompt=$(build_triage_prompt)

    # Capture the actual prompt being sent for audit/debugging
    # This MUST match exactly what gets sent to the agent
    capture_prompt_sent "$triage_prompt"

    # Determine if verbose mode is enabled (Claude only)
    local verbose_flag=""
    if [[ "${AGENT_VERBOSE:-false}" == "true" || "${DEBUG:-false}" == "true" ]]; then
        verbose_flag="--verbose"
    fi

    # Run agent with tee for real-time output + file capture
    # Capture exit code for result.json
    case "$AGENT_CLI" in
        claude)
            # Claude: -p flag (print mode), prompt is last positional
            claude -p \
              --model "$LLM_MODEL" \
              --allowedTools "Read,Grep,Glob,Bash,Write" \
              --max-turns 50 $verbose_flag "$triage_prompt" \
              2>&1 | tee "$log_file" || EXIT_CODE=$?
            ;;
        codex)
            # Codex: no json output option, prompt is last positional
            codex exec \
                --skip-git-repo-check \
                --enable skills \
                --dangerously-bypass-approvals-and-sandbox \
                -m "$LLM_MODEL" \
                "$triage_prompt" \
                2>&1 | tee "$log_file" || EXIT_CODE=$?
            ;;
        gemini)
            # Gemini: --yolo for auto-approval, positional prompt
            gemini \
                --model "$LLM_MODEL" \
                --yolo \
                "$triage_prompt" \
                2>&1 | tee "$log_file" || EXIT_CODE=$?
            ;;
        goose)
            # Goose: use 'goose run' with --text for headless mode
            #
            # TODO: make sure we can pass in an llm provider using the nightcrier config; until that is done this won't work. however, have verified that this is the correct arguments and structure to do it.
            goose run \
                --model "$LLM_MODEL" \
                --provider "$LLM_PROVIDER" \
                --text "$triage_prompt" \
                2>&1 | tee "$log_file" || EXIT_CODE=$?
            ;;
        *)
            echo "Error: Unknown AGENT_CLI: $AGENT_CLI" | tee "$log_file"
            EXIT_CODE=1
            ;;
    esac

    echo "=========================================="
    echo "Agent completed with exit code: $EXIT_CODE"
}

#######################################
# Extract commands executed by the agent from session data
# Each agent stores session data in a different format:
# - Claude: JSONL in ~/.claude/projects/*/
# - Codex: JSONL in ~/.codex/sessions/
# - Gemini: JSON in ~/.gemini/tmp/*/chats/session-*.json
# - Goose: SQLite in ~/.config/goose/sessions.db
# Globals:
#   AGENT_CLI
#   SESSION_DIR
# Arguments:
#   None
# Outputs:
#   Creates /tmp/commands-executed.log with $ prefixed commands
#######################################
extract_commands() {
    local commands_file="/tmp/commands-executed.log"

    echo "Extracting commands executed by $AGENT_CLI..."

    case "$AGENT_CLI" in
        claude)
            # Find most recent JSONL file in projects directory
            local jsonl
            jsonl=$(find ~/.claude/projects -name "*.jsonl" -type f -print0 2>/dev/null | xargs -0 ls -t 2>/dev/null | head -1)

            if [[ -f "$jsonl" ]]; then
                echo "Found Claude session: $jsonl"
                jq -r 'select(.type == "assistant") |
                    .message.content[]? |
                    select(.type == "tool_use" and .name == "Bash") |
                    "$ " + .input.command' "$jsonl" > "$commands_file" 2>/dev/null || true
            else
                echo "No Claude session data found"
                touch "$commands_file"
            fi
            ;;
        codex)
            # Codex logs command executions directly in its output as "exec" lines
            # Extract these from the agent.log file instead of parsing JSONL
            echo "Extracting Codex commands from agent.log..."
            local log_file="/home/agent/logs/agent.log"

            if [[ -f "$log_file" ]]; then
                echo "Found agent log: $log_file ($(wc -c < "$log_file") bytes)"

                # Extract lines that start with "exec" (Codex command execution format)
                # Format: exec <command> in <workdir> succeeded/failed in <time>
                # We want to capture the command part
                grep -E "^exec$" "$log_file" -A 1 2>/dev/null | grep -v "^exec$" | grep -v "^--$" | while read -r line; do
                    # Clean up the line and prefix with $
                    echo "$ $line"
                done > "$commands_file" 2>/dev/null || touch "$commands_file"

                # If that didn't work, try alternative pattern
                if [[ ! -s "$commands_file" ]]; then
                    echo "DEBUG: Trying alternative extraction from agent output..."
                    # Look for lines that contain command execution patterns
                    grep -E "bash -lc|kubectl |echo " "$log_file" 2>/dev/null | head -20 | while read -r line; do
                        # Extract just the command part
                        if [[ "$line" =~ bash\ -lc\ \'(.+)\' ]]; then
                            echo "$ ${BASH_REMATCH[1]}"
                        elif [[ "$line" =~ (kubectl[^\'\"]+) ]]; then
                            echo "$ ${BASH_REMATCH[1]}"
                        fi
                    done > "$commands_file" 2>/dev/null || touch "$commands_file"
                fi

                if [[ -s "$commands_file" ]]; then
                    local cmd_count
                    cmd_count=$(wc -l < "$commands_file" | tr -d ' ')
                    echo "SUCCESS: Extracted $cmd_count commands from agent.log"
                else
                    echo "WARN: No commands extracted from agent.log"
                    echo "DEBUG: Showing sample lines from log:"
                    head -50 "$log_file" | tail -20
                fi
            else
                echo "ERROR: Agent log file not found at $log_file"
                touch "$commands_file"
            fi
            ;;
        gemini)
            # Find most recent session JSON file
            local json
            json=$(find ~/.gemini/tmp -path "*/chats/session-*.json" -type f -print0 2>/dev/null | xargs -0 ls -t 2>/dev/null | head -1)

            if [[ -f "$json" ]]; then
                echo "Found Gemini session: $json"
                jq -r '.[] | select(.type == "tool_use" and .tool_name == "bash") |
                    "$ " + .tool_input.command' "$json" > "$commands_file" 2>/dev/null || true
            else
                echo "No Gemini session data found"
                touch "$commands_file"
            fi
            ;;
        goose)
            # Query SQLite database for shell commands
            if [[ -f ~/.config/goose/sessions.db ]]; then
                echo "Found Goose session database"
                sqlite3 ~/.config/goose/sessions.db \
                    "SELECT '$ ' || json_extract(content, '$.command')
                     FROM messages
                     WHERE role = 'tool' AND json_extract(content, '$.command') IS NOT NULL
                     ORDER BY created_at" > "$commands_file" 2>/dev/null || {
                    echo "Failed to extract from Goose database, creating empty file"
                    touch "$commands_file"
                }
            else
                echo "No Goose session database found"
                touch "$commands_file"
            fi
            ;;
        *)
            echo "Unknown agent type, creating empty commands file"
            touch "$commands_file"
            ;;
    esac

    if [[ -s "$commands_file" ]]; then
        local cmd_count
        cmd_count=$(wc -l < "$commands_file" | tr -d ' ')
        echo "Extracted $cmd_count commands"
    else
        echo "No commands extracted"
    fi
}

#######################################
# Teardown function called on container exit
# Performs command extraction, session archiving, and artifact upload
# Globals:
#   EXIT_CODE
#   SESSION_DIR
#   OUTPUT_URL_*
# Arguments:
#   None
#######################################
teardown() {
    echo ""
    echo "=========================================="
    echo "Starting teardown and artifact upload..."
    echo "=========================================="

    # Extract commands from agent session before archiving
    extract_commands

    # Archive session directory if it exists
    if [[ -n "$SESSION_DIR" && -d "$SESSION_DIR" ]]; then
        echo "Archiving session directory: $SESSION_DIR"
        tar -czf /tmp/session.tar.gz -C "$SESSION_DIR" . 2>/dev/null || {
            echo "Warning: Failed to create session archive"
        }
    fi

    # Create result.json with exit code
    cat > /tmp/result.json <<EOF
{
  "exit_code": $EXIT_CODE,
  "incident_id": "$INCIDENT_ID",
  "agent_cli": "$AGENT_CLI",
  "model": "$LLM_MODEL",
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
}
EOF

    echo ""
    echo "Uploading artifacts to object storage..."

    # Upload report.md if it exists
    if [[ -f /home/agent/output/report.md ]]; then
        echo "Uploading report.md..."
        curl -X PUT -H "x-ms-blob-type: BlockBlob" -T /home/agent/output/report.md "$OUTPUT_URL_REPORT" || {
            echo "Warning: Failed to upload report.md"
        }
    else
        echo "Warning: report.md not found, skipping upload"
    fi

    # Upload agent.log
    if [[ -f /home/agent/logs/agent.log ]]; then
        echo "Uploading agent.log..."
        curl -X PUT -H "x-ms-blob-type: BlockBlob" -T /home/agent/logs/agent.log "$OUTPUT_URL_LOG" || {
            echo "Warning: Failed to upload agent.log"
        }
    else
        echo "Warning: agent.log not found, skipping upload"
    fi

    # Upload session.tar.gz if it exists
    if [[ -f /tmp/session.tar.gz ]]; then
        echo "Uploading session.tar.gz..."
        curl -X PUT -H "x-ms-blob-type: BlockBlob" -T /tmp/session.tar.gz "$OUTPUT_URL_SESSION" || {
            echo "Warning: Failed to upload session.tar.gz"
        }
    else
        echo "Warning: session.tar.gz not found, skipping upload"
    fi

    # Upload commands-executed.log if it exists and has content
    if [[ -s /tmp/commands-executed.log ]]; then
        echo "Uploading commands-executed.log..."
        curl -X PUT -H "x-ms-blob-type: BlockBlob" -T /tmp/commands-executed.log "$OUTPUT_URL_COMMANDS" || {
            echo "Warning: Failed to upload commands-executed.log"
        }
    else
        echo "No commands to upload (empty or missing commands-executed.log)"
    fi

    # Upload result.json
    echo "Uploading result.json..."
    curl -X PUT -H "x-ms-blob-type: BlockBlob" -T /tmp/result.json "$OUTPUT_URL_RESULT" || {
        echo "Warning: Failed to upload result.json"
    }

    # Upload prompt-sent.md (source of truth for what was sent to agent)
    if [[ -f /tmp/prompt-sent.md ]]; then
        echo "Uploading prompt-sent.md..."
        curl -X PUT -H "x-ms-blob-type: BlockBlob" -T /tmp/prompt-sent.md "$OUTPUT_URL_PROMPT_SENT" || {
            echo "Warning: Failed to upload prompt-sent.md"
        }
    else
        echo "Warning: prompt-sent.md not found, skipping upload"
    fi

    echo ""
    echo "=========================================="
    echo "Teardown complete"
    echo "Final exit code: $EXIT_CODE"
    echo "=========================================="
}

#######################################
# Main execution flow
#######################################
main() {
    echo "=========================================="
    echo "nc-agent-runner starting"
    echo "=========================================="
    echo "Incident ID: $INCIDENT_ID"
    echo "Agent: $AGENT_CLI"
    echo "Model: $LLM_MODEL"
    echo ""

    # Register teardown to run on exit (handles normal exit, SIGTERM, etc.)
    trap teardown EXIT

    # Clone k8s4agents skill repository if needed
    clone_skills

    echo ""

    # Setup agent-specific paths and configuration
    setup_agent_paths

    echo ""

    # Run the agent with real-time logging
    run_agent

    # Note: teardown() will be called automatically by trap on exit
}

# Start main execution
main
