# Contributing to Nightcrier

Thank you for your interest in contributing to Nightcrier!

## Getting Started

1. Fork the repository on GitHub
2. Clone your fork locally
3. Set up your development environment (see [dev_setup.md](dev_setup.md))
4. Create a feature branch from `main`

## Development Workflow

1. Check existing issues or create a new one
2. Discuss major changes in an issue first
3. Implement changes with tests
4. Submit a pull request

## Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat(scope): add new feature
fix(scope): fix bug description
docs(scope): update documentation
refactor(scope): refactor code
test(scope): add tests
```

## Code Style

- Follow [Effective Go](https://go.dev/doc/effective_go)
- Use `gofmt` for formatting
- Use `golangci-lint` for linting

## Testing

```bash
# Run all tests
make test

# Run with coverage
make test-coverage
```

## Pull Request Guidelines

- Keep PRs focused and reasonably sized
- Include tests for new functionality
- Update documentation as needed
- Reference related issues

## Questions?

Open an issue or discussion on GitHub.
