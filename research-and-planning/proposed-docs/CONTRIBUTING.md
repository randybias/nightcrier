# Contributing to Nightcrier

Thank you for your interest in contributing to Nightcrier! This document provides guidelines and instructions for contributing.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Process](#development-process)
- [Coding Standards](#coding-standards)
- [Testing](#testing)
- [Submitting Changes](#submitting-changes)
- [Review Process](#review-process)

## Code of Conduct

### Our Pledge

We are committed to providing a welcoming and inspiring community for all. Please be respectful and constructive in your interactions.

### Expected Behavior

- Be respectful and inclusive
- Provide constructive feedback
- Focus on what is best for the community
- Show empathy towards other community members

### Unacceptable Behavior

- Harassment, discrimination, or derogatory comments
- Trolling or insulting comments
- Publishing others' private information
- Other conduct which could reasonably be considered inappropriate

## Getting Started

1. **Fork the repository** on GitHub
2. **Clone your fork** locally:
   ```bash
   git clone https://github.com/YOUR_USERNAME/nightcrier.git
   cd nightcrier
   ```

3. **Add upstream remote**:
   ```bash
   git remote add upstream https://github.com/rbias/nightcrier.git
   ```

4. **Create a feature branch**:
   ```bash
   git checkout -b feature/your-feature-name
   ```

5. **Set up development environment** (see [DEV_SETUP.md](DEV_SETUP.md))

## Development Process

### Workflow

1. **Check existing issues** - Look for existing issues or create a new one
2. **Discuss major changes** - For significant changes, discuss in an issue first
3. **Create feature branch** - Branch from `main`
4. **Implement changes** - Write code following our standards
5. **Add tests** - Ensure adequate test coverage
6. **Update documentation** - Update relevant docs
7. **Commit changes** - Use conventional commits
8. **Push to fork** - Push your branch to your fork
9. **Open Pull Request** - Submit PR with clear description

### Branch Naming

Use descriptive branch names:
- `feature/add-prometheus-metrics`
- `fix/handle-nil-pointer-in-executor`
- `docs/update-installation-guide`
- `refactor/simplify-storage-interface`

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `test`: Adding or updating tests
- `chore`: Maintenance tasks

**Examples:**
```
feat(executor): add K8s-native agent execution

Implement Kubernetes Job-based agent execution to replace
Docker containers. This enables stateless execution without
volume mounts.

Closes #123
```

```
fix(storage): handle nil pointer in artifact upload

Add nil check before accessing artifact metadata to prevent
panic when artifact is missing.

Fixes #456
```

## Coding Standards

### Go Code Style

- Follow [Effective Go](https://go.dev/doc/effective_go)
- Use `gofmt` for formatting
- Use `golangci-lint` for linting
- Keep functions small and focused
- Write clear, descriptive variable names
- Add comments for exported functions and types

### Code Organization

```
├── cmd/                    # Application entrypoints
│   └── nightcrier/        # Main application
├── internal/              # Private application code
│   ├── agent/            # Agent execution
│   ├── cluster/          # Multi-cluster management
│   ├── config/           # Configuration
│   ├── events/           # Event handling
│   ├── incident/         # Incident management
│   ├── reporting/        # Slack/notifications
│   ├── skills/           # Agent skills
│   └── storage/          # Storage backends
├── deploy/               # Deployment configs
│   └── dev/             # Local development
├── docs/                # Documentation
├── migrations/          # Database migrations
├── nc-agent-runner/     # Agent container
└── tests/              # Integration tests
```

### Error Handling

- Always handle errors explicitly
- Use descriptive error messages
- Wrap errors with context: `fmt.Errorf("failed to create job: %w", err)`
- Log errors with appropriate severity

### Logging

Use structured logging with `log/slog`:

```go
slog.Info("job created", "job_name", jobName, "namespace", namespace)
slog.Error("failed to create job", "error", err, "incident_id", incidentID)
```

## Testing

### Writing Tests

- Write unit tests for new functionality
- Use table-driven tests when appropriate
- Mock external dependencies
- Test error conditions

**Example:**
```go
func TestProcessEvent(t *testing.T) {
    tests := []struct {
        name    string
        event   *events.FaultEvent
        wantErr bool
    }{
        {
            name:    "valid event",
            event:   &events.FaultEvent{...},
            wantErr: false,
        },
        {
            name:    "nil event",
            event:   nil,
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ProcessEvent(tt.event)
            if (err != nil) != tt.wantErr {
                t.Errorf("ProcessEvent() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### Running Tests

```bash
# All tests
make test

# Specific package
go test ./internal/agent/...

# With coverage
make test-coverage

# Integration tests
make test-integration
```

### Test Coverage

- Aim for 80%+ coverage for new code
- Critical paths should have 100% coverage
- Don't sacrifice readability for coverage

## Submitting Changes

### Before Submitting

- [ ] Code builds successfully
- [ ] All tests pass
- [ ] Linting passes
- [ ] Documentation updated
- [ ] Commit messages follow conventions
- [ ] Branch is up to date with main

### Pull Request Guidelines

1. **Title**: Clear, descriptive title
2. **Description**:
   - What changes were made
   - Why the changes were necessary
   - How to test the changes
   - Link to related issues
3. **Screenshots**: If UI changes, include screenshots
4. **Breaking Changes**: Clearly mark breaking changes

**PR Template:**
```markdown
## Description
Brief description of changes

## Motivation
Why these changes are needed

## Changes
- Change 1
- Change 2
- Change 3

## Testing
How to test these changes

## Checklist
- [ ] Tests added/updated
- [ ] Documentation updated
- [ ] Commits follow conventions
- [ ] No breaking changes (or documented)

Closes #issue_number
```

### PR Size

- Keep PRs focused and reasonably sized
- Large changes should be split into multiple PRs
- If PR is too large, consider breaking it up

## Review Process

### What to Expect

1. **Automated checks** run (tests, linting)
2. **Maintainer review** within 3-5 business days
3. **Feedback and iteration** as needed
4. **Approval** from at least one maintainer
5. **Merge** once approved and checks pass

### Review Feedback

- Address all review comments
- Mark conversations as resolved when fixed
- Push new commits (don't force push)
- Re-request review when ready

### Addressing Feedback

```bash
# Make requested changes
git add .
git commit -m "fix: address review feedback"
git push origin feature/your-feature-name
```

## Development Guidelines

### Adding New Features

1. Open an issue to discuss the feature
2. Get maintainer approval before starting work
3. Implement with tests and documentation
4. Submit PR with clear description

### Fixing Bugs

1. Check if issue already exists
2. Create issue if it doesn't exist
3. Reference issue in commits and PR
4. Add regression test

### Improving Documentation

Documentation improvements are always welcome!

- Fix typos
- Clarify confusing sections
- Add examples
- Update outdated information

No need to open an issue for simple doc fixes.

## Project Structure

### Key Packages

- **`internal/agent`** - Agent execution (Docker and K8s)
- **`internal/cluster`** - Multi-cluster management
- **`internal/config`** - Configuration management
- **`internal/storage`** - Storage backends (filesystem, Azure Blob, S3)
- **`internal/incident`** - Incident management
- **`internal/reporting`** - Notifications (Slack)

### Adding Dependencies

1. **Minimize dependencies** - Only add if necessary
2. **Verify license** - Must be compatible (Apache 2.0, MIT, BSD)
3. **Update go.mod**:
   ```bash
   go get github.com/example/package@version
   go mod tidy
   ```

## Release Process

Releases are managed by maintainers:

1. Version bump in code
2. Update CHANGELOG.md
3. Create Git tag
4. Build and publish artifacts
5. Update documentation

## Getting Help

- **Questions**: Open a discussion on GitHub
- **Bugs**: Open an issue with reproduction steps
- **Features**: Open an issue with detailed description
- **Security**: Email security@example.com (do not open public issue)

## Recognition

Contributors are recognized in:
- CHANGELOG.md
- Release notes
- GitHub contributors page

Thank you for contributing to Nightcrier!
