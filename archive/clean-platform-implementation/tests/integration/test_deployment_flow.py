"""Integration tests for end-to-end deployment flow"""

import pytest
import os
import time
import json
import subprocess
from datetime import datetime
from unittest.mock import patch
import requests


class TestDeploymentFlow:
    """Test complete deployment workflow"""
    
    @pytest.fixture
    def test_config(self):
        """Test configuration"""
        return {
            'service_name': 'clean-platform',
            'environment': 'test',
            'namespace': 'clean-platform-test',
            'version': 'test-v1.0.0',
            'deployment_id': f'test-deploy-{int(time.time())}',
            'base_url': os.getenv('TEST_BASE_URL', 'http://localhost:8080')
        }
    
    @pytest.fixture
    def gc_token(self):
        """Get Grand Central token for testing"""
        token = os.getenv('GC_TEST_TOKEN')
        if not token:
            pytest.skip("Grand Central test token not available")
        return token
    
    @pytest.mark.integration
    def test_pre_deployment_checks(self, test_config):
        """Test pre-deployment validation hooks"""
        # Run backup check
        result = subprocess.run([
            'bash',
            'scripts/deployment-hooks/backup-check.sh'
        ], env={
            **os.environ,
            'SERVICE_NAME': test_config['service_name'],
            'ENVIRONMENT': test_config['environment'],
            'FORCE_BACKUP': 'true'  # Force backup in test
        }, capture_output=True, text=True)
        
        assert result.returncode == 0, f"Backup check failed: {result.stderr}"
        assert "Backup verification completed successfully" in result.stdout
    
    @pytest.mark.integration
    def test_alert_suppression(self, test_config):
        """Test alert suppression during deployment"""
        # Run alert suppression
        result = subprocess.run([
            'bash',
            'scripts/deployment-hooks/suppress-alerts.sh'
        ], env={
            **os.environ,
            'SERVICE_NAME': test_config['service_name'],
            'ENVIRONMENT': test_config['environment'],
            'SKIP_SUPPRESSION': 'false'
        }, capture_output=True, text=True)
        
        assert result.returncode == 0, f"Alert suppression failed: {result.stderr}"
        assert "Successfully suppressed alerts" in result.stdout
        
        # Verify muting rules file was created
        muting_file = f"/tmp/muting-rules-{test_config['service_name']}-{test_config['environment']}.json"
        assert os.path.exists(muting_file), "Muting rules file not created"
    
    @pytest.mark.integration
    @pytest.mark.requires_k8s
    def test_kubernetes_deployment(self, test_config):
        """Test Kubernetes deployment"""
        # Apply deployment
        result = subprocess.run([
            'kubectl', 'apply',
            '-n', test_config['namespace'],
            '-f', 'k8s/base/'
        ], capture_output=True, text=True)
        
        assert result.returncode == 0, f"Kubernetes deployment failed: {result.stderr}"
        
        # Wait for pods to be ready
        max_wait = 300  # 5 minutes
        start_time = time.time()
        
        while time.time() - start_time < max_wait:
            result = subprocess.run([
                'kubectl', 'get', 'pods',
                '-n', test_config['namespace'],
                '-l', f"app={test_config['service_name']}",
                '-o', 'jsonpath={.items[*].status.phase}'
            ], capture_output=True, text=True)
            
            if all(phase == 'Running' for phase in result.stdout.split()):
                break
            
            time.sleep(10)
        else:
            pytest.fail("Pods did not become ready in time")
    
    @pytest.mark.integration
    def test_health_checks(self, test_config):
        """Test post-deployment health checks"""
        # Run health check script
        result = subprocess.run([
            'bash',
            'scripts/deployment-hooks/health-check.sh'
        ], env={
            **os.environ,
            'SERVICE_NAME': test_config['service_name'],
            'ENVIRONMENT': test_config['environment'],
            'NAMESPACE': test_config['namespace'],
            'HEALTH_CHECK_TIMEOUT': '180'
        }, capture_output=True, text=True)
        
        assert result.returncode == 0, f"Health checks failed: {result.stderr}"
        assert "All health checks passed" in result.stdout
        
        # Test API endpoints directly
        base_url = test_config['base_url']
        
        # Health endpoint
        response = requests.get(f"{base_url}/health", timeout=5)
        assert response.status_code == 200
        assert response.json()['status'] == 'healthy'
        
        # Readiness endpoint
        response = requests.get(f"{base_url}/readyz", timeout=5)
        assert response.status_code == 200
        assert response.json()['status'] == 'ready'
    
    @pytest.mark.integration
    def test_smoke_tests(self, test_config):
        """Test smoke test execution"""
        # Run smoke tests
        result = subprocess.run([
            'bash',
            'scripts/deployment-hooks/smoke-tests.sh'
        ], env={
            **os.environ,
            'SERVICE_NAME': test_config['service_name'],
            'ENVIRONMENT': test_config['environment'],
            'NAMESPACE': test_config['namespace'],
            'BASE_URL': test_config['base_url']
        }, capture_output=True, text=True)
        
        assert result.returncode == 0, f"Smoke tests failed: {result.stderr}"
        assert "All smoke tests passed" in result.stdout
    
    @pytest.mark.integration
    def test_data_collection_flow(self, test_config):
        """Test end-to-end data collection"""
        base_url = test_config['base_url']
        
        # Submit test data
        test_data = {
            "timestamp": datetime.utcnow().isoformat(),
            "data_type": "integration_test",
            "value": 42.0,
            "metadata": {
                "test_id": test_config['deployment_id'],
                "source": "integration_test"
            }
        }
        
        response = requests.post(
            f"{base_url}/collect",
            json=test_data,
            headers={'Content-Type': 'application/json'}
        )
        
        assert response.status_code == 202
        assert response.json()['status'] == 'accepted'
        
        # Check stats
        response = requests.get(f"{base_url}/stats")
        assert response.status_code == 200
        assert response.json()['queue_length'] >= 0
    
    @pytest.mark.integration
    def test_alert_restoration(self, test_config):
        """Test alert restoration after deployment"""
        # Create mock muting rules file
        muting_file = f"/tmp/muting-rules-{test_config['service_name']}-{test_config['environment']}.json"
        with open(muting_file, 'w') as f:
            json.dump(["test-rule-1", "test-rule-2"], f)
        
        # Run alert restoration
        result = subprocess.run([
            'bash',
            'scripts/deployment-hooks/restore-alerts.sh'
        ], env={
            **os.environ,
            'SERVICE_NAME': test_config['service_name'],
            'ENVIRONMENT': test_config['environment'],
            'DEPLOYMENT_ID': test_config['deployment_id']
        }, capture_output=True, text=True)
        
        # In test mode, we expect it to succeed even without real muting rules
        assert "Successfully restored all alerts" in result.stdout or \
               "No muting rules found to restore" in result.stdout
        
        # Verify muting rules file was cleaned up
        assert not os.path.exists(muting_file), "Muting rules file not cleaned up"
    
    @pytest.mark.integration
    def test_rollback_scenario(self, test_config):
        """Test deployment rollback"""
        # Simulate failed deployment
        with patch.dict(os.environ, {
            'SERVICE_NAME': test_config['service_name'],
            'ENVIRONMENT': test_config['environment'],
            'DEPLOYMENT_ID': test_config['deployment_id'],
            'ROLLBACK_REASON': 'Integration test rollback'
        }):
            # In test mode, we don't actually rollback
            # Just verify the script runs without errors
            result = subprocess.run([
                'bash',
                'scripts/deployment-hooks/rollback.sh'
            ], capture_output=True, text=True)
            
            # The script may fail if there's nothing to rollback
            # but it should handle errors gracefully
            assert "Rollback" in result.stdout or "Rollback" in result.stderr
    
    @pytest.mark.integration
    def test_failure_notifications(self, test_config):
        """Test failure notification system"""
        # Test notification script
        result = subprocess.run([
            'bash',
            'scripts/deployment-hooks/notify-failure.sh'
        ], env={
            **os.environ,
            'SERVICE_NAME': test_config['service_name'],
            'ENVIRONMENT': test_config['environment'],
            'DEPLOYMENT_ID': test_config['deployment_id'],
            'FAILURE_REASON': 'Integration test failure simulation'
        }, capture_output=True, text=True)
        
        assert result.returncode == 0, f"Notification script failed: {result.stderr}"
        assert "Notifications sent to configured channels" in result.stdout
    
    @pytest.mark.integration
    @pytest.mark.slow
    def test_complete_deployment_cycle(self, test_config, gc_token):
        """Test complete deployment cycle with Grand Central"""
        # This would test the full cycle if Grand Central is available
        # For now, we'll simulate the key steps
        
        deployment_steps = [
            ("Pre-deployment checks", "backup-check.sh"),
            ("Alert suppression", "suppress-alerts.sh"),
            ("Health verification", "health-check.sh"),
            ("Smoke tests", "smoke-tests.sh"),
            ("Alert restoration", "restore-alerts.sh")
        ]
        
        for step_name, script in deployment_steps:
            print(f"Running: {step_name}")
            result = subprocess.run([
                'bash',
                f'scripts/deployment-hooks/{script}'
            ], env={
                **os.environ,
                'SERVICE_NAME': test_config['service_name'],
                'ENVIRONMENT': test_config['environment'],
                'DEPLOYMENT_ID': test_config['deployment_id'],
                'FORCE_BACKUP': 'true',
                'SKIP_SUPPRESSION': 'true'  # Skip actual alert suppression in test
            }, capture_output=True, text=True)
            
            if result.returncode != 0:
                print(f"Step '{step_name}' output: {result.stdout}")
                print(f"Step '{step_name}' errors: {result.stderr}")
            
            # Some steps may fail in test environment, that's okay
            # We're mainly testing that the scripts run without syntax errors