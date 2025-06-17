"""
Performance testing suite for Platform API using Locust
"""
import json
import random
import time
import uuid
from datetime import datetime, timezone
from typing import Dict, Any

from locust import HttpUser, task, between, events
from locust.env import Environment
from locust.stats import stats_printer, stats_history
from locust.log import setup_logging


# Setup logging
setup_logging("INFO", None)


class PlatformUser(HttpUser):
    """Simulated platform user for load testing"""
    
    wait_time = between(1, 3)  # Wait 1-3 seconds between tasks
    
    def on_start(self):
        """Initialize user session"""
        # Generate unique user ID for this session
        self.user_id = str(uuid.uuid4())
        self.api_key = self.get_api_key()
        
        # Set common headers
        self.client.headers.update({
            "Content-Type": "application/json",
            "X-API-Key": self.api_key,
            "X-User-ID": self.user_id,
        })
        
        # Initialize data cache for realistic workflows
        self.collected_data_ids = []
        self.processing_task_ids = []
    
    def get_api_key(self) -> str:
        """Get API key for testing (would be from config in real scenario)"""
        return "test-api-key-12345"
    
    @task(10)
    def collect_data(self):
        """Test data collection endpoint - most frequent operation"""
        data = {
            "source": f"load-test-{self.user_id}",
            "timestamp": datetime.now(timezone.utc).isoformat(),
            "metrics": {
                "cpu_usage": random.uniform(10, 90),
                "memory_usage": random.uniform(20, 80),
                "disk_usage": random.uniform(30, 70),
                "network_in": random.randint(1000, 1000000),
                "network_out": random.randint(1000, 1000000),
            },
            "tags": {
                "host": f"host-{random.randint(1, 100)}",
                "region": random.choice(["us-east-1", "us-west-2", "eu-west-1"]),
                "environment": random.choice(["dev", "staging", "prod"]),
                "service": random.choice(["api", "web", "worker", "database"]),
            }
        }
        
        with self.client.post(
            "/api/v1/collect",
            json=data,
            name="/api/v1/collect",
            catch_response=True
        ) as response:
            if response.status_code == 202:
                response_data = response.json()
                if "id" in response_data:
                    self.collected_data_ids.append(response_data["id"])
                    # Keep only last 100 IDs to avoid memory issues
                    if len(self.collected_data_ids) > 100:
                        self.collected_data_ids = self.collected_data_ids[-100:]
                response.success()
            else:
                response.failure(f"Got status code {response.status_code}")
    
    @task(5)
    def process_data(self):
        """Test data processing endpoint"""
        if not self.collected_data_ids:
            # Skip if no data collected yet
            return
        
        data_id = random.choice(self.collected_data_ids)
        processing_request = {
            "data_id": data_id,
            "processing_type": random.choice(["aggregate", "transform", "analyze"]),
            "parameters": {
                "time_bucket": random.choice(["1m", "5m", "1h"]),
                "aggregations": ["mean", "max", "min", "p95"],
            },
            "priority": random.randint(1, 10)
        }
        
        with self.client.post(
            "/api/v1/process",
            json=processing_request,
            name="/api/v1/process",
            catch_response=True
        ) as response:
            if response.status_code == 202:
                response_data = response.json()
                if "task_id" in response_data:
                    self.processing_task_ids.append(response_data["task_id"])
                    if len(self.processing_task_ids) > 50:
                        self.processing_task_ids = self.processing_task_ids[-50:]
                response.success()
            else:
                response.failure(f"Got status code {response.status_code}")
    
    @task(3)
    def check_processing_status(self):
        """Check status of processing tasks"""
        if not self.processing_task_ids:
            return
        
        task_id = random.choice(self.processing_task_ids)
        
        with self.client.get(
            f"/api/v1/process/status/{task_id}",
            name="/api/v1/process/status/[task_id]",
            catch_response=True
        ) as response:
            if response.status_code == 200:
                response.success()
            elif response.status_code == 404:
                # Task might have been cleaned up
                self.processing_task_ids.remove(task_id)
                response.success()
            else:
                response.failure(f"Got status code {response.status_code}")
    
    @task(8)
    def query_data(self):
        """Test data query endpoint"""
        query_request = {
            "query_type": random.choice(["aggregated", "raw"]),
            "filters": {
                "environment": random.choice(["dev", "staging", "prod"]),
                "service": random.choice(["api", "web", "worker", "database"]),
            },
            "start_time": datetime.now(timezone.utc).replace(hour=0, minute=0, second=0).isoformat(),
            "end_time": datetime.now(timezone.utc).isoformat(),
            "limit": random.choice([10, 50, 100]),
            "offset": 0
        }
        
        with self.client.post(
            "/api/v1/query",
            json=query_request,
            name="/api/v1/query",
            catch_response=True
        ) as response:
            if response.status_code == 200:
                response_data = response.json()
                if "data" in response_data and isinstance(response_data["data"], list):
                    response.success()
                else:
                    response.failure("Invalid response format")
            else:
                response.failure(f"Got status code {response.status_code}")
    
    @task(2)
    def health_check(self):
        """Periodic health check"""
        with self.client.get(
            "/health",
            name="/health",
            catch_response=True
        ) as response:
            if response.status_code == 200:
                response.success()
            else:
                response.failure(f"Health check failed with status {response.status_code}")
    
    @task(1)
    def metrics_endpoint(self):
        """Check metrics endpoint"""
        with self.client.get(
            "/metrics",
            name="/metrics",
            catch_response=True
        ) as response:
            if response.status_code == 200:
                # Verify it's actually Prometheus format
                if "# HELP" in response.text and "# TYPE" in response.text:
                    response.success()
                else:
                    response.failure("Invalid metrics format")
            else:
                response.failure(f"Got status code {response.status_code}")


class AdminUser(HttpUser):
    """Simulated admin user performing management tasks"""
    
    wait_time = between(5, 10)  # Admins perform actions less frequently
    weight = 1  # Only 1 admin for every 10 regular users
    
    def on_start(self):
        """Initialize admin session"""
        self.admin_id = f"admin-{uuid.uuid4()}"
        self.api_key = "admin-api-key-67890"  # Admin API key
        
        self.client.headers.update({
            "Content-Type": "application/json",
            "X-API-Key": self.api_key,
            "X-Admin-ID": self.admin_id,
        })
    
    @task(5)
    def bulk_query(self):
        """Admin bulk data query"""
        query_request = {
            "query_type": "aggregated",
            "filters": {},  # No filters - query all data
            "start_time": datetime.now(timezone.utc).replace(day=1).isoformat(),
            "end_time": datetime.now(timezone.utc).isoformat(),
            "limit": 1000,
            "offset": 0
        }
        
        with self.client.post(
            "/api/v1/query",
            json=query_request,
            name="/api/v1/query [admin-bulk]",
            catch_response=True
        ) as response:
            if response.status_code == 200:
                response.success()
            else:
                response.failure(f"Admin query failed with status {response.status_code}")
    
    @task(3)
    def system_stats(self):
        """Get system statistics"""
        endpoints = [
            "/api/v1/stats/processing",
            "/api/v1/stats/storage",
            "/api/v1/stats/performance"
        ]
        
        for endpoint in endpoints:
            with self.client.get(
                endpoint,
                name=endpoint,
                catch_response=True
            ) as response:
                if response.status_code == 200:
                    response.success()
                else:
                    response.failure(f"Stats endpoint failed with status {response.status_code}")
    
    @task(1)
    def trigger_maintenance(self):
        """Trigger maintenance tasks"""
        maintenance_tasks = [
            {"task": "cleanup_old_data", "params": {"days": 30}},
            {"task": "optimize_indices", "params": {"force": False}},
            {"task": "refresh_materialized_views", "params": {}},
        ]
        
        task = random.choice(maintenance_tasks)
        
        with self.client.post(
            "/api/v1/admin/maintenance",
            json=task,
            name="/api/v1/admin/maintenance",
            catch_response=True
        ) as response:
            if response.status_code in [200, 202]:
                response.success()
            else:
                response.failure(f"Maintenance task failed with status {response.status_code}")


class StressUser(HttpUser):
    """User for stress testing with aggressive patterns"""
    
    wait_time = between(0.1, 0.5)  # Very aggressive timing
    weight = 0.5  # Only use for specific stress tests
    
    def on_start(self):
        self.client.headers.update({
            "Content-Type": "application/json",
            "X-API-Key": "stress-test-key",
        })
    
    @task
    def rapid_fire_collect(self):
        """Send data as fast as possible"""
        batch_data = []
        for i in range(10):  # Send 10 items per request
            batch_data.append({
                "source": f"stress-test-{i}",
                "timestamp": datetime.now(timezone.utc).isoformat(),
                "value": random.random() * 1000,
                "tags": {"test": "stress"}
            })
        
        with self.client.post(
            "/api/v1/collect/batch",
            json={"data": batch_data},
            name="/api/v1/collect/batch [stress]",
            catch_response=True
        ) as response:
            if response.status_code in [200, 202]:
                response.success()
            elif response.status_code == 429:
                # Rate limited - this is expected
                response.success()
            else:
                response.failure(f"Unexpected status {response.status_code}")


# Custom event handlers for detailed reporting
@events.test_start.add_listener
def on_test_start(environment, **kwargs):
    """Initialize test reporting"""
    print(f"Load test starting with {environment.parsed_options.num_users} users")
    print(f"Spawn rate: {environment.parsed_options.spawn_rate} users/second")
    print(f"Host: {environment.host}")


@events.test_stop.add_listener
def on_test_stop(environment, **kwargs):
    """Generate test report"""
    print("\n" + "="*80)
    print("LOAD TEST SUMMARY")
    print("="*80)
    
    # Calculate percentiles
    response_times = []
    for key, value in environment.stats.entries.items():
        if value.num_requests > 0:
            response_times.extend([value.avg_response_time] * value.num_requests)
    
    if response_times:
        response_times.sort()
        p50 = response_times[int(len(response_times) * 0.5)]
        p90 = response_times[int(len(response_times) * 0.9)]
        p95 = response_times[int(len(response_times) * 0.95)]
        p99 = response_times[int(len(response_times) * 0.99)]
        
        print(f"\nResponse Time Percentiles:")
        print(f"  P50: {p50:.2f} ms")
        print(f"  P90: {p90:.2f} ms")
        print(f"  P95: {p95:.2f} ms")
        print(f"  P99: {p99:.2f} ms")
    
    # Error summary
    total_failures = sum(value.num_failures for value in environment.stats.entries.values())
    total_requests = sum(value.num_requests for value in environment.stats.entries.values())
    
    if total_requests > 0:
        error_rate = (total_failures / total_requests) * 100
        print(f"\nError Rate: {error_rate:.2f}%")
        print(f"Total Requests: {total_requests}")
        print(f"Total Failures: {total_failures}")


# Test scenarios
class LightLoadTest(PlatformUser):
    """Light load test scenario"""
    weight = 10
    

class MediumLoadTest(PlatformUser):
    """Medium load test scenario"""
    weight = 5
    wait_time = between(0.5, 2)


class HeavyLoadTest(PlatformUser):
    """Heavy load test scenario"""
    weight = 2
    wait_time = between(0.1, 1)


# Spike test user
class SpikeTestUser(PlatformUser):
    """User for spike testing"""
    wait_time = between(0.1, 0.3)
    
    @task(20)
    def spike_collect(self):
        """Generate spike load"""
        for _ in range(5):
            super().collect_data()
            time.sleep(0.1)