"""
Okta Integration for Team Access Control
Implements NR-Prod Okta SAML authentication and authorization
"""

import os
import json
import logging
from typing import Dict, List, Optional, Any
from datetime import datetime, timedelta
import requests
from flask import Flask, request, redirect, session, url_for
from flask_saml2 import SAML2Manager
import hvac
from functools import wraps

logger = logging.getLogger(__name__)

class OktaIntegration:
    """Okta SAML integration for production access"""
    
    def __init__(self, app: Flask):
        self.app = app
        self.vault_client = self._init_vault()
        self.okta_config = self._load_okta_config()
        self.saml_manager = self._init_saml()
        
    def _init_vault(self) -> hvac.Client:
        """Initialize Vault client for secrets"""
        vault_client = hvac.Client(url=os.getenv('VAULT_ADDR'))
        vault_client.token = os.getenv('VAULT_TOKEN')
        return vault_client
    
    def _load_okta_config(self) -> Dict[str, Any]:
        """Load Okta configuration from Vault"""
        secret_path = 'secret/teams/platform-team/okta-config'
        
        try:
            response = self.vault_client.read(secret_path)
            return response['data']
        except Exception as e:
            logger.error(f"Failed to load Okta config: {e}")
            # Return default config for development
            return {
                'entity_id': 'platform-team-services',
                'idp_metadata_url': 'https://nr-prod.okta.com/app/exk1fxpho5gSIpXDg0h8/sso/saml/metadata',
                'acs_url': 'https://platform.nr-ops.net/saml/acs',
                'allowed_groups': ['platform-team', 'platform-team-readonly']
            }
    
    def _init_saml(self) -> SAML2Manager:
        """Initialize SAML2 manager"""
        saml_config = {
            'SECRET_KEY': os.urandom(32),
            'SAML2_ENTITY_ID': self.okta_config['entity_id'],
            'SAML2_IDP': {
                'default': {
                    'entity_id': 'http://www.okta.com/exk1fxpho5gSIpXDg0h8',
                    'sso_url': 'https://nr-prod.okta.com/app/nr-prod_platformteamservices_1/exk1fxpho5gSIpXDg0h8/sso/saml',
                    'slo_url': 'https://nr-prod.okta.com/app/nr-prod_platformteamservices_1/exk1fxpho5gSIpXDg0h8/slo/saml',
                    'x509cert': self._get_okta_certificate(),
                    'metadata_url': self.okta_config['idp_metadata_url']
                }
            },
            'SAML2_SERVICE_PROVIDER': {
                'entity_id': self.okta_config['entity_id'],
                'acs_url': self.okta_config['acs_url'],
                'certificate': self._get_sp_certificate(),
                'private_key': self._get_sp_private_key()
            }
        }
        
        self.app.config.update(saml_config)
        return SAML2Manager(self.app)
    
    def _get_okta_certificate(self) -> str:
        """Retrieve Okta X.509 certificate"""
        try:
            response = requests.get(self.okta_config['idp_metadata_url'])
            # Parse certificate from metadata (simplified)
            # In production, use proper XML parsing
            return self.okta_config.get('x509cert', '')
        except Exception as e:
            logger.error(f"Failed to get Okta certificate: {e}")
            return ''
    
    def _get_sp_certificate(self) -> str:
        """Get service provider certificate from Vault"""
        try:
            response = self.vault_client.read('secret/teams/platform-team/saml-sp-cert')
            return response['data']['certificate']
        except Exception:
            # Generate self-signed for development
            return ''
    
    def _get_sp_private_key(self) -> str:
        """Get service provider private key from Vault"""
        try:
            response = self.vault_client.read('secret/teams/platform-team/saml-sp-key')
            return response['data']['private_key']
        except Exception:
            # Generate for development
            return ''
    
    def init_routes(self):
        """Initialize SAML routes"""
        
        @self.app.route('/saml/login')
        def saml_login():
            """Initiate SAML login"""
            next_url = request.args.get('next', '/')
            session['next_url'] = next_url
            
            # Create SAML auth request
            auth_request = self.saml_manager.create_auth_request('default')
            return redirect(auth_request.url)
        
        @self.app.route('/saml/acs', methods=['POST'])
        def saml_acs():
            """SAML Assertion Consumer Service"""
            try:
                # Process SAML response
                auth_response = self.saml_manager.process_response()
                
                if auth_response.is_valid():
                    # Extract user info
                    user_info = {
                        'username': auth_response.name_id,
                        'email': auth_response.attributes.get('email', [''])[0],
                        'groups': auth_response.attributes.get('groups', []),
                        'first_name': auth_response.attributes.get('firstName', [''])[0],
                        'last_name': auth_response.attributes.get('lastName', [''])[0]
                    }
                    
                    # Check group membership
                    allowed = any(group in user_info['groups'] 
                                for group in self.okta_config['allowed_groups'])
                    
                    if not allowed:
                        logger.warning(f"Access denied for user {user_info['username']}: "
                                     f"not in allowed groups")
                        return "Access Denied: Not authorized for this application", 403
                    
                    # Create session
                    session['user'] = user_info
                    session['authenticated'] = True
                    session['auth_time'] = datetime.utcnow().isoformat()
                    
                    # Log successful authentication
                    logger.info(f"User {user_info['username']} authenticated successfully")
                    
                    # Redirect to original URL
                    next_url = session.pop('next_url', '/')
                    return redirect(next_url)
                else:
                    logger.error("Invalid SAML response")
                    return "Authentication failed", 401
                    
            except Exception as e:
                logger.error(f"SAML ACS error: {e}")
                return "Authentication error", 500
        
        @self.app.route('/saml/logout')
        def saml_logout():
            """Initiate SAML logout"""
            session.clear()
            
            # Create SAML logout request
            logout_request = self.saml_manager.create_logout_request('default')
            return redirect(logout_request.url)
        
        @self.app.route('/saml/sls', methods=['POST'])
        def saml_sls():
            """SAML Single Logout Service"""
            try:
                # Process logout response
                self.saml_manager.process_logout_response()
                session.clear()
                return redirect('/')
            except Exception as e:
                logger.error(f"SAML SLS error: {e}")
                return "Logout error", 500
    
    def require_auth(self, allowed_groups: Optional[List[str]] = None):
        """Decorator to require authentication"""
        def decorator(f):
            @wraps(f)
            def wrapped(*args, **kwargs):
                if not session.get('authenticated'):
                    return redirect(url_for('saml_login', next=request.url))
                
                # Check session expiry (12 hours)
                auth_time = datetime.fromisoformat(session.get('auth_time', ''))
                if datetime.utcnow() - auth_time > timedelta(hours=12):
                    session.clear()
                    return redirect(url_for('saml_login', next=request.url))
                
                # Check group membership if specified
                if allowed_groups:
                    user_groups = session.get('user', {}).get('groups', [])
                    if not any(group in user_groups for group in allowed_groups):
                        return "Access Denied: Insufficient permissions", 403
                
                return f(*args, **kwargs)
            return wrapped
        return decorator
    
    def get_current_user(self) -> Optional[Dict[str, Any]]:
        """Get current authenticated user"""
        if session.get('authenticated'):
            return session.get('user')
        return None


class TeamPermissionsManager:
    """Manage team permissions and access control"""
    
    def __init__(self):
        self.vault_client = self._init_vault()
        self.team_config = self._load_team_config()
        
    def _init_vault(self) -> hvac.Client:
        """Initialize Vault client"""
        vault_client = hvac.Client(url=os.getenv('VAULT_ADDR'))
        vault_client.token = os.getenv('VAULT_TOKEN')
        return vault_client
    
    def _load_team_config(self) -> Dict[str, Any]:
        """Load team permissions configuration"""
        # This would normally load from team-permissions repository
        # For now, return the configuration
        return {
            'team': 'platform-team',
            'teamstore_id': 12345,  # Real TeamStore ID
            'permissions': {
                'aws': {
                    'iam': [{
                        'accounts': {
                            'ids': ['895102219545', '392988681574']
                        },
                        'roles': {
                            'predefined': ['NRReadOnly']
                        }
                    }]
                },
                'vault': {
                    'policies': [
                        {
                            'path': 'secret/teams/platform-team/*',
                            'capabilities': ['create', 'read', 'update', 'delete', 'list']
                        },
                        {
                            'path': 'terraform/platform-team/*',
                            'capabilities': ['read', 'list']
                        }
                    ]
                },
                'production_access': {
                    'vpn': True,
                    'okta_groups': ['platform-team', 'platform-team-readonly']
                },
                'pagerduty': {
                    'enabled': True,
                    'service_id': 'P123456'
                }
            }
        }
    
    def check_user_permissions(self, username: str, resource: str, 
                              action: str) -> bool:
        """Check if user has permission for resource/action"""
        # Get user's team membership from TeamStore
        user_teams = self._get_user_teams(username)
        
        # Check each team's permissions
        for team in user_teams:
            if self._check_team_permission(team, resource, action):
                return True
        
        return False
    
    def _get_user_teams(self, username: str) -> List[str]:
        """Get user's teams from TeamStore"""
        # This would integrate with TeamStore API
        # For now, return mock data
        return ['platform-team']
    
    def _check_team_permission(self, team: str, resource: str, 
                              action: str) -> bool:
        """Check if team has permission"""
        # Implement permission checking logic
        # This would check against team-permissions configuration
        return True
    
    def sync_vault_policies(self):
        """Sync team Vault policies"""
        team_policies = self.team_config['permissions']['vault']['policies']
        
        policy_name = f"team-{self.team_config['team']}"
        policy_document = self._generate_vault_policy(team_policies)
        
        try:
            self.vault_client.sys.create_or_update_policy(
                name=policy_name,
                policy=policy_document
            )
            logger.info(f"Updated Vault policy: {policy_name}")
        except Exception as e:
            logger.error(f"Failed to update Vault policy: {e}")
    
    def _generate_vault_policy(self, policies: List[Dict]) -> str:
        """Generate Vault policy document"""
        policy_rules = []
        
        for policy in policies:
            rule = f'path "{policy["path"]}" {{\n'
            rule += f'  capabilities = {json.dumps(policy["capabilities"])}\n'
            rule += '}'
            policy_rules.append(rule)
        
        return '\n\n'.join(policy_rules)
    
    def create_aws_iam_role(self):
        """Create AWS IAM role for team"""
        # This would integrate with AWS IAM
        # Creating cross-account assume role policies
        pass
    
    def sync_okta_groups(self):
        """Sync Okta group membership"""
        # This would integrate with Okta API
        # to manage group membership
        pass


# Usage example
def create_app():
    """Create Flask app with Okta integration"""
    app = Flask(__name__)
    app.secret_key = os.urandom(32)
    
    # Initialize Okta integration
    okta = OktaIntegration(app)
    okta.init_routes()
    
    # Initialize permissions manager
    permissions = TeamPermissionsManager()
    
    @app.route('/')
    def index():
        user = okta.get_current_user()
        if user:
            return f"Welcome {user['first_name']} {user['last_name']}"
        return "Please login"
    
    @app.route('/api/data')
    @okta.require_auth(allowed_groups=['platform-team'])
    def api_data():
        return {"data": "sensitive information"}
    
    return app, okta, permissions