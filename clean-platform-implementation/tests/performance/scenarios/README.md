# Performance Test Scenarios

This directory contains specific performance test scenarios for the platform.

## Test Types

### 1. Smoke Test
- **Purpose**: Quick validation that the system is functional
- **Duration**: 2 minutes
- **Load**: 10 concurrent users
- **Use Case**: Run after every deployment

### 2. Load Test
- **Purpose**: Verify system performance under normal expected load
- **Duration**: 10 minutes
- **Load**: 100 concurrent users
- **Use Case**: Daily validation

### 3. Stress Test
- **Purpose**: Find the breaking point of the system
- **Duration**: 5 minutes
- **Load**: 500+ concurrent users
- **Use Case**: Capacity planning

### 4. Spike Test
- **Purpose**: Test system behavior under sudden load increase
- **Duration**: 3 minutes
- **Load**: Spike from 100 to 1000 users
- **Use Case**: Black Friday, flash sales

### 5. Soak Test
- **Purpose**: Identify memory leaks and degradation over time
- **Duration**: 60 minutes
- **Load**: 200 concurrent users (steady)
- **Use Case**: Weekly validation

### 6. Breakpoint Test
- **Purpose**: Gradually increase load until system breaks
- **Duration**: Variable
- **Load**: Step increase of 10 users every 30 seconds
- **Use Case**: Capacity planning

## Running Tests

### Local Testing
```bash
# Run smoke test locally
./run-performance-tests.sh smoke local

# Run with custom Locust parameters
locust -f ../locustfile.py --host http://localhost:8080
```

### CI/CD Integration
```yaml
# GitHub Actions example
- name: Run Performance Tests
  run: |
    ./tests/performance/run-performance-tests.sh smoke staging
  continue-on-error: true
```

### Scheduled Tests
```bash
# Cron job for nightly load tests
0 2 * * * cd /path/to/tests/performance && ./run-performance-tests.sh load prod
```

## Performance Baselines

### API Gateway
- P50: < 50ms
- P90: < 200ms
- P95: < 300ms
- P99: < 500ms

### Data Collection Endpoint
- Throughput: 10,000 requests/second
- P95 Latency: < 100ms
- Error Rate: < 0.1%

### Data Processing
- Queue Time P95: < 5 seconds
- Processing Time P95: < 30 seconds
- Success Rate: > 99.9%

### Query Endpoint
- Simple Queries P95: < 200ms
- Complex Queries P95: < 2 seconds
- Concurrent Queries: 1000+

## SLA Targets

| Metric | Target | Critical |
|--------|--------|----------|
| Availability | 99.9% | 99.5% |
| P95 Response Time | < 500ms | < 1000ms |
| Error Rate | < 1% | < 5% |
| Throughput | > 5000 RPS | > 1000 RPS |

## Monitoring During Tests

### Grafana Dashboards
- Platform Performance Dashboard
- Infrastructure Metrics
- Application Metrics

### Key Metrics to Watch
1. **Application Metrics**
   - Request rate
   - Response times
   - Error rates
   - Queue depths

2. **Infrastructure Metrics**
   - CPU utilization
   - Memory usage
   - Network I/O
   - Disk I/O

3. **Database Metrics**
   - Connection pool usage
   - Query execution time
   - Lock waits
   - Replication lag

## Troubleshooting

### High Response Times
1. Check database query performance
2. Verify cache hit rates
3. Look for CPU throttling
4. Check network latency

### High Error Rates
1. Check application logs
2. Verify resource limits
3. Look for rate limiting
4. Check downstream dependencies

### Memory Issues
1. Look for memory leaks
2. Check garbage collection
3. Verify heap settings
4. Review object allocation

## Best Practices

1. **Warm up** the system before tests
2. **Reset** test data between runs
3. **Monitor** all components during tests
4. **Document** any configuration changes
5. **Compare** results with baselines
6. **Investigate** any anomalies

## Reporting

Test results are automatically:
- Saved to `results/` directory
- Uploaded to S3 (if configured)
- Annotated in Grafana
- Included in CI/CD artifacts

## Custom Scenarios

Create custom test scenarios by:
1. Extending the base `PlatformUser` class
2. Defining specific user behaviors
3. Setting appropriate wait times
4. Adding to `locustfile.py`

Example:
```python
class APIHeavyUser(PlatformUser):
    """User that makes heavy API calls"""
    wait_time = constant(0.5)
    
    @task(10)
    def heavy_query(self):
        # Custom heavy query implementation
        pass
```