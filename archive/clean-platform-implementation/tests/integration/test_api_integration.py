import pytest
import requests
import time
import json
from datetime import datetime, timezone
from typing import Dict, Any
import redis
import psycopg2
from psycopg2.extras import RealDictCursor


class TestAPIIntegration:
    """Integration tests for the platform API"""
    
    @pytest.fixture(scope="class")
    def api_base_url(self):
        """Get API base URL from environment or use default"""
        import os
        return os.getenv('API_BASE_URL', 'http://localhost:8080')
    
    @pytest.fixture(scope="class")
    def redis_client(self):
        """Redis client for test verification"""
        import os
        redis_url = os.getenv('REDIS_URL', 'redis://localhost:6379')
        return redis.from_url(redis_url, decode_responses=True)
    
    @pytest.fixture(scope="class")
    def db_connection(self):
        """Database connection for test verification"""
        import os
        db_url = os.getenv('DATABASE_URL', 'postgresql://postgres:postgres@localhost:5432/platform')
        conn = psycopg2.connect(db_url, cursor_factory=RealDictCursor)
        yield conn
        conn.close()
    
    def test_health_endpoints(self, api_base_url):
        """Test health check endpoints"""
        # API Gateway health
        response = requests.get(f"{api_base_url}/health")
        assert response.status_code == 200
        assert response.text.strip() == "healthy"
        
        # Data Collector health
        response = requests.get(f"{api_base_url}/api/v1/collector/health")
        assert response.status_code == 200
        data = response.json()
        assert data['status'] in ['healthy', 'degraded']
        assert 'redis' in data
        assert 'database' in data
        
        # Data Processor health
        response = requests.get(f"{api_base_url}/api/v1/processor/health")
        assert response.status_code == 200
        data = response.json()
        assert data['status'] in ['healthy', 'degraded']
    
    def test_data_collection_flow(self, api_base_url, redis_client):
        """Test complete data collection flow"""
        # Prepare test data
        test_data = {
            'source': 'test_integration',
            'timestamp': datetime.now(timezone.utc).isoformat(),
            'metrics': {
                'cpu_usage': 45.2,
                'memory_usage': 62.8,
                'disk_usage': 78.1
            },
            'tags': {
                'host': 'test-host-01',
                'region': 'us-east-1',
                'environment': 'test'
            }
        }
        
        # Submit data
        response = requests.post(
            f"{api_base_url}/api/v1/collect",
            json=test_data,
            headers={'Content-Type': 'application/json'}
        )
        assert response.status_code == 202
        collection_id = response.json()['id']
        
        # Verify data in Redis
        queue_data = redis_client.get(f"queue:data:{collection_id}")
        assert queue_data is not None
        stored_data = json.loads(queue_data)
        assert stored_data['source'] == test_data['source']
        assert stored_data['metrics'] == test_data['metrics']
    
    def test_data_processing_flow(self, api_base_url, db_connection):
        """Test data processing workflow"""
        # Submit processing request
        process_request = {
            'data_id': 'test_data_001',
            'processing_type': 'aggregate',
            'parameters': {
                'time_bucket': '5m',
                'aggregations': ['mean', 'max', 'min']
            },
            'priority': 5
        }
        
        response = requests.post(
            f"{api_base_url}/api/v1/process",
            json=process_request,
            headers={'Content-Type': 'application/json'}
        )
        assert response.status_code == 202
        task_data = response.json()
        assert 'task_id' in task_data
        assert 'status_url' in task_data
        
        # Check task status
        task_id = task_data['task_id']
        max_attempts = 30
        for i in range(max_attempts):
            response = requests.get(f"{api_base_url}{task_data['status_url']}")
            assert response.status_code == 200
            status_data = response.json()
            
            if status_data['status'] in ['completed', 'failed']:
                break
            
            time.sleep(1)
        
        assert status_data['status'] == 'completed'
        assert 'result' in status_data
    
    def test_query_functionality(self, api_base_url):
        """Test data query functionality"""
        # Simple query
        query_request = {
            'query_type': 'aggregated',
            'filters': {
                'environment': 'test'
            },
            'limit': 10
        }
        
        response = requests.post(
            f"{api_base_url}/api/v1/query",
            json=query_request,
            headers={'Content-Type': 'application/json'}
        )
        assert response.status_code == 200
        result = response.json()
        assert 'data' in result
        assert 'count' in result
        assert result['limit'] == 10
        
        # Time-based query
        query_request = {
            'query_type': 'raw',
            'start_time': '2024-01-01T00:00:00Z',
            'end_time': '2024-12-31T23:59:59Z',
            'limit': 100
        }
        
        response = requests.post(
            f"{api_base_url}/api/v1/query",
            json=query_request,
            headers={'Content-Type': 'application/json'}
        )
        assert response.status_code == 200
    
    def test_rate_limiting(self, api_base_url):
        """Test API rate limiting"""
        # Make rapid requests
        responses = []
        for i in range(15):
            response = requests.post(
                f"{api_base_url}/api/v1/collect",
                json={'test': 'data'},
                headers={'Content-Type': 'application/json'}
            )
            responses.append(response.status_code)
        
        # Should have some 429 responses due to rate limiting
        assert 429 in responses
    
    def test_error_handling(self, api_base_url):
        """Test API error handling"""
        # Invalid data
        response = requests.post(
            f"{api_base_url}/api/v1/collect",
            json={'invalid': 'structure'},
            headers={'Content-Type': 'application/json'}
        )
        assert response.status_code == 400
        error_data = response.json()
        assert 'error' in error_data
        
        # Missing required fields
        response = requests.post(
            f"{api_base_url}/api/v1/process",
            json={'incomplete': 'data'},
            headers={'Content-Type': 'application/json'}
        )
        assert response.status_code == 400
        
        # Non-existent endpoint
        response = requests.get(f"{api_base_url}/api/v1/nonexistent")
        assert response.status_code == 404
    
    def test_metrics_endpoint(self, api_base_url):
        """Test Prometheus metrics endpoint"""
        response = requests.get(f"{api_base_url}/metrics")
        assert response.status_code == 200
        metrics_text = response.text
        
        # Check for expected metrics
        assert 'data_collector_requests_total' in metrics_text
        assert 'data_processor_requests_total' in metrics_text
        assert 'request_duration_seconds' in metrics_text
    
    @pytest.mark.parametrize("endpoint,method", [
        ("/api/v1/collect", "POST"),
        ("/api/v1/process", "POST"),
        ("/api/v1/query", "POST"),
        ("/health", "GET"),
        ("/metrics", "GET")
    ])
    def test_endpoint_availability(self, api_base_url, endpoint, method):
        """Test that all endpoints are available"""
        url = f"{api_base_url}{endpoint}"
        
        if method == "GET":
            response = requests.get(url)
        else:
            response = requests.post(url, json={}, headers={'Content-Type': 'application/json'})
        
        # Should not return 404 or 405
        assert response.status_code not in [404, 405]
    
    def test_concurrent_requests(self, api_base_url):
        """Test handling of concurrent requests"""
        import concurrent.futures
        
        def make_request(i):
            data = {
                'source': f'concurrent_test_{i}',
                'timestamp': datetime.now(timezone.utc).isoformat(),
                'value': i
            }
            response = requests.post(
                f"{api_base_url}/api/v1/collect",
                json=data,
                headers={'Content-Type': 'application/json'}
            )
            return response.status_code
        
        with concurrent.futures.ThreadPoolExecutor(max_workers=10) as executor:
            futures = [executor.submit(make_request, i) for i in range(50)]
            results = [future.result() for future in concurrent.futures.as_completed(futures)]
        
        # Most requests should succeed
        success_count = sum(1 for status in results if status in [200, 202])
        assert success_count > 40  # At least 80% success rate
    
    def test_data_validation(self, api_base_url):
        """Test input data validation"""
        # Test various invalid inputs
        invalid_inputs = [
            # Missing required fields
            {'metrics': {'test': 1}},
            # Invalid data types
            {'source': 123, 'metrics': 'not_a_dict'},
            # Exceeding limits
            {'source': 'x' * 1000, 'metrics': {f'metric_{i}': i for i in range(1000)}}
        ]
        
        for invalid_data in invalid_inputs:
            response = requests.post(
                f"{api_base_url}/api/v1/collect",
                json=invalid_data,
                headers={'Content-Type': 'application/json'}
            )
            assert response.status_code == 400
            assert 'error' in response.json()


if __name__ == "__main__":
    pytest.main([__file__, "-v"])