#!/bin/bash

# Load generation script for host monitoring test

echo "Starting load generation..."

# CPU load
echo "Generating CPU load..."
stress-ng --cpu 2 --cpu-load 80 --timeout 60s &

# Memory load
echo "Generating memory load..."
stress-ng --vm 2 --vm-bytes 256M --vm-method all --timeout 60s &

# Disk I/O load
echo "Generating disk I/O load..."
stress-ng --hdd 2 --hdd-bytes 100M --timeout 60s &

# Network load
echo "Generating network load..."
for i in {1..100}; do
    curl -s http://localhost:8888/metrics > /dev/null 2>&1 &
    sleep 0.1
done

wait
echo "Load generation complete"