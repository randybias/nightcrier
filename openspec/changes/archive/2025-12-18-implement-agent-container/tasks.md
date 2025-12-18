# Implementation Tasks: Agent Container

All tasks are **COMPLETE**.

## 1. Docker Container Build

- [x] 1.1 Create Dockerfile with debian:bookworm-slim base
- [x] 1.2 Install core utilities (curl, wget, git, jq, yq)
- [x] 1.3 Install search tools (ripgrep, fd, fzf, tree, bat)
- [x] 1.4 Install network debugging tools (dnsutils, netcat, ping, iproute2)
- [x] 1.5 Install Node.js 20 LTS for npm-based CLIs
- [x] 1.6 Install kubectl 1.31
- [x] 1.7 Install helm 3.x
- [x] 1.8 Install GitHub CLI
- [x] 1.9 Create symlinks for Debian tool names (fdfind→fd, batcat→bat)

## 2. AI CLI Installation

- [x] 2.1 Install Claude Code CLI (@anthropic-ai/claude-code)
- [x] 2.2 Install OpenAI Codex CLI (@openai/codex)
- [x] 2.3 Install Google Gemini CLI (@google/gemini-cli)
- [x] 2.4 Document Goose as disabled (X11 dependency)

## 3. Skills Integration

- [x] 3.1 Clone k8s4agents from GitHub during build
- [x] 3.2 Copy skills to /skills/ directory
- [x] 3.3 Create Claude slash command for k8s-troubleshooter
- [x] 3.4 Verify skill files accessible in container

## 4. Run Script (run-agent.sh)

- [x] 4.1 Implement argument parsing with getopts
- [x] 4.2 Add agent selection flag (-a/--agent)
- [x] 4.3 Add required workspace flag (-w/--workspace)
- [x] 4.4 Add Claude-specific options (-m model, -t tools, -s system-prompt)
- [x] 4.5 Add container options (--timeout, --memory, --network)
- [x] 4.6 Implement API key validation per agent
- [x] 4.7 Build Docker run command with volume mounts
- [x] 4.8 Implement agent-specific CLI invocation:
  - [x] Claude: `-p` flag with model and tools
  - [x] Codex: login step + `exec --skip-git-repo-check`
  - [x] Gemini: `-p` flag
- [x] 4.9 Implement output capture with tee
- [x] 4.10 Add debug mode (-d) for command inspection
- [x] 4.11 Add help text (--help)

## 5. Workspace Isolation

- [x] 5.1 Remove default workspace (was pwd, now required)
- [x] 5.2 Add validation error for missing workspace
- [x] 5.3 Mount workspace to /workspace in container
- [x] 5.4 Mount output directory to /output
- [x] 5.5 Mount kubeconfig read-only

## 6. Output Capture

- [x] 6.1 Create output directory if not exists
- [x] 6.2 Generate timestamped output filename
- [x] 6.3 Pipe agent output through tee
- [x] 6.4 Report output path on completion

## 7. Makefile

- [x] 7.1 Add build target
- [x] 7.2 Add build-clean target (no cache)
- [x] 7.3 Add test-claude target
- [x] 7.4 Add test-codex target
- [x] 7.5 Add test-gemini target
- [x] 7.6 Add test-tools target (verify installations)
- [x] 7.7 Add test-kubectl target
- [x] 7.8 Add test-workspace target (create isolated test dir)
- [x] 7.9 Add shell target (interactive debugging)
- [x] 7.10 Add info and clean targets

## 8. Documentation

- [x] 8.1 Write README.md with quick start
- [x] 8.2 Document supported agents table
- [x] 8.3 Document architecture diagram
- [x] 8.4 Document configuration options
- [x] 8.5 Document API key requirements
- [x] 8.6 Add troubleshooting section (Codex auth, Goose X11)
- [x] 8.7 Document integration with event runner

## 9. Testing

- [x] 9.1 Build container successfully
- [x] 9.2 Verify all tools installed (make test-tools)
- [x] 9.3 Test Claude invocation with isolated workspace
- [x] 9.4 Test Codex invocation (auth fix verified)
- [x] 9.5 Test Gemini invocation
- [x] 9.6 Verify skills accessible at /skills/
- [x] 9.7 Verify output capture to log file

## 10. OpenSpec Updates

- [x] 10.1 Update implement-agent-runtime design.md with container reference
- [x] 10.2 Update implement-agent-runtime tasks.md with completed items
- [x] 10.3 Mark multi-agent support (21.2) as complete
