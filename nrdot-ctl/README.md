# nrdot-ctl

Command-line interface for managing NRDOT (New Relic Data on Tap) system.

## Installation

```bash
make build
make install
```

## Usage

### Show collector status
```bash
nrdot-ctl status
nrdot-ctl status --output json
```

### Configuration management
```bash
# Validate configuration
nrdot-ctl config validate -f config.yaml

# Generate OTel config from NRDOT config
nrdot-ctl config generate -f nrdot-config.yaml -o otel-config.yaml

# Apply configuration
nrdot-ctl config apply -f config.yaml
```

### Collector control
```bash
nrdot-ctl collector start
nrdot-ctl collector stop
nrdot-ctl collector restart
nrdot-ctl collector logs --follow
```

### View metrics
```bash
nrdot-ctl metrics
nrdot-ctl metrics --output json
```

### Version information
```bash
nrdot-ctl version
```

## Global Flags

- `--output`: Output format (table, json, yaml)
- `--api-endpoint`: API server endpoint (default: http://localhost:8080)
- `--config`: Config file path
- `--no-color`: Disable colored output
- `--verbose`: Enable verbose logging

## Environment Variables

- `NRDOT_API_ENDPOINT`: API server endpoint
- `NRDOT_CONFIG`: Config file path
- `NRDOT_OUTPUT`: Default output format

## Shell Completion

```bash
# Bash
nrdot-ctl completion bash > /etc/bash_completion.d/nrdot-ctl

# Zsh
nrdot-ctl completion zsh > "${fpath[1]}/_nrdot-ctl"

# Fish
nrdot-ctl completion fish > ~/.config/fish/completions/nrdot-ctl.fish
```

## Development

```bash
# Run tests
make test

# Format code
make fmt

# Run linter
make lint

# Generate completions
make completion
```
