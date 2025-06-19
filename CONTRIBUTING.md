# Contributing to NRDOT-HOST

Thank you for your interest in contributing to NRDOT-HOST, New Relic's next-generation Linux telemetry collector! This document provides guidelines for contributing to the project.

## Code of Conduct

This project adheres to a Code of Conduct. By participating, you are expected to uphold this code.

## How to Contribute

### Reporting Issues

- Check existing issues before creating a new one
- Use issue templates when available
- Include relevant information:
  - NRDOT-HOST version (`nrdot-host --version`)
  - Linux distribution and version
  - Steps to reproduce
  - Expected vs actual behavior
  - Logs (`journalctl -u nrdot-host`)

### Submitting Pull Requests

1. **Fork the Repository**
   ```bash
   git clone https://github.com/newrelic/nrdot-host.git
   cd nrdot-host
   git remote add upstream https://github.com/newrelic/nrdot-host.git
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

1. **Linux-Only Focus**
   - All new code must be Linux-specific
   - No Windows or macOS code will be accepted
   - Use build tags for Linux-specific files: `//go:build linux`
   - Remove cross-platform abstractions

2. **Go Code**
   - Follow [Effective Go](https://golang.org/doc/effective_go.html)
   - Use `gofmt` for formatting
   - Run `golangci-lint` before committing
   - Maintain 80%+ test coverage

3. **Documentation**
   - Document all exported functions and types
   - Include examples in documentation
   - Update relevant docs in `/docs` directory

4. **Testing**
   - Write unit tests for all new code
   - Use table-driven tests where appropriate
   - Mock external dependencies
   - Test on multiple Linux distributions

### Project Structure

```
component-name/
├── cmd/           # Main applications
├── pkg/           # Public packages
├── internal/      # Private packages
├── test/          # Integration tests
└── README.md      # Component documentation
```

### Building

```bash
# Build unified binary (Linux only)
cd cmd/nrdot-host
make build

# Build all components
make all

# Build for specific Linux architectures
make build-linux-amd64
make build-linux-arm64
```

**Note**: Cross-platform builds are being removed. Only Linux targets are supported.

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

## Development Areas

### Priority Areas for Contribution

1. **Linux-Only Optimization** (Phase 0)
   - Remove Windows/macOS code
   - Optimize for Linux syscalls
   - Improve /proc parsing efficiency

2. **Process Telemetry** (Phase 1)
   - Enhanced /proc metrics collection
   - Top-N process tracking
   - Service pattern detection

3. **Auto-Configuration** (Phase 2)
   - Service discovery modules
   - Configuration templates
   - Integration receivers

### OpenTelemetry Processors

When working on processors:
1. Focus on Linux-specific optimizations
2. Add comprehensive unit tests
3. Include benchmarks
4. Document configuration options

### Future CLI Commands

Planned commands for contribution:
- `nrdot-host discover` - Service discovery (Phase 2)
- `nrdot-host migrate-infra` - Migration tool (Phase 3)
- `nrdot-host validate` - Config validation

Guidelines:
1. Follow existing command patterns
2. Add comprehensive help text
3. Include examples
4. Test on multiple Linux distributions

## Release Process

1. Maintainers create release branches
2. Version bumps follow semantic versioning
3. Changelog is automatically generated
4. Releases trigger CI/CD pipelines

## Getting Help

- Review the [Roadmap](docs/roadmap/ROADMAP.md)
- Check the [Architecture](docs/architecture/ARCHITECTURE.md)
- Ask questions in GitHub Issues
- See [Troubleshooting Guide](docs/troubleshooting.md)

## Recognition

Contributors are recognized in:
- Release notes
- Contributors file
- Project documentation

Thank you for contributing to NRDOT-HOST!