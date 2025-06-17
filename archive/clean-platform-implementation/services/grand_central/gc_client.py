"""
Grand Central API Client
Handles project registration, deployments, and lifecycle management
"""

import os
import json
import time
import logging
from typing import Dict, Optional, List, Any
from datetime import datetime
import requests
from requests.exceptions import RequestException
import hvac
from tenacity import retry, stop_after_attempt, wait_exponential

logger = logging.getLogger(__name__)

class GrandCentralClient:
    """Client for interacting with Grand Central API"""
    
    def __init__(self, environment: str = "production"):
        self.environment = environment
        self.base_url = self._get_base_url()
        self.auth_token = self._get_auth_token()
        self.session = self._create_session()
        
    def _get_base_url(self) -> str:
        """Get Grand Central API URL based on environment"""
        urls = {
            "staging": "https://staging-grand-central.nr-ops.net",
            "production": "https://grand-central.nr-ops.net"
        }
        return urls.get(self.environment, urls["production"])
    
    def _get_auth_token(self) -> str:
        """Retrieve Grand Central auth token from Vault"""
        vault_client = hvac.Client(url=os.getenv('VAULT_ADDR'))
        vault_client.token = os.getenv('VAULT_TOKEN')
        
        secret_path = os.getenv(
            'GC_TOKEN_PATH',
            'secret/teams/platform-team/grand-central-token'
        )
        
        try:
            response = vault_client.read(secret_path)
            return response['data']['token']
        except Exception as e:
            logger.error(f"Failed to retrieve Grand Central token: {e}")
            raise
    
    def _create_session(self) -> requests.Session:
        """Create HTTP session with auth headers"""
        session = requests.Session()
        session.headers.update({
            'X-Grand-Central-Auth': self.auth_token,
            'Content-Type': 'application/json',
            'Accept': 'application/json'
        })
        return session
    
    @retry(stop=stop_after_attempt(3), wait=wait_exponential(multiplier=1, min=4, max=10))
    def register_project(self, org: str, repo: str, path: Optional[str] = None,
                        bypass_master_build: bool = False) -> Dict[str, Any]:
        """
        Register a project with Grand Central
        
        POST /api/v1/register
        """
        payload = {
            "org": org,
            "repo": repo
        }
        
        if path:
            payload["path"] = path
        if bypass_master_build:
            payload["bypassMasterBuild"] = bypass_master_build
            
        try:
            response = self.session.post(
                f"{self.base_url}/api/v1/register",
                json=payload
            )
            response.raise_for_status()
            
            result = response.json()
            logger.info(f"Successfully registered project: {org}/{repo}")
            return result
            
        except RequestException as e:
            logger.error(f"Failed to register project: {e}")
            raise
    
    @retry(stop=stop_after_attempt(3), wait=wait_exponential(multiplier=1, min=4, max=10))
    def deploy(self, project_org: str, project_repo: str, environment_name: str,
              version: str, configuration_version: Optional[str] = None,
              **kwargs) -> Dict[str, Any]:
        """
        Create a deployment
        
        POST /api/v1/deploy
        """
        payload = {
            "projectOrg": project_org,
            "projectRepo": project_repo,
            "environmentName": environment_name,
            "version": version
        }
        
        if configuration_version:
            payload["configurationVersion"] = configuration_version
            
        # Add optional parameters
        optional_params = [
            'skipCanary', 'skipGatekeeper', 'skipHealthCheck',
            'forceDeployWithoutCert', 'deploymentIssue'
        ]
        
        for param in optional_params:
            if param in kwargs:
                payload[param] = kwargs[param]
                
        try:
            response = self.session.post(
                f"{self.base_url}/api/v1/deploy",
                json=payload
            )
            response.raise_for_status()
            
            result = response.json()
            logger.info(f"Deployment started: {result.get('id')}")
            return result
            
        except RequestException as e:
            logger.error(f"Failed to create deployment: {e}")
            raise
    
    def get_deployment_status(self, deployment_id: str) -> Dict[str, Any]:
        """
        Get deployment status
        
        GET /api/v1/deploy/{deploymentID}
        """
        try:
            response = self.session.get(
                f"{self.base_url}/api/v1/deploy/{deployment_id}"
            )
            response.raise_for_status()
            return response.json()
            
        except RequestException as e:
            logger.error(f"Failed to get deployment status: {e}")
            raise
    
    def wait_for_deployment(self, deployment_id: str, timeout: int = 3600) -> Dict[str, Any]:
        """Wait for deployment to complete"""
        start_time = time.time()
        
        while time.time() - start_time < timeout:
            status = self.get_deployment_status(deployment_id)
            deployment_state = status.get('status', 'unknown')
            
            if deployment_state == 'completed':
                logger.info(f"Deployment {deployment_id} completed successfully")
                return status
            elif deployment_state in ['failed', 'rolled_back']:
                logger.error(f"Deployment {deployment_id} failed: {deployment_state}")
                raise Exception(f"Deployment failed with status: {deployment_state}")
            
            logger.info(f"Deployment {deployment_id} status: {deployment_state}")
            time.sleep(30)
        
        raise TimeoutError(f"Deployment {deployment_id} timed out after {timeout} seconds")
    
    def rollback_deployment(self, deployment_id: str, reason: str) -> Dict[str, Any]:
        """
        Rollback a deployment
        
        POST /api/v1/deploy/{deploymentID}/rollback
        """
        payload = {
            "reason": reason
        }
        
        try:
            response = self.session.post(
                f"{self.base_url}/api/v1/deploy/{deployment_id}/rollback",
                json=payload
            )
            response.raise_for_status()
            
            result = response.json()
            logger.info(f"Rollback initiated for deployment: {deployment_id}")
            return result
            
        except RequestException as e:
            logger.error(f"Failed to rollback deployment: {e}")
            raise
    
    def get_project_environments(self, org: str, repo: str) -> List[Dict[str, Any]]:
        """
        Get project environments
        
        GET /api/v1/projects/{org}/{repo}/environments
        """
        try:
            response = self.session.get(
                f"{self.base_url}/api/v1/projects/{org}/{repo}/environments"
            )
            response.raise_for_status()
            return response.json()
            
        except RequestException as e:
            logger.error(f"Failed to get project environments: {e}")
            raise
    
    def create_change_record(self, description: str, team_name: str,
                           deployment_id: Optional[str] = None) -> Dict[str, Any]:
        """
        Create a change record for tracking
        
        POST /api/v1/change_record
        """
        payload = {
            "description": description,
            "type": "deployment",
            "teamName": team_name,
            "timestamp": datetime.utcnow().isoformat(),
            "payload": {}
        }
        
        if deployment_id:
            payload["payload"]["deploymentId"] = deployment_id
            
        try:
            response = self.session.post(
                f"{self.base_url}/api/v1/change_record",
                json=payload
            )
            response.raise_for_status()
            
            result = response.json()
            logger.info(f"Change record created: {result.get('id')}")
            return result
            
        except RequestException as e:
            logger.error(f"Failed to create change record: {e}")
            raise
    
    def get_deployment_history(self, org: str, repo: str, 
                             environment: Optional[str] = None,
                             limit: int = 20) -> List[Dict[str, Any]]:
        """
        Get deployment history for a project
        
        GET /api/v1/projects/{org}/{repo}/deployments
        """
        params = {"limit": limit}
        if environment:
            params["environment"] = environment
            
        try:
            response = self.session.get(
                f"{self.base_url}/api/v1/projects/{org}/{repo}/deployments",
                params=params
            )
            response.raise_for_status()
            return response.json()
            
        except RequestException as e:
            logger.error(f"Failed to get deployment history: {e}")
            raise
    
    def validate_deployment(self, org: str, repo: str, environment: str,
                          version: str) -> Dict[str, Any]:
        """
        Validate a deployment before executing
        
        POST /api/v1/validate_deployment
        """
        payload = {
            "projectOrg": org,
            "projectRepo": repo,
            "environmentName": environment,
            "version": version
        }
        
        try:
            response = self.session.post(
                f"{self.base_url}/api/v1/validate_deployment",
                json=payload
            )
            response.raise_for_status()
            
            result = response.json()
            logger.info(f"Deployment validation result: {result.get('valid', False)}")
            return result
            
        except RequestException as e:
            logger.error(f"Failed to validate deployment: {e}")
            raise


class DeploymentManager:
    """High-level deployment orchestration using Grand Central"""
    
    def __init__(self, gc_client: GrandCentralClient):
        self.gc_client = gc_client
        
    def deploy_with_canary(self, org: str, repo: str, environment: str, 
                          version: str, canary_duration: int = 300) -> str:
        """Deploy with canary phase"""
        # First validate deployment
        validation = self.gc_client.validate_deployment(org, repo, environment, version)
        if not validation.get('valid', False):
            raise ValueError(f"Deployment validation failed: {validation.get('errors', [])}")
        
        # Create change record
        change_record = self.gc_client.create_change_record(
            description=f"Deploying {org}/{repo} version {version} to {environment}",
            team_name="platform-team"
        )
        
        # Start deployment
        deployment = self.gc_client.deploy(
            project_org=org,
            project_repo=repo,
            environment_name=environment,
            version=version
        )
        
        deployment_id = deployment['id']
        
        try:
            # Wait for canary phase
            logger.info(f"Waiting {canary_duration} seconds for canary phase...")
            time.sleep(canary_duration)
            
            # Check canary metrics
            # This would integrate with New Relic APM verification
            
            # Complete deployment
            result = self.gc_client.wait_for_deployment(deployment_id)
            
            # Update change record with success
            self.gc_client.create_change_record(
                description=f"Deployment {deployment_id} completed successfully",
                team_name="platform-team",
                deployment_id=deployment_id
            )
            
            return deployment_id
            
        except Exception as e:
            logger.error(f"Deployment failed, initiating rollback: {e}")
            self.gc_client.rollback_deployment(deployment_id, str(e))
            raise