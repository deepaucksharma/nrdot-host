# High-Performance Tuning Guide for NRDOT-HOST

This guide provides detailed instructions for optimizing NRDOT-HOST for maximum performance.

## Performance Goals

- **Throughput**: 1M+ events/second per instance
- **Latency**: <10ms p99 end-to-end
- **Memory**: <20GB under peak load
- **CPU**: Efficient utilization across all cores

## System Requirements

### Hardware Recommendations

**CPU**:
- Minimum: 8 cores (16 threads)
- Recommended: 16+ cores (32+ threads)
- Architecture: x86_64 with AVX2 support
- NUMA: Single node preferred

**Memory**:
- Minimum: 32GB RAM
- Recommended: 64GB+ RAM
- Type: DDR4-3200 or better
- Configuration: Dual channel or better

**Storage**:
- Type: NVMe SSD
- Capacity: 1TB+ for buffers and temporary storage
- IOPS: 100K+ random read/write
- Latency: <100μs

**Network**:
- Bandwidth: 10Gbps+
- NICs: SR-IOV capable
- Offloading: Enable all hardware offloads

### OS Tuning

#### Linux Kernel Parameters

Add to `/etc/sysctl.conf`:

```bash
# Network performance
net.core.rmem_max = 134217728
net.core.wmem_max = 134217728
net.ipv4.tcp_rmem = 4096 87380 134217728
net.ipv4.tcp_wmem = 4096 65536 134217728
net.core.netdev_max_backlog = 30000
net.ipv4.tcp_congestion_control = bbr
net.core.default_qdisc = fq
net.ipv4.tcp_mtu_probing = 1

# Connection handling
net.ipv4.ip_local_port_range = 1024 65535
net.ipv4.tcp_max_syn_backlog = 65536
net.core.somaxconn = 65536
net.ipv4.tcp_tw_reuse = 1
net.ipv4.tcp_fin_timeout = 30

# Memory
vm.swappiness = 10
vm.dirty_ratio = 15
vm.dirty_background_ratio = 5
vm.overcommit_memory = 1

# File handles
fs.file-max = 2097152
fs.nr_open = 2097152
```

Apply settings:
```bash
sudo sysctl -p
```

#### CPU Governor

Set CPU to performance mode:
```bash
echo performance | sudo tee /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor
```

#### NUMA Configuration

Bind NRDOT-HOST to specific NUMA node:
```bash
numactl --cpunodebind=0 --membind=0 ./nrdot-host
```

#### Transparent Huge Pages

Disable for consistent performance:
```bash
echo never | sudo tee /sys/kernel/mm/transparent_hugepage/enabled
echo never | sudo tee /sys/kernel/mm/transparent_hugepage/defrag
```

## Application Tuning

### Memory Management

#### Garbage Collection

For high-throughput scenarios:
```yaml
memory:
  gc_percent: 200  # Reduce GC frequency
  gc_target_percentage: 80
```

For low-latency scenarios:
```yaml
memory:
  gc_percent: 50  # More frequent GC
  gc_target_percentage: 70
```

#### Memory Pools

Enable object pooling:
```yaml
memory:
  object_pools:
    events: 100000      # Pre-allocate event objects
    batches: 1000       # Pre-allocate batch containers
    connections: 1000   # Connection pool size
```

### CPU Optimization

#### CPU Affinity

Pin worker threads to specific CPUs:
```yaml
cpu:
  affinity:
    enabled: true
    cpus: [0, 1, 2, 3, 4, 5, 6, 7]
```

#### Thread Configuration

Optimize thread counts:
```yaml
cpu:
  runtime_threads: 16    # Go runtime threads
  io_threads: 8          # I/O handling threads
  worker_threads: 32     # Processing threads
```

### Network Optimization

#### Buffer Sizes

Configure based on bandwidth-delay product:
```yaml
network:
  receive_buffer: 4194304  # 4MB for 10Gbps
  send_buffer: 4194304
```

#### Connection Pooling

Maintain persistent connections:
```yaml
network:
  http_client:
    max_idle_conns: 1000
    max_conns_per_host: 100
    idle_conn_timeout: "90s"
```

### Data Pipeline Optimization

#### Batching

Optimal batch sizes for different scenarios:

**High Throughput**:
```yaml
processors:
  - name: "batch-aggregator"
    config:
      size: 50000
      timeout: "500ms"
```

**Low Latency**:
```yaml
processors:
  - name: "batch-aggregator"
    config:
      size: 1000
      timeout: "10ms"
```

#### Parallelism

Configure parallel processing:
```yaml
processors:
  - name: "parallel-processor"
    config:
      workers: 32
      queue_size: 100000
      strategy: "work_stealing"  # Better for uneven loads
```

### Storage Optimization

#### ClickHouse

Optimize for bulk inserts:
```yaml
outputs:
  - name: "clickhouse"
    config:
      batch_size: 100000
      async_insert: true
      compression: "lz4"
```

#### S3/Parquet

Optimize file sizes:
```yaml
outputs:
  - name: "s3-parquet"
    config:
      row_group_size: 134217728  # 128MB
      page_size: 1048576         # 1MB
      compression: "snappy"
```

## Monitoring and Profiling

### Performance Metrics

Key metrics to monitor:

1. **Throughput**:
   - Events per second
   - Bytes per second
   - Batches per second

2. **Latency**:
   - End-to-end processing time
   - Pipeline stage durations
   - Output write latency

3. **Resource Usage**:
   - CPU utilization per core
   - Memory allocation rate
   - GC pause times
   - Network bandwidth

### Profiling

Enable CPU profiling:
```yaml
debug:
  cpu_profile:
    enabled: true
    path: "/tmp/cpu.prof"
    duration: "30s"
```

Enable memory profiling:
```yaml
debug:
  mem_profile:
    enabled: true
    path: "/tmp/mem.prof"
    interval: "10s"
```

Analyze profiles:
```bash
go tool pprof -http=:8080 cpu.prof
go tool pprof -http=:8081 mem.prof
```

### Benchmarking

Run built-in benchmarks:
```bash
nrdot-host benchmark --duration 60s --events 1000000
```

Custom load testing:
```bash
# Generate load
nrdot-load-gen --rate 100000 --duration 300s --payload-size 1024

# Monitor performance
nrdot-monitor --interval 1s
```

## Troubleshooting Performance Issues

### High CPU Usage

1. Check GC activity:
   ```bash
   GODEBUG=gctrace=1 ./nrdot-host
   ```

2. Profile CPU usage:
   ```bash
   curl http://localhost:8080/debug/pprof/profile?seconds=30 > cpu.prof
   go tool pprof cpu.prof
   ```

3. Review hot paths and optimize

### Memory Growth

1. Check for leaks:
   ```bash
   curl http://localhost:8080/debug/pprof/heap > heap.prof
   go tool pprof heap.prof
   ```

2. Monitor allocations:
   ```bash
   curl http://localhost:8080/debug/pprof/allocs > allocs.prof
   go tool pprof allocs.prof
   ```

3. Tune GC and pool sizes

### Network Bottlenecks

1. Check network stats:
   ```bash
   ss -s
   netstat -s
   ```

2. Monitor packet drops:
   ```bash
   ip -s link show
   ```

3. Tune kernel parameters and buffers

### Storage Latency

1. Monitor disk I/O:
   ```bash
   iostat -x 1
   ```

2. Check write amplification:
   ```bash
   iotop -o
   ```

3. Optimize batch sizes and compression

## Performance Testing

### Load Testing Script

```bash
#!/bin/bash

# Test configuration
DURATION=300
RATE=100000
PAYLOAD_SIZE=1024
WORKERS=32

echo "Starting performance test..."
echo "Duration: ${DURATION}s"
echo "Rate: ${RATE} events/s"
echo "Payload: ${PAYLOAD_SIZE} bytes"
echo "Workers: ${WORKERS}"

# Start monitoring
nrdot-monitor --interval 1s > perf_results.log &
MONITOR_PID=$!

# Run load test
nrdot-load-gen \
  --rate ${RATE} \
  --duration ${DURATION} \
  --payload-size ${PAYLOAD_SIZE} \
  --workers ${WORKERS} \
  --endpoint http://localhost:8080/events

# Stop monitoring
kill $MONITOR_PID

# Analyze results
echo "Test completed. Analyzing results..."
nrdot-analyze perf_results.log
```

### Stress Testing

Test system limits:
```bash
# Find maximum throughput
nrdot-stress --mode throughput --step 10000 --duration 60

# Find minimum latency
nrdot-stress --mode latency --target-p99 10ms

# Test resource limits
nrdot-stress --mode resources --memory-limit 20GB --cpu-limit 800%
```

## Best Practices

1. **Start with profiling**: Always profile before optimizing
2. **Optimize hot paths**: Focus on the most frequently executed code
3. **Batch operations**: Group similar operations together
4. **Avoid allocations**: Reuse objects where possible
5. **Use appropriate data structures**: Choose based on access patterns
6. **Monitor continuously**: Set up alerts for performance degradation
7. **Test at scale**: Performance characteristics change with load
8. **Document changes**: Keep track of what optimizations work
9. **Measure impact**: Quantify the improvement of each change
10. **Consider trade-offs**: Balance throughput, latency, and resource usage

## Configuration Templates

### Ultra-High Throughput (1M+ events/sec)

Focus on maximum throughput with acceptable latency:
- Large batches (50k+ events)
- Parallel processing (32+ workers)
- Minimal logging
- Relaxed consistency

### Low Latency (<5ms p99)

Focus on minimal latency with good throughput:
- Small batches (100-1000 events)
- Fast algorithms (no compression)
- Direct I/O paths
- Minimal buffering

### Balanced Performance

Good throughput and latency for general use:
- Medium batches (5-10k events)
- Adaptive algorithms
- Smart buffering
- Efficient resource usage

Choose the profile that best matches your requirements and fine-tune from there.