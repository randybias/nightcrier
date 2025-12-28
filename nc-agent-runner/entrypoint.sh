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
# Build the system prompt with incident context
# This is passed via --append-system-prompt, not -p
# Globals:
#   None
# Arguments:
#   None
# Outputs:
#   Combined system prompt to stdout
#######################################
build_system_prompt_with_context() {
    local system_content=""

    # 1. Read system prompt
    if [[ -f /home/agent/system-prompt.md ]]; then
        system_content=$(cat /home/agent/system-prompt.md)
        system_content+="\n\n"
    fi

    # 2. Add incident context
    if [[ -f /home/agent/incident.json ]]; then
        system_content+="<incident>\n"
        system_content+=$(cat /home/agent/incident.json)
        system_content+="\n</incident>\n\n"
    fi

    # 3. Add permissions context
    if [[ -f /home/agent/incident_cluster_permissions.json ]]; then
        system_content+="<kubernetes_cluster_access_permissions>\n"
        system_content+=$(cat /home/agent/incident_cluster_permissions.json)
        system_content+="\n</kubernetes_cluster_access_permissions>\n\n"
    fi

    echo -e "$system_content"
}

#######################################
# Run the AI agent with real-time logging
# Captures stdout/stderr to both console and log file using tee
# Globals:
#   AGENT_CLI
#   PROMPT
#   LLM_MODEL
#   EXIT_CODE
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

    # Build the system prompt with incident context and save to file
    # Using a file avoids issues with long strings and special characters
    local system_prompt_file="/tmp/combined-system-prompt.md"
    build_system_prompt_with_context > "$system_prompt_file"

    # The investigation prompt (from PROMPT env var or empty for autonomous mode)
    local investigation_prompt="${PROMPT:-Investigate this incident and generate a report.}"

    # Run agent with tee for real-time output + file capture
    # Capture exit code for result.json
    case "$AGENT_CLI" in
        claude)
            echo "DEBUG: About to execute claude command"
            echo "DEBUG: log_file=$log_file"
            set -x
            claude -p "Investigate this incident and generate a report." --model claude-sonnet-4-5-20250929 --allowedTools Read,Grep,Glob,Bash,Write --append-system-prompt-file /tmp/combined-system-prompt.md --max-turns 50 2>&1 | tee "$log_file" || EXIT_CODE=$?
            set +x
            echo "DEBUG: Command completed with EXIT_CODE=$EXIT_CODE"
            ;;
        codex)
            # Combine system prompt with investigation prompt (like old agent-container setup)
            # Codex exec expects the full prompt as a positional argument
            echo "DEBUG: Building combined prompt"
            echo "DEBUG: System prompt file exists: $(test -f /tmp/combined-system-prompt.md && echo YES || echo NO)"
            echo "DEBUG: System prompt file size: $(wc -c < /tmp/combined-system-prompt.md 2>/dev/null || echo 0) bytes"
            echo "DEBUG: Investigation prompt: $investigation_prompt"

            local combined_prompt
            combined_prompt="$(cat /tmp/combined-system-prompt.md)

${investigation_prompt}"

            echo "DEBUG: Combined prompt length: ${#combined_prompt} characters"
            echo "DEBUG: Combined prompt first 200 chars: ${combined_prompt:0:200}"

            # Use same flags as old working setup
            codex exec --skip-git-repo-check --enable skills --dangerously-bypass-approvals-and-sandbox -m "$LLM_MODEL" "$combined_prompt" 2>&1 | tee "$log_file" || EXIT_CODE=$?
            ;;
        gemini)
            gemini -p "$investigation_prompt" --model "$LLM_MODEL" 2>&1 | tee "$log_file" || EXIT_CODE=$?
            ;;
        goose)
            goose session start --profile "$LLM_MODEL" --prompt "$investigation_prompt" 2>&1 | tee "$log_file" || EXIT_CODE=$?
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
            # Find most recent JSONL file in sessions directory
            local jsonl
            jsonl=$(find ~/.codex/sessions -name "*.jsonl" -type f -print0 2>/dev/null | xargs -0 ls -t 2>/dev/null | head -1)

            if [[ -f "$jsonl" ]]; then
                echo "Found Codex session: $jsonl"
                jq -r 'select(.type == "response_item" and .payload.type == "function_call" and .payload.name == "shell_command") |
                    .payload.arguments | fromjson | "$ " + .command' "$jsonl" > "$commands_file" 2>/dev/null || true
            else
                echo "No Codex session data found"
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
