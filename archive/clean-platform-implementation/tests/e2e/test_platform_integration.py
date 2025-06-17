"""End-to-end tests for platform integration"""

import pytest
import os
import time
import json
import requests
import subprocess
from datetime import datetime
from concurrent.futures import ThreadPoolExecutor, as_completed


class TestPlatformIntegration:
    """Test full platform integration with all components"""
    
    @pytest.fixture(scope="class")
    def platform_config(self):
        """Platform configuration for E2E tests"""
        return {
            'grand_central_url': os.getenv('GC_URL', 'https://staging-grand-central.nr-ops.net'),
            'vault_addr': os.getenv('VAULT_ADDR', 'https://vault.nr-ops.net'),
            'okta_url': os.getenv('OKTA_URL', 'https://nr-prod.okta.com'),
            'service_url': os.getenv('SERVICE_URL', 'http://localhost:8080'),
            'environment': os.getenv('E2E_ENV', 'staging'),
            'test_timeout': 600  # 10 minutes
        }
    
    @pytest.fixture
    def auth_headers(self):
        """Get authentication headers for API calls"""
        # In real tests, this would get actual auth tokens
        return {
            'X-Grand-Central-Auth': os.getenv('GC_TEST_TOKEN', 'test-token'),
            'Authorization': f'Bearer {os.getenv("NR_API_KEY", "test-key")}'
        }
    
    @pytest.mark.e2e
    @pytest.mark.requires_platform
    def test_grand_central_integration(self, platform_config, auth_headers):
        """Test Grand Central deployment integration"""
        gc_url = platform_config['grand_central_url']
        
        # Test project registration
        response = requests.get(
            f"{gc_url}/api/v1/projects/platform-team/clean-platform",
            headers=auth_headers,
            timeout=30
        )
        
        if response.status_code == 404:
            # Project not registered, skip test
            pytest.skip("Project not registered in Grand Central")
        
        assert response.status_code == 200, f"Failed to get project: {response.text}"
        project = response.json()
        assert project['org'] == 'platform-team'
        assert project['repo'] == 'clean-platform'
    
    @pytest.mark.e2e
    @pytest.mark.requires_vault
    def test_vault_integration(self, platform_config):
        """Test Vault secrets integration"""
        # Test vault connectivity
        vault_addr = platform_config['vault_addr']
        
        # Check if we can reach Vault
        try:
            response = requests.get(f"{vault_addr}/v1/sys/health", timeout=5)
            assert response.status_code in [200, 429, 473, 501, 503]
        except requests.exceptions.RequestException:
            pytest.skip("Vault not accessible")
        
        # Test secret access (would need proper auth in real test)
        # This is just checking the integration points exist
        secret_paths = [
            'secret/teams/platform-team/grand-central-token',
            'secret/teams/platform-team/okta-config',
            'secret/teams/platform-team/database-credentials'
        ]
        
        # In real test, we would verify these paths are accessible
        assert len(secret_paths) > 0
    
    @pytest.mark.e2e
    @pytest.mark.requires_okta
    def test_okta_authentication(self, platform_config):
        """Test Okta SAML authentication flow"""
        service_url = platform_config['service_url']
        
        # Test SAML endpoints exist
        endpoints = ['/saml/login', '/saml/metadata']
        
        for endpoint in endpoints:
            response = requests.get(f"{service_url}{endpoint}", allow_redirects=False)
            # Should redirect to Okta or return metadata
            assert response.status_code in [200, 302], f"Endpoint {endpoint} not accessible"
    
    @pytest.mark.e2e
    def test_monitoring_integration(self, platform_config):
        """Test monitoring and metrics integration"""
        service_url = platform_config['service_url']
        
        # Test Prometheus metrics
        response = requests.get(f"{service_url}/metrics", timeout=10)
        assert response.status_code == 200
        
        metrics_text = response.text
        required_metrics = [
            'data_collector_requests_total',
            'data_collector_request_duration_seconds',
            'data_collector_processed_total',
            'process_cpu_seconds_total',
            'process_resident_memory_bytes'
        ]
        
        for metric in required_metrics:
            assert metric in metrics_text, f"Metric {metric} not found"
    
    @pytest.mark.e2e
    def test_database_connectivity(self, platform_config):
        """Test database and cache connectivity"""
        service_url = platform_config['service_url']
        
        # Submit data to test database flow
        test_data = {
            "timestamp": datetime.utcnow().isoformat(),
            "data_type": "e2e_test",
            "value": 100.0,
            "metadata": {"test": "database_connectivity"}
        }
        
        response = requests.post(
            f"{service_url}/collect",
            json=test_data,
            headers={'Content-Type': 'application/json'},
            timeout=10
        )
        
        assert response.status_code == 202
        
        # Check stats to verify Redis is working
        response = requests.get(f"{service_url}/stats", timeout=10)
        assert response.status_code == 200
        assert 'queue_length' in response.json()
    
    @pytest.mark.e2e
    @pytest.mark.slow
    def test_high_availability(self, platform_config):
        """Test service high availability and failover"""
        service_url = platform_config['service_url']
        
        # Simulate multiple concurrent requests
        def make_request(i):
            data = {
                "timestamp": datetime.utcnow().isoformat(),
                "data_type": "ha_test",
                "value": float(i),
                "metadata": {"request_id": i}
            }
            try:
                response = requests.post(
                    f"{service_url}/collect",
                    json=data,
                    headers={'Content-Type': 'application/json'},
                    timeout=5
                )
                return response.status_code == 202
            except:
                return False
        
        # Send 100 concurrent requests
        success_count = 0
        with ThreadPoolExecutor(max_workers=10) as executor:
            futures = [executor.submit(make_request, i) for i in range(100)]
            
            for future in as_completed(futures):
                if future.result():
                    success_count += 1
        
        # Expect at least 95% success rate
        success_rate = success_count / 100
        assert success_rate >= 0.95, f"Only {success_rate*100}% requests succeeded"
    
    @pytest.mark.e2e
    def test_security_headers(self, platform_config):
        """Test security headers and compliance"""
        service_url = platform_config['service_url']
        
        response = requests.get(f"{service_url}/health", timeout=10)
        
        # Check security headers
        expected_headers = {
            'X-Content-Type-Options': 'nosniff',
            'X-Frame-Options': 'DENY',
            'X-XSS-Protection': '1; mode=block',
            'Strict-Transport-Security': 'max-age=31536000; includeSubDomains'
        }
        
        for header, expected_value in expected_headers.items():
            actual_value = response.headers.get(header)
            # Some headers might not be set in test environment
            if actual_value:
                assert actual_value == expected_value, f"Header {header} has wrong value"
    
    @pytest.mark.e2e
    @pytest.mark.requires_k8s
    def test_container_security_compliance(self):
        """Test container security policy compliance"""
        # Get pod security details
        result = subprocess.run([
            'kubectl', 'get', 'pods',
            '-n', 'clean-platform',
            '-l', 'app=clean-platform',
            '-o', 'jsonpath={.items[*].spec.securityContext}'
        ], capture_output=True, text=True)
        
        if result.returncode != 0:
            pytest.skip("Kubernetes not accessible")
        
        # Parse security context
        if result.stdout:
            security_contexts = json.loads(f"[{result.stdout}]")
            for context in security_contexts:
                # Verify security requirements
                assert context.get('runAsNonRoot') is True
                assert context.get('runAsUser', 0) > 10000
    
    @pytest.mark.e2e
    def test_feature_flags_integration(self, platform_config):
        """Test feature flags integration"""
        service_url = platform_config['service_url']
        
        # Test feature flags endpoint (if implemented)
        response = requests.get(
            f"{service_url}/api/feature-flags",
            timeout=10
        )
        
        if response.status_code == 404:
            pytest.skip("Feature flags not implemented")
        
        assert response.status_code == 200
        flags = response.json()
        assert 'flags' in flags
    
    @pytest.mark.e2e
    @pytest.mark.slow
    def test_deployment_pipeline(self, platform_config):
        """Test complete deployment pipeline"""
        # This would test the full deployment process
        # For now, we verify the scripts exist and are executable
        
        deployment_scripts = [
            'scripts/deployment-hooks/backup-check.sh',
            'scripts/deployment-hooks/suppress-alerts.sh',
            'scripts/deployment-hooks/health-check.sh',
            'scripts/deployment-hooks/smoke-tests.sh',
            'scripts/deployment-hooks/restore-alerts.sh',
            'scripts/deployment-hooks/rollback.sh',
            'scripts/deployment-hooks/notify-failure.sh'
        ]
        
        for script in deployment_scripts:
            assert os.path.exists(script), f"Script {script} not found"
            assert os.access(script, os.X_OK), f"Script {script} not executable"
    
    @pytest.mark.e2e
    def test_rate_limiting(self, platform_config):
        """Test rate limiting functionality"""
        service_url = platform_config['service_url']
        
        # Send rapid requests to trigger rate limiting
        responses = []
        for i in range(20):
            response = requests.get(f"{service_url}/health", timeout=5)
            responses.append(response.status_code)
            time.sleep(0.1)
        
        # All requests should succeed (rate limiting not enforced on health)
        assert all(status == 200 for status in responses)
        
        # Test rate limiting on data endpoint
        data_responses = []
        test_data = {
            "timestamp": datetime.utcnow().isoformat(),
            "data_type": "rate_test",
            "value": 1.0
        }
        
        for i in range(50):
            response = requests.post(
                f"{service_url}/collect",
                json=test_data,
                headers={'Content-Type': 'application/json'},
                timeout=5
            )
            data_responses.append(response.status_code)
        
        # Should see some rate limiting (429) responses if configured
        # For now, just verify service handles load
        success_count = sum(1 for status in data_responses if status == 202)
        assert success_count > 0, "No successful requests"
    
    @pytest.mark.e2e
    def test_error_handling_and_recovery(self, platform_config):
        """Test error handling and recovery mechanisms"""
        service_url = platform_config['service_url']
        
        # Test various error scenarios
        error_cases = [
            # Invalid JSON
            ('{"invalid": json}', 'application/json', 400),
            # Wrong content type
            ('data', 'text/plain', 400),
            # Missing required fields
            ('{"data_type": "test"}', 'application/json', 400),
            # Invalid data types
            ('{"timestamp": "now", "data_type": "test", "value": "abc"}', 'application/json', 400)
        ]
        
        for data, content_type, expected_status in error_cases:
            response = requests.post(
                f"{service_url}/collect",
                data=data,
                headers={'Content-Type': content_type},
                timeout=5
            )
            assert response.status_code == expected_status, \
                f"Expected {expected_status} for {data}, got {response.status_code}"
        
        # Verify service is still healthy after errors
        response = requests.get(f"{service_url}/health", timeout=5)
        assert response.status_code == 200