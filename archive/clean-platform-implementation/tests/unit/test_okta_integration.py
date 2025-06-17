"""Unit tests for Okta integration"""

import pytest
import json
from datetime import datetime, timedelta
from unittest.mock import Mock, patch, MagicMock
from flask import Flask, session
import sys
import os

# Add the services directory to the path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..', '..', 'services'))

from team_access.okta_integration import OktaIntegration, TeamPermissionsManager


class TestOktaIntegration:
    """Test cases for Okta SAML integration"""
    
    @pytest.fixture
    def app(self):
        """Create Flask app for testing"""
        app = Flask(__name__)
        app.config['SECRET_KEY'] = 'test-secret'
        app.config['TESTING'] = True
        return app
    
    @pytest.fixture
    def mock_vault(self):
        """Mock Vault client"""
        with patch('team_access.okta_integration.hvac.Client') as mock:
            vault_instance = Mock()
            vault_instance.read.return_value = {
                'data': {
                    'entity_id': 'test-entity',
                    'idp_metadata_url': 'https://test.okta.com/metadata',
                    'acs_url': 'https://test.nr-ops.net/saml/acs',
                    'allowed_groups': ['platform-team', 'platform-team-readonly'],
                    'x509cert': 'test-cert'
                }
            }
            mock.return_value = vault_instance
            yield vault_instance
    
    @pytest.fixture
    def mock_saml(self):
        """Mock SAML2Manager"""
        with patch('team_access.okta_integration.SAML2Manager') as mock:
            yield mock
    
    @pytest.fixture
    def okta_integration(self, app, mock_vault, mock_saml):
        """Create Okta integration instance"""
        with patch.dict(os.environ, {
            'VAULT_ADDR': 'https://vault.test',
            'VAULT_TOKEN': 'vault-token'
        }):
            integration = OktaIntegration(app)
            integration.init_routes()
            return integration
    
    def test_initialization(self, okta_integration):
        """Test Okta integration initialization"""
        assert okta_integration.app is not None
        assert okta_integration.okta_config['entity_id'] == 'test-entity'
        assert 'platform-team' in okta_integration.okta_config['allowed_groups']
    
    def test_saml_login_route(self, app, okta_integration):
        """Test SAML login initiation"""
        with app.test_client() as client:
            # Mock SAML auth request
            mock_auth_request = Mock()
            mock_auth_request.url = 'https://test.okta.com/sso/saml'
            okta_integration.saml_manager.create_auth_request = Mock(
                return_value=mock_auth_request
            )
            
            response = client.get('/saml/login?next=/dashboard')
            
            assert response.status_code == 302
            assert response.location == 'https://test.okta.com/sso/saml'
            
            with client.session_transaction() as sess:
                assert sess.get('next_url') == '/dashboard'
    
    def test_saml_acs_success(self, app, okta_integration):
        """Test successful SAML assertion consumer service"""
        with app.test_client() as client:
            # Set next URL in session
            with client.session_transaction() as sess:
                sess['next_url'] = '/dashboard'
            
            # Mock successful SAML response
            mock_response = Mock()
            mock_response.is_valid.return_value = True
            mock_response.name_id = 'test.user@newrelic.com'
            mock_response.attributes = {
                'email': ['test.user@newrelic.com'],
                'groups': ['platform-team', 'other-group'],
                'firstName': ['Test'],
                'lastName': ['User']
            }
            
            okta_integration.saml_manager.process_response = Mock(
                return_value=mock_response
            )
            
            response = client.post('/saml/acs')
            
            assert response.status_code == 302
            assert response.location == '/dashboard'
            
            with client.session_transaction() as sess:
                assert sess['authenticated'] is True
                assert sess['user']['username'] == 'test.user@newrelic.com'
                assert 'platform-team' in sess['user']['groups']
    
    def test_saml_acs_unauthorized(self, app, okta_integration):
        """Test SAML ACS with unauthorized user"""
        with app.test_client() as client:
            # Mock SAML response with unauthorized groups
            mock_response = Mock()
            mock_response.is_valid.return_value = True
            mock_response.name_id = 'unauthorized@newrelic.com'
            mock_response.attributes = {
                'email': ['unauthorized@newrelic.com'],
                'groups': ['other-team'],  # Not in allowed groups
                'firstName': ['Unauthorized'],
                'lastName': ['User']
            }
            
            okta_integration.saml_manager.process_response = Mock(
                return_value=mock_response
            )
            
            response = client.post('/saml/acs')
            
            assert response.status_code == 403
            assert b'Access Denied' in response.data
    
    def test_saml_acs_invalid_response(self, app, okta_integration):
        """Test SAML ACS with invalid response"""
        with app.test_client() as client:
            # Mock invalid SAML response
            mock_response = Mock()
            mock_response.is_valid.return_value = False
            
            okta_integration.saml_manager.process_response = Mock(
                return_value=mock_response
            )
            
            response = client.post('/saml/acs')
            
            assert response.status_code == 401
            assert b'Authentication failed' in response.data
    
    def test_require_auth_decorator(self, app, okta_integration):
        """Test authentication decorator"""
        @app.route('/protected')
        @okta_integration.require_auth()
        def protected_route():
            return 'Protected content'
        
        with app.test_client() as client:
            # Test unauthenticated access
            response = client.get('/protected')
            assert response.status_code == 302
            assert '/saml/login' in response.location
            
            # Test authenticated access
            with client.session_transaction() as sess:
                sess['authenticated'] = True
                sess['user'] = {'username': 'test.user', 'groups': ['platform-team']}
                sess['auth_time'] = datetime.utcnow().isoformat()
            
            response = client.get('/protected')
            assert response.status_code == 200
            assert b'Protected content' in response.data
    
    def test_require_auth_with_groups(self, app, okta_integration):
        """Test authentication decorator with group requirements"""
        @app.route('/admin')
        @okta_integration.require_auth(allowed_groups=['platform-team-admin'])
        def admin_route():
            return 'Admin content'
        
        with app.test_client() as client:
            # Test with insufficient permissions
            with client.session_transaction() as sess:
                sess['authenticated'] = True
                sess['user'] = {'username': 'test.user', 'groups': ['platform-team']}
                sess['auth_time'] = datetime.utcnow().isoformat()
            
            response = client.get('/admin')
            assert response.status_code == 403
            
            # Test with correct permissions
            with client.session_transaction() as sess:
                sess['user']['groups'] = ['platform-team-admin']
            
            response = client.get('/admin')
            assert response.status_code == 200
    
    def test_session_expiry(self, app, okta_integration):
        """Test session expiry after 12 hours"""
        @app.route('/protected')
        @okta_integration.require_auth()
        def protected_route():
            return 'Protected content'
        
        with app.test_client() as client:
            # Set expired session
            with client.session_transaction() as sess:
                sess['authenticated'] = True
                sess['user'] = {'username': 'test.user'}
                # Set auth time to 13 hours ago
                old_time = datetime.utcnow() - timedelta(hours=13)
                sess['auth_time'] = old_time.isoformat()
            
            response = client.get('/protected')
            assert response.status_code == 302
            assert '/saml/login' in response.location
    
    def test_get_current_user(self, app, okta_integration):
        """Test getting current user"""
        with app.test_request_context():
            # No user in session
            assert okta_integration.get_current_user() is None
            
            # User in session
            session['authenticated'] = True
            session['user'] = {'username': 'test.user', 'email': 'test@example.com'}
            
            user = okta_integration.get_current_user()
            assert user['username'] == 'test.user'
            assert user['email'] == 'test@example.com'


class TestTeamPermissionsManager:
    """Test cases for team permissions manager"""
    
    @pytest.fixture
    def mock_vault(self):
        """Mock Vault client"""
        with patch('team_access.okta_integration.hvac.Client') as mock:
            vault_instance = Mock()
            mock.return_value = vault_instance
            yield vault_instance
    
    @pytest.fixture
    def permissions_manager(self, mock_vault):
        """Create permissions manager"""
        with patch.dict(os.environ, {
            'VAULT_ADDR': 'https://vault.test',
            'VAULT_TOKEN': 'vault-token'
        }):
            return TeamPermissionsManager()
    
    def test_initialization(self, permissions_manager):
        """Test permissions manager initialization"""
        assert permissions_manager.vault_client is not None
        assert permissions_manager.team_config['team'] == 'platform-team'
        assert permissions_manager.team_config['teamstore_id'] == 12345
    
    def test_check_user_permissions(self, permissions_manager):
        """Test user permission checking"""
        # Mock user teams
        permissions_manager._get_user_teams = Mock(return_value=['platform-team'])
        permissions_manager._check_team_permission = Mock(return_value=True)
        
        result = permissions_manager.check_user_permissions(
            'test.user',
            'vault:secret/teams/platform-team/*',
            'read'
        )
        
        assert result is True
        permissions_manager._get_user_teams.assert_called_with('test.user')
    
    def test_sync_vault_policies(self, permissions_manager):
        """Test Vault policy synchronization"""
        permissions_manager.vault_client.sys.create_or_update_policy = Mock()
        
        permissions_manager.sync_vault_policies()
        
        # Verify policy was created
        permissions_manager.vault_client.sys.create_or_update_policy.assert_called_once()
        call_args = permissions_manager.vault_client.sys.create_or_update_policy.call_args
        
        assert call_args[1]['name'] == 'team-platform-team'
        assert 'secret/teams/platform-team/*' in call_args[1]['policy']
    
    def test_generate_vault_policy(self, permissions_manager):
        """Test Vault policy generation"""
        policies = [
            {
                'path': 'secret/teams/platform-team/*',
                'capabilities': ['create', 'read', 'update', 'delete', 'list']
            },
            {
                'path': 'terraform/platform-team/*',
                'capabilities': ['read', 'list']
            }
        ]
        
        policy_doc = permissions_manager._generate_vault_policy(policies)
        
        assert 'path "secret/teams/platform-team/*"' in policy_doc
        assert 'path "terraform/platform-team/*"' in policy_doc
        assert 'capabilities = ["create", "read", "update", "delete", "list"]' in policy_doc