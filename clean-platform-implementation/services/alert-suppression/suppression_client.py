"""
Alert Suppression Client for Clean Platform Implementation
Integrates with the platform alert suppression service
"""

import logging
import requests
from typing import Dict, List, Optional
from datetime import datetime, timedelta
import json

logger = logging.getLogger(__name__)


class AlertSuppressionClient:
    """Client for managing alert suppressions during deployments and incidents"""
    
    def __init__(self, api_key: str, base_url: str = "https://api.newrelic.com/v2"):
        self.api_key = api_key
        self.base_url = base_url
        self.headers = {
            "Api-Key": api_key,
            "Content-Type": "application/json"
        }
    
    def create_deployment_suppression(
        self,
        service_name: str,
        environment: str,
        duration_minutes: int = 30,
        reason: str = "deployment",
        deployment_id: Optional[str] = None
    ) -> Dict:
        """
        Create alert suppression for deployment
        
        Args:
            service_name: Name of the service being deployed
            environment: Target environment (dev, staging, production)
            duration_minutes: Duration of suppression in minutes
            reason: Reason for suppression
            deployment_id: Optional deployment tracking ID
        
        Returns:
            Suppression details including ID
        """
        start_time = datetime.utcnow()
        end_time = start_time + timedelta(minutes=duration_minutes)
        
        payload = {
            "suppression": {
                "name": f"{service_name} - {reason} - {environment}",
                "description": f"Alert suppression for {service_name} {reason} in {environment}",
                "start_time": start_time.isoformat() + "Z",
                "end_time": end_time.isoformat() + "Z",
                "filter": {
                    "type": "NRQL",
                    "query": f"appName = '{service_name}' AND environment = '{environment}'"
                },
                "metadata": {
                    "service": service_name,
                    "environment": environment,
                    "reason": reason,
                    "deployment_id": deployment_id,
                    "created_by": "clean-platform-automation"
                }
            }
        }
        
        try:
            response = requests.post(
                f"{self.base_url}/suppressions",
                headers=self.headers,
                json=payload
            )
            response.raise_for_status()
            
            result = response.json()
            logger.info(
                f"Created alert suppression for {service_name} in {environment}. "
                f"ID: {result.get('id')}, Duration: {duration_minutes} minutes"
            )
            return result
            
        except requests.exceptions.RequestException as e:
            logger.error(f"Failed to create alert suppression: {e}")
            raise
    
    def create_incident_suppression(
        self,
        incident_id: str,
        duration_minutes: int = 60,
        reason: str = "incident_response"
    ) -> Dict:
        """
        Create alert suppression during incident response
        
        Args:
            incident_id: ID of the incident being handled
            duration_minutes: Duration of suppression
            reason: Reason for suppression
        
        Returns:
            Suppression details
        """
        payload = {
            "suppression": {
                "type": "incident",
                "incident_id": incident_id,
                "duration_minutes": duration_minutes,
                "reason": reason,
                "metadata": {
                    "created_by": "emergency-room",
                    "incident_id": incident_id
                }
            }
        }
        
        try:
            response = requests.post(
                f"{self.base_url}/incident_suppressions",
                headers=self.headers,
                json=payload
            )
            response.raise_for_status()
            
            result = response.json()
            logger.info(f"Created incident suppression for {incident_id}")
            return result
            
        except requests.exceptions.RequestException as e:
            logger.error(f"Failed to create incident suppression: {e}")
            raise
    
    def update_suppression(self, suppression_id: str, extend_minutes: int) -> Dict:
        """
        Extend an existing suppression
        
        Args:
            suppression_id: ID of the suppression to extend
            extend_minutes: Additional minutes to extend
        
        Returns:
            Updated suppression details
        """
        payload = {
            "extend_minutes": extend_minutes
        }
        
        try:
            response = requests.patch(
                f"{self.base_url}/suppressions/{suppression_id}",
                headers=self.headers,
                json=payload
            )
            response.raise_for_status()
            
            result = response.json()
            logger.info(f"Extended suppression {suppression_id} by {extend_minutes} minutes")
            return result
            
        except requests.exceptions.RequestException as e:
            logger.error(f"Failed to update suppression: {e}")
            raise
    
    def delete_suppression(self, suppression_id: str) -> None:
        """
        Delete (restore) an alert suppression
        
        Args:
            suppression_id: ID of the suppression to delete
        """
        try:
            response = requests.delete(
                f"{self.base_url}/suppressions/{suppression_id}",
                headers=self.headers
            )
            response.raise_for_status()
            
            logger.info(f"Deleted suppression {suppression_id}")
            
        except requests.exceptions.RequestException as e:
            logger.error(f"Failed to delete suppression: {e}")
            raise
    
    def list_active_suppressions(self, service_name: Optional[str] = None) -> List[Dict]:
        """
        List active suppressions, optionally filtered by service
        
        Args:
            service_name: Optional service name to filter by
        
        Returns:
            List of active suppressions
        """
        params = {
            "filter[active]": "true"
        }
        
        if service_name:
            params["filter[service]"] = service_name
        
        try:
            response = requests.get(
                f"{self.base_url}/suppressions",
                headers=self.headers,
                params=params
            )
            response.raise_for_status()
            
            return response.json().get("suppressions", [])
            
        except requests.exceptions.RequestException as e:
            logger.error(f"Failed to list suppressions: {e}")
            raise
    
    def create_maintenance_window(
        self,
        service_name: str,
        start_time: datetime,
        end_time: datetime,
        reason: str = "scheduled_maintenance"
    ) -> Dict:
        """
        Create a maintenance window suppression
        
        Args:
            service_name: Service under maintenance
            start_time: Start of maintenance window
            end_time: End of maintenance window
            reason: Reason for maintenance
        
        Returns:
            Suppression details
        """
        payload = {
            "suppression": {
                "name": f"{service_name} - Maintenance Window",
                "description": f"Scheduled maintenance for {service_name}",
                "start_time": start_time.isoformat() + "Z",
                "end_time": end_time.isoformat() + "Z",
                "filter": {
                    "type": "NRQL",
                    "query": f"appName = '{service_name}'"
                },
                "metadata": {
                    "service": service_name,
                    "reason": reason,
                    "type": "maintenance_window"
                }
            }
        }
        
        try:
            response = requests.post(
                f"{self.base_url}/suppressions",
                headers=self.headers,
                json=payload
            )
            response.raise_for_status()
            
            result = response.json()
            logger.info(
                f"Created maintenance window for {service_name} "
                f"from {start_time} to {end_time}"
            )
            return result
            
        except requests.exceptions.RequestException as e:
            logger.error(f"Failed to create maintenance window: {e}")
            raise


# Integration with deployment hooks
class DeploymentSuppressionManager:
    """Manages alert suppressions during deployments"""
    
    def __init__(self, client: AlertSuppressionClient):
        self.client = client
        self.active_suppressions = {}
    
    def suppress_for_deployment(
        self,
        service_name: str,
        environment: str,
        deployment_id: str
    ) -> str:
        """
        Create suppression for deployment
        
        Returns:
            Suppression ID
        """
        suppression = self.client.create_deployment_suppression(
            service_name=service_name,
            environment=environment,
            deployment_id=deployment_id
        )
        
        suppression_id = suppression["id"]
        self.active_suppressions[deployment_id] = suppression_id
        
        return suppression_id
    
    def restore_after_deployment(self, deployment_id: str) -> None:
        """
        Restore alerts after deployment
        
        Args:
            deployment_id: Deployment tracking ID
        """
        if deployment_id in self.active_suppressions:
            suppression_id = self.active_suppressions[deployment_id]
            self.client.delete_suppression(suppression_id)
            del self.active_suppressions[deployment_id]
    
    def extend_for_rollback(self, deployment_id: str, additional_minutes: int = 30) -> None:
        """
        Extend suppression during rollback
        
        Args:
            deployment_id: Deployment tracking ID
            additional_minutes: Additional time needed
        """
        if deployment_id in self.active_suppressions:
            suppression_id = self.active_suppressions[deployment_id]
            self.client.update_suppression(suppression_id, additional_minutes)


if __name__ == "__main__":
    # Example usage
    import os
    
    api_key = os.environ.get("NEW_RELIC_API_KEY")
    client = AlertSuppressionClient(api_key)
    
    # Create deployment suppression
    suppression = client.create_deployment_suppression(
        service_name="clean-platform",
        environment="production",
        duration_minutes=30,
        deployment_id="deploy-123"
    )
    
    print(f"Created suppression: {json.dumps(suppression, indent=2)}")