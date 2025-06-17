"""Unit tests for Grand Central API client"""

import pytest
import json
from unittest.mock import Mock, patch, MagicMock
import requests
from requests.exceptions import RequestException
import sys
import os

# Add the services directory to the path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..', '..', 'services'))

from grand_central.gc_client import GrandCentralClient, DeploymentManager


class TestGrandCentralClient:
    """Test cases for Grand Central API client"""
    
    @pytest.fixture
    def mock_vault(self):
        """Mock Vault client"""
        with patch('grand_central.gc_client.hvac.Client') as mock:
            vault_instance = Mock()
            vault_instance.read.return_value = {'data': {'token': 'test-token'}}
            mock.return_value = vault_instance
            yield vault_instance
    
    @pytest.fixture
    def gc_client(self, mock_vault):
        """Create Grand Central client with mocked dependencies"""
        with patch.dict(os.environ, {
            'VAULT_ADDR': 'https://vault.test',
            'VAULT_TOKEN': 'vault-token'
        }):
            client = GrandCentralClient(environment='staging')
            return client
    
    def test_client_initialization(self, gc_client):
        """Test client initialization"""
        assert gc_client.environment == 'staging'
        assert gc_client.base_url == 'https://staging-grand-central.nr-ops.net'
        assert gc_client.auth_token == 'test-token'
    
    def test_register_project(self, gc_client):
        """Test project registration"""
        expected_response = {'id': 'project-123', 'status': 'registered'}
        gc_client.session.post = Mock(return_value=Mock(
            json=Mock(return_value=expected_response),
            raise_for_status=Mock()
        ))
        
        result = gc_client.register_project('platform-team', 'clean-platform')
        
        assert result == expected_response
        gc_client.session.post.assert_called_once()
        call_args = gc_client.session.post.call_args
        assert call_args[0][0] == 'https://staging-grand-central.nr-ops.net/api/v1/register'
        assert call_args[1]['json']['org'] == 'platform-team'
        assert call_args[1]['json']['repo'] == 'clean-platform'
    
    def test_deploy(self, gc_client):
        """Test deployment creation"""
        expected_response = {'id': 'deploy-123', 'status': 'pending'}
        gc_client.session.post = Mock(return_value=Mock(
            json=Mock(return_value=expected_response),
            raise_for_status=Mock()
        ))
        
        result = gc_client.deploy(
            project_org='platform-team',
            project_repo='clean-platform',
            environment_name='staging',
            version='v1.2.3'
        )
        
        assert result == expected_response
        assert result['id'] == 'deploy-123'
    
    def test_deploy_with_options(self, gc_client):
        """Test deployment with optional parameters"""
        gc_client.session.post = Mock(return_value=Mock(
            json=Mock(return_value={'id': 'deploy-456'}),
            raise_for_status=Mock()
        ))
        
        gc_client.deploy(
            project_org='platform-team',
            project_repo='clean-platform',
            environment_name='production',
            version='v1.2.3',
            skipCanary=True,
            skipHealthCheck=True
        )
        
        call_args = gc_client.session.post.call_args[1]['json']
        assert call_args['skipCanary'] is True
        assert call_args['skipHealthCheck'] is True
    
    def test_get_deployment_status(self, gc_client):
        """Test getting deployment status"""
        expected_status = {
            'id': 'deploy-123',
            'status': 'completed',
            'version': 'v1.2.3'
        }
        gc_client.session.get = Mock(return_value=Mock(
            json=Mock(return_value=expected_status),
            raise_for_status=Mock()
        ))
        
        result = gc_client.get_deployment_status('deploy-123')
        
        assert result == expected_status
        gc_client.session.get.assert_called_with(
            'https://staging-grand-central.nr-ops.net/api/v1/deploy/deploy-123'
        )
    
    def test_wait_for_deployment_success(self, gc_client):
        """Test waiting for deployment to complete"""
        # Mock status progression
        statuses = [
            {'status': 'pending'},
            {'status': 'deploying'},
            {'status': 'completed'}
        ]
        gc_client.get_deployment_status = Mock(side_effect=statuses)
        
        with patch('time.sleep'):  # Speed up test
            result = gc_client.wait_for_deployment('deploy-123', timeout=60)
        
        assert result['status'] == 'completed'
        assert gc_client.get_deployment_status.call_count == 3
    
    def test_wait_for_deployment_failure(self, gc_client):
        """Test deployment failure during wait"""
        gc_client.get_deployment_status = Mock(return_value={'status': 'failed'})
        
        with pytest.raises(Exception) as exc_info:
            gc_client.wait_for_deployment('deploy-123')
        
        assert 'Deployment failed' in str(exc_info.value)
    
    def test_wait_for_deployment_timeout(self, gc_client):
        """Test deployment timeout"""
        gc_client.get_deployment_status = Mock(return_value={'status': 'deploying'})
        
        with patch('time.sleep'):  # Speed up test
            with pytest.raises(TimeoutError):
                gc_client.wait_for_deployment('deploy-123', timeout=1)
    
    def test_rollback_deployment(self, gc_client):
        """Test deployment rollback"""
        expected_response = {'id': 'rollback-123', 'status': 'pending'}
        gc_client.session.post = Mock(return_value=Mock(
            json=Mock(return_value=expected_response),
            raise_for_status=Mock()
        ))
        
        result = gc_client.rollback_deployment('deploy-123', 'Test failure')
        
        assert result == expected_response
        call_args = gc_client.session.post.call_args
        assert 'deploy-123/rollback' in call_args[0][0]
        assert call_args[1]['json']['reason'] == 'Test failure'
    
    def test_create_change_record(self, gc_client):
        """Test change record creation"""
        expected_response = {'id': 'change-123', 'status': 'created'}
        gc_client.session.post = Mock(return_value=Mock(
            json=Mock(return_value=expected_response),
            raise_for_status=Mock()
        ))
        
        result = gc_client.create_change_record(
            description='Deploy v1.2.3',
            team_name='platform-team',
            deployment_id='deploy-123'
        )
        
        assert result == expected_response
        call_args = gc_client.session.post.call_args[1]['json']
        assert call_args['description'] == 'Deploy v1.2.3'
        assert call_args['type'] == 'deployment'
    
    def test_error_handling(self, gc_client):
        """Test error handling for API failures"""
        gc_client.session.post = Mock(side_effect=RequestException("API Error"))
        
        with pytest.raises(RequestException):
            gc_client.deploy(
                project_org='platform-team',
                project_repo='clean-platform',
                environment_name='staging',
                version='v1.2.3'
            )


class TestDeploymentManager:
    """Test cases for deployment manager"""
    
    @pytest.fixture
    def mock_gc_client(self):
        """Mock Grand Central client"""
        return Mock(spec=GrandCentralClient)
    
    @pytest.fixture
    def deployment_manager(self, mock_gc_client):
        """Create deployment manager"""
        return DeploymentManager(mock_gc_client)
    
    def test_deploy_with_canary_success(self, deployment_manager, mock_gc_client):
        """Test successful deployment with canary"""
        # Mock successful validation
        mock_gc_client.validate_deployment.return_value = {'valid': True}
        
        # Mock deployment creation
        mock_gc_client.deploy.return_value = {'id': 'deploy-123'}
        
        # Mock successful wait
        mock_gc_client.wait_for_deployment.return_value = {
            'id': 'deploy-123',
            'status': 'completed'
        }
        
        # Mock change record creation
        mock_gc_client.create_change_record.return_value = {'id': 'change-123'}
        
        with patch('time.sleep'):  # Speed up test
            result = deployment_manager.deploy_with_canary(
                org='platform-team',
                repo='clean-platform',
                environment='production',
                version='v1.2.3',
                canary_duration=5
            )
        
        assert result == 'deploy-123'
        assert mock_gc_client.validate_deployment.called
        assert mock_gc_client.deploy.called
        assert mock_gc_client.wait_for_deployment.called
        assert mock_gc_client.create_change_record.call_count == 2
    
    def test_deploy_with_canary_validation_failure(self, deployment_manager, mock_gc_client):
        """Test deployment with validation failure"""
        mock_gc_client.validate_deployment.return_value = {
            'valid': False,
            'errors': ['Missing required configuration']
        }
        
        with pytest.raises(ValueError) as exc_info:
            deployment_manager.deploy_with_canary(
                org='platform-team',
                repo='clean-platform',
                environment='production',
                version='v1.2.3'
            )
        
        assert 'Deployment validation failed' in str(exc_info.value)
        assert not mock_gc_client.deploy.called
    
    def test_deploy_with_canary_rollback(self, deployment_manager, mock_gc_client):
        """Test deployment rollback on failure"""
        mock_gc_client.validate_deployment.return_value = {'valid': True}
        mock_gc_client.deploy.return_value = {'id': 'deploy-123'}
        mock_gc_client.wait_for_deployment.side_effect = Exception("Deployment failed")
        
        with patch('time.sleep'):
            with pytest.raises(Exception):
                deployment_manager.deploy_with_canary(
                    org='platform-team',
                    repo='clean-platform',
                    environment='production',
                    version='v1.2.3',
                    canary_duration=5
                )
        
        # Verify rollback was called
        mock_gc_client.rollback_deployment.assert_called_with(
            'deploy-123',
            'Deployment failed'
        )