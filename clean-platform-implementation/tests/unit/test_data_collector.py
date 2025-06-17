"""Unit tests for data collector service"""

import pytest
import json
from datetime import datetime
from unittest.mock import Mock, patch, MagicMock
import sys
import os

# Add the services directory to the path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..', '..', 'services', 'data-collector'))

from app import app, data_schema


class TestDataCollector:
    """Test cases for data collector service"""
    
    @pytest.fixture
    def client(self):
        """Create test client"""
        app.config['TESTING'] = True
        with app.test_client() as client:
            yield client
    
    @pytest.fixture
    def mock_redis(self):
        """Mock Redis client"""
        with patch('app.redis_client') as mock:
            yield mock
    
    def test_health_endpoint(self, client, mock_redis):
        """Test health check endpoint"""
        mock_redis.ping.return_value = True
        
        response = client.get('/health')
        assert response.status_code == 200
        
        data = json.loads(response.data)
        assert data['status'] == 'healthy'
        assert 'timestamp' in data
        assert data['service'] == 'data-collector'
    
    def test_health_endpoint_unhealthy(self, client, mock_redis):
        """Test health check when Redis is down"""
        mock_redis.ping.side_effect = Exception("Connection failed")
        
        response = client.get('/health')
        assert response.status_code == 503
        
        data = json.loads(response.data)
        assert data['status'] == 'unhealthy'
        assert 'error' in data
    
    def test_healthz_endpoint(self, client):
        """Test Kubernetes liveness probe"""
        response = client.get('/healthz')
        assert response.status_code == 200
        
        data = json.loads(response.data)
        assert data['status'] == 'ok'
    
    def test_readyz_endpoint(self, client, mock_redis):
        """Test Kubernetes readiness probe"""
        mock_redis.ping.return_value = True
        
        response = client.get('/readyz')
        assert response.status_code == 200
        
        data = json.loads(response.data)
        assert data['status'] == 'ready'
    
    def test_readyz_not_ready(self, client, mock_redis):
        """Test readiness when dependencies are not available"""
        mock_redis.ping.side_effect = Exception("Not connected")
        
        response = client.get('/readyz')
        assert response.status_code == 503
        
        data = json.loads(response.data)
        assert data['status'] == 'not ready'
    
    def test_collect_valid_data(self, client, mock_redis):
        """Test collecting valid data"""
        test_data = {
            "timestamp": datetime.utcnow().isoformat(),
            "data_type": "temperature",
            "value": 23.5,
            "metadata": {"sensor": "sensor1"}
        }
        
        response = client.post('/collect',
                              data=json.dumps(test_data),
                              content_type='application/json')
        
        assert response.status_code == 202
        data = json.loads(response.data)
        assert data['status'] == 'accepted'
        assert data['count'] == 1
        
        # Verify Redis was called
        mock_redis.lpush.assert_called_once()
    
    def test_collect_batch_data(self, client, mock_redis):
        """Test collecting batch data"""
        test_data = [
            {
                "timestamp": datetime.utcnow().isoformat(),
                "data_type": "temperature",
                "value": 23.5
            },
            {
                "timestamp": datetime.utcnow().isoformat(),
                "data_type": "humidity",
                "value": 65.0
            }
        ]
        
        response = client.post('/collect',
                              data=json.dumps(test_data),
                              content_type='application/json')
        
        assert response.status_code == 202
        data = json.loads(response.data)
        assert data['count'] == 2
        
        # Verify Redis was called twice
        assert mock_redis.lpush.call_count == 2
    
    def test_collect_invalid_content_type(self, client):
        """Test rejection of invalid content type"""
        response = client.post('/collect',
                              data="invalid",
                              content_type='text/plain')
        
        assert response.status_code == 400
        data = json.loads(response.data)
        assert 'Content-Type must be application/json' in data['error']
    
    def test_collect_invalid_json(self, client):
        """Test rejection of invalid JSON"""
        response = client.post('/collect',
                              data="invalid json",
                              content_type='application/json')
        
        assert response.status_code == 400
        data = json.loads(response.data)
        assert data['error'] == 'Invalid JSON'
    
    def test_collect_validation_error(self, client):
        """Test data validation error"""
        test_data = {
            "data_type": "temperature",
            "value": "not a number"  # Invalid: should be float
        }
        
        response = client.post('/collect',
                              data=json.dumps(test_data),
                              content_type='application/json')
        
        assert response.status_code == 400
        data = json.loads(response.data)
        assert data['error'] == 'Validation failed'
        assert 'details' in data
    
    def test_stats_endpoint(self, client, mock_redis):
        """Test stats endpoint"""
        mock_redis.llen.return_value = 42
        
        response = client.get('/stats')
        assert response.status_code == 200
        
        data = json.loads(response.data)
        assert data['queue_length'] == 42
        assert 'timestamp' in data
    
    def test_stats_error(self, client, mock_redis):
        """Test stats endpoint error handling"""
        mock_redis.llen.side_effect = Exception("Redis error")
        
        response = client.get('/stats')
        assert response.status_code == 500
        
        data = json.loads(response.data)
        assert data['error'] == 'Internal server error'
    
    def test_404_handler(self, client):
        """Test 404 error handler"""
        response = client.get('/nonexistent')
        assert response.status_code == 404
        
        data = json.loads(response.data)
        assert data['error'] == 'Not found'
    
    def test_data_schema_validation(self):
        """Test data schema validation"""
        # Valid data
        valid_data = {
            "timestamp": datetime.utcnow().isoformat(),
            "data_type": "test",
            "value": 42.0,
            "metadata": {"key": "value"}
        }
        result = data_schema.load(valid_data)
        assert result['data_type'] == 'test'
        assert result['value'] == 42.0
        
        # Missing required field
        with pytest.raises(Exception):
            data_schema.load({"data_type": "test"})
        
        # Invalid type
        with pytest.raises(Exception):
            data_schema.load({
                "timestamp": "not a datetime",
                "data_type": "test",
                "value": "not a float"
            })


class TestMetrics:
    """Test Prometheus metrics"""
    
    @pytest.fixture
    def client(self):
        """Create test client"""
        app.config['TESTING'] = True
        with app.test_client() as client:
            yield client
    
    def test_metrics_endpoint(self, client):
        """Test metrics endpoint returns Prometheus format"""
        response = client.get('/metrics')
        assert response.status_code == 200
        assert response.content_type == 'text/plain; version=0.0.4; charset=utf-8'
        
        # Check for expected metrics
        metrics_text = response.data.decode('utf-8')
        assert 'data_collector_requests_total' in metrics_text
        assert 'data_collector_request_duration_seconds' in metrics_text
        assert 'data_collector_processed_total' in metrics_text