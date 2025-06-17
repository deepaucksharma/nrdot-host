# Contributing to NRDOT-HOST

Thank you for your interest in contributing to NRDOT-HOST! This document provides guidelines and instructions for contributing.

## Code of Conduct

This project adheres to a Code of Conduct. By participating, you are expected to uphold this code.

## How to Contribute

### Reporting Issues

- Check existing issues before creating a new one
- Use issue templates when available
- Include relevant information:
  - NRDOT version
  - Operating system
  - Steps to reproduce
  - Expected vs actual behavior
  - Logs and error messages

### Submitting Pull Requests

1. **Fork the Repository**
   ```bash
   git clone https://github.com/deepaucksharma/nrdot-host.git
   cd nrdot-host
   git remote add upstream https://github.com/deepaucksharma/nrdot-host.git
   ```

2. **Create a Branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

3. **Make Changes**
   - Follow the coding standards below
   - Add tests for new functionality
   - Update documentation as needed

4. **Test Your Changes**
   ```bash
   make test
   make lint
   make test-integration  # if applicable
   ```

5. **Commit Your Changes**
   ```bash
   git add .
   git commit -m "feat: add new feature"
   ```
   
   Follow [Conventional Commits](https://www.conventionalcommits.org/):
   - `feat:` new feature
   - `fix:` bug fix
   - `docs:` documentation changes
   - `test:` test additions/changes
   - `refactor:` code refactoring
   - `chore:` maintenance tasks

6. **Push and Create PR**
   ```bash
   git push origin feature/your-feature-name
   ```

## Development Guidelines

### Code Standards

1. **Go Code**
   - Follow [Effective Go](https://golang.org/doc/effective_go.html)
   - Use `gofmt` for formatting
   - Run `golangci-lint` before committing
   - Maintain 80%+ test coverage

2. **Documentation**
   - Document all exported functions and types
   - Include examples in documentation
   - Update README.md for user-facing changes

3. **Testing**
   - Write unit tests for all new code
   - Use table-driven tests where appropriate
   - Mock external dependencies
   - Test error cases

### Project Structure

```
component-name/
├── cmd/           # Main applications
├── pkg/           # Public packages
├── internal/      # Private packages
├── test/          # Integration tests
└── README.md      # Component documentation
```

### Building Components

```bash
# Build a specific component
cd nrdot-ctl
go build -o bin/nrdot-ctl ./cmd/nrdot-ctl

# Build all components
make all

# Build with specific flags
make build-nrdot-ctl GOFLAGS="-v"
```

### Testing

```bash
# Unit tests
go test ./...

# With coverage
go test -cover ./...

# Integration tests
make test-integration

# E2E tests
make test-e2e
```

## Component-Specific Guidelines

### OpenTelemetry Processors

When creating or modifying processors:

1. Implement all required interfaces from `otel-processor-common`
2. Add comprehensive unit tests
3. Include benchmarks for performance-critical code
4. Document configuration options
5. Add integration tests in `e2e-tests`

### CLI Commands

For `nrdot-ctl` commands:

1. Use cobra for command structure
2. Provide helpful descriptions and examples
3. Support both flags and config files
4. Include completion scripts
5. Test interactive and non-interactive modes

## Release Process

1. Maintainers create release branches
2. Version bumps follow semantic versioning
3. Changelog is automatically generated
4. Releases trigger CI/CD pipelines

## Getting Help

- Join the [Discussions](https://github.com/deepaucksharma/nrdot-host/discussions)
- Check the [troubleshooting guide](./docs/troubleshooting.md)
- Ask questions in issues with the `question` label

## Recognition

Contributors are recognized in:
- Release notes
- Contributors file
- Project documentation

Thank you for contributing to NRDOT-HOST!