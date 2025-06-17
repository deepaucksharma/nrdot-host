# nrdot-supervisor

The nrdot-supervisor component manages the OpenTelemetry Collector lifecycle, providing process management, health monitoring, and automatic restart capabilities.

## Features

- **Process Lifecycle Management**: Start, stop, and restart the OTel Collector process
- **Health Monitoring**: Continuous health checks with configurable intervals and thresholds
- **Restart Strategies**: Support for never, on-failure, and always restart policies with exponential backoff
- **Memory Monitoring**: Track and enforce memory limits with automatic restart on breach
- **Signal Handling**: Graceful shutdown and configuration reload support
- **Telemetry Integration**: Reports health metrics using nrdot-telemetry-client
- **Log Streaming**: Captures and logs collector stdout/stderr output

## Installation

```bash
go install github.com/newrelic/nrdot-host/nrdot-supervisor/cmd/supervisor
```

## Usage

### Basic Usage

```bash
nrdot-supervisor --collector-binary /path/to/otelcol --collector-config /etc/otel/config.yaml
```

### Configuration Options

#### Collector Options
- `--collector-binary`: Path to OTel collector binary (default: "otelcol")
- `--collector-config`: Path to collector config file (default: "/etc/otel/config.yaml")
- `--collector-args`: Additional collector arguments
- `--collector-workdir`: Collector working directory
- `--memory-limit`: Collector memory limit in bytes (default: 536870912)

#### Health Check Options
- `--health-endpoint`: Collector health endpoint (default: "http://localhost:13133/health")
- `--health-interval`: Health check interval (default: 10s)
- `--health-timeout`: Health check timeout (default: 5s)
- `--health-threshold`: Consecutive failures before restart (default: 3)

#### Restart Options
- `--restart-policy`: Restart policy: never, on-failure, always (default: "on-failure")
- `--restart-max-retries`: Maximum restart attempts (default: 10)
- `--restart-initial-delay`: Initial restart delay (default: 1s)
- `--restart-max-delay`: Maximum restart delay (default: 5m)
- `--restart-backoff`: Restart backoff multiplier (default: 2.0)

#### Telemetry Options
- `--telemetry-endpoint`: Telemetry endpoint (default: "http://localhost:4318/v1/metrics")
- `--telemetry-interval`: Telemetry reporting interval (default: 10s)

#### General Options
- `--log-level`: Log level: debug, info, warn, error (default: "info")
- `--log-json`: Output logs in JSON format

### Examples

#### Run with custom memory limit and restart policy
```bash
nrdot-supervisor \
  --collector-binary /usr/local/bin/otelcol \
  --collector-config /etc/otel/config.yaml \
  --memory-limit 1073741824 \
  --restart-policy always \
  --restart-max-retries 20
```

#### Run with debug logging and health check configuration
```bash
nrdot-supervisor \
  --log-level debug \
  --health-interval 5s \
  --health-timeout 2s \
  --health-threshold 5
```

#### Run with telemetry endpoint
```bash
nrdot-supervisor \
  --telemetry-endpoint http://telemetry.example.com:4318/v1/metrics \
  --telemetry-interval 30s
```

## Signals

The supervisor responds to the following signals:

- **SIGTERM/SIGINT**: Initiates graceful shutdown
- **SIGHUP**: Forwards to collector for configuration reload

## Metrics

The supervisor reports the following metrics via telemetry-client:

- `supervisor.start.failed`: Supervisor startup failures
- `supervisor.start.success`: Successful supervisor starts
- `supervisor.collector.start.failed`: Collector startup failures
- `supervisor.collector.start.success`: Successful collector starts
- `supervisor.collector.start.duration`: Time taken to start collector
- `supervisor.collector.unexpected_exit`: Unexpected collector exits
- `supervisor.collector.failure`: Collector failures with reason tag
- `supervisor.collector.memory.usage`: Current memory usage in bytes
- `supervisor.collector.memory.limit_exceeded`: Memory limit breach events
- `supervisor.health_check.failed`: Health check failures
- `supervisor.restart.exhausted`: Restart strategy exhaustion
- `supervisor.restart.failed`: Failed restart attempts
- `supervisor.restart.success`: Successful restarts
- `supervisor.reload.requested`: Configuration reload requests
- `supervisor.reload.success`: Successful reloads
- `supervisor.reload.failed`: Failed reloads

## Development

### Building

```bash
make build
```

### Testing

```bash
make test
```

### Running locally

```bash
make run
```

## Architecture

The supervisor consists of several key components:

1. **CollectorProcess**: Manages the OTel Collector process lifecycle
2. **HealthChecker**: Monitors collector health via HTTP endpoint
3. **Restart Strategies**: Implements various restart policies
4. **Supervisor**: Orchestrates all components and handles signals

The supervisor runs a main loop that:
- Monitors collector process status
- Performs periodic health checks
- Checks memory usage
- Handles OS signals
- Manages restarts according to policy