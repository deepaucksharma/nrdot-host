"""
Feature Flags Client for Clean Platform Implementation
Integrates with the platform feature flags service
"""

import logging
import requests
from typing import Dict, List, Optional, Any, Union
from enum import Enum
import json
from functools import lru_cache
from datetime import datetime, timedelta

logger = logging.getLogger(__name__)


class RolloutStrategy(Enum):
    """Feature rollout strategies"""
    PERCENTAGE = "percentage"
    USER_LIST = "user_list"
    ACCOUNT_LIST = "account_list"
    GRADUAL = "gradual"
    ENTITLEMENT = "entitlement"
    A_B_TEST = "ab_test"


class FeatureFlagsClient:
    """Client for managing feature flags"""
    
    def __init__(
        self,
        service_url: str,
        api_key: str,
        environment: str,
        cache_ttl_seconds: int = 300
    ):
        self.service_url = service_url.rstrip('/')
        self.api_key = api_key
        self.environment = environment
        self.cache_ttl = cache_ttl_seconds
        self.headers = {
            "Authorization": f"Bearer {api_key}",
            "Content-Type": "application/json",
            "X-Environment": environment
        }
        self._cache_timestamp = {}
    
    @lru_cache(maxsize=1000)
    def is_enabled(
        self,
        feature_name: str,
        context: Optional[Dict[str, Any]] = None
    ) -> bool:
        """
        Check if a feature is enabled
        
        Args:
            feature_name: Name of the feature flag
            context: Optional context (user_id, account_id, etc.)
        
        Returns:
            True if feature is enabled
        """
        # Check cache expiry
        cache_key = f"{feature_name}:{json.dumps(context or {}, sort_keys=True)}"
        if cache_key in self._cache_timestamp:
            if datetime.now() - self._cache_timestamp[cache_key] > timedelta(seconds=self.cache_ttl):
                self.is_enabled.cache_clear()
                del self._cache_timestamp[cache_key]
        
        try:
            params = {
                "feature": feature_name,
                "environment": self.environment
            }
            
            if context:
                params.update(context)
            
            response = requests.get(
                f"{self.service_url}/api/v1/features/evaluate",
                headers=self.headers,
                params=params
            )
            response.raise_for_status()
            
            result = response.json()
            self._cache_timestamp[cache_key] = datetime.now()
            
            return result.get("enabled", False)
            
        except requests.exceptions.RequestException as e:
            logger.error(f"Failed to check feature flag {feature_name}: {e}")
            # Fail open - return False if service is unavailable
            return False
    
    def get_variant(
        self,
        feature_name: str,
        context: Optional[Dict[str, Any]] = None
    ) -> Optional[str]:
        """
        Get feature variant for A/B testing
        
        Args:
            feature_name: Name of the feature flag
            context: Optional context
        
        Returns:
            Variant name or None
        """
        try:
            params = {
                "feature": feature_name,
                "environment": self.environment
            }
            
            if context:
                params.update(context)
            
            response = requests.get(
                f"{self.service_url}/api/v1/features/variant",
                headers=self.headers,
                params=params
            )
            response.raise_for_status()
            
            result = response.json()
            return result.get("variant")
            
        except requests.exceptions.RequestException as e:
            logger.error(f"Failed to get variant for {feature_name}: {e}")
            return None
    
    def create_feature_flag(
        self,
        name: str,
        description: str,
        enabled: bool = False,
        strategy: RolloutStrategy = RolloutStrategy.PERCENTAGE,
        rollout_percentage: int = 0,
        tags: Optional[List[str]] = None
    ) -> Dict:
        """
        Create a new feature flag
        
        Args:
            name: Feature flag name
            description: Feature description
            enabled: Initial enabled state
            strategy: Rollout strategy
            rollout_percentage: Initial rollout percentage
            tags: Optional tags for organization
        
        Returns:
            Created feature flag details
        """
        payload = {
            "name": name,
            "description": description,
            "enabled": enabled,
            "environment": self.environment,
            "strategy": strategy.value,
            "rollout_config": {
                "percentage": rollout_percentage
            },
            "tags": tags or []
        }
        
        try:
            response = requests.post(
                f"{self.service_url}/api/v1/features",
                headers=self.headers,
                json=payload
            )
            response.raise_for_status()
            
            result = response.json()
            logger.info(f"Created feature flag: {name}")
            return result
            
        except requests.exceptions.RequestException as e:
            logger.error(f"Failed to create feature flag: {e}")
            raise
    
    def update_rollout(
        self,
        feature_name: str,
        percentage: int,
        strategy: Optional[RolloutStrategy] = None
    ) -> Dict:
        """
        Update feature rollout percentage
        
        Args:
            feature_name: Name of the feature
            percentage: New rollout percentage (0-100)
            strategy: Optional new strategy
        
        Returns:
            Updated feature details
        """
        payload = {
            "rollout_config": {
                "percentage": percentage
            }
        }
        
        if strategy:
            payload["strategy"] = strategy.value
        
        try:
            response = requests.patch(
                f"{self.service_url}/api/v1/features/{feature_name}",
                headers=self.headers,
                json=payload
            )
            response.raise_for_status()
            
            result = response.json()
            logger.info(f"Updated rollout for {feature_name} to {percentage}%")
            return result
            
        except requests.exceptions.RequestException as e:
            logger.error(f"Failed to update rollout: {e}")
            raise
    
    def add_to_allowlist(
        self,
        feature_name: str,
        entity_type: str,
        entity_ids: List[Union[str, int]]
    ) -> Dict:
        """
        Add entities to feature allowlist
        
        Args:
            feature_name: Name of the feature
            entity_type: Type of entity (user, account, etc.)
            entity_ids: List of entity IDs to allow
        
        Returns:
            Updated feature details
        """
        payload = {
            "allowlist": {
                entity_type: entity_ids
            }
        }
        
        try:
            response = requests.post(
                f"{self.service_url}/api/v1/features/{feature_name}/allowlist",
                headers=self.headers,
                json=payload
            )
            response.raise_for_status()
            
            result = response.json()
            logger.info(f"Added {len(entity_ids)} {entity_type}s to {feature_name} allowlist")
            return result
            
        except requests.exceptions.RequestException as e:
            logger.error(f"Failed to update allowlist: {e}")
            raise
    
    def create_gradual_rollout(
        self,
        feature_name: str,
        stages: List[Dict[str, Any]]
    ) -> Dict:
        """
        Create a gradual rollout plan
        
        Args:
            feature_name: Name of the feature
            stages: List of rollout stages with percentages and durations
        
        Returns:
            Rollout plan details
        """
        payload = {
            "strategy": "gradual",
            "rollout_plan": {
                "stages": stages
            }
        }
        
        try:
            response = requests.put(
                f"{self.service_url}/api/v1/features/{feature_name}/rollout",
                headers=self.headers,
                json=payload
            )
            response.raise_for_status()
            
            result = response.json()
            logger.info(f"Created gradual rollout for {feature_name} with {len(stages)} stages")
            return result
            
        except requests.exceptions.RequestException as e:
            logger.error(f"Failed to create gradual rollout: {e}")
            raise
    
    def get_all_features(self, tags: Optional[List[str]] = None) -> List[Dict]:
        """
        Get all feature flags
        
        Args:
            tags: Optional tag filter
        
        Returns:
            List of feature flags
        """
        params = {
            "environment": self.environment
        }
        
        if tags:
            params["tags"] = ",".join(tags)
        
        try:
            response = requests.get(
                f"{self.service_url}/api/v1/features",
                headers=self.headers,
                params=params
            )
            response.raise_for_status()
            
            return response.json().get("features", [])
            
        except requests.exceptions.RequestException as e:
            logger.error(f"Failed to get features: {e}")
            raise
    
    def emergency_kill_switch(self, feature_name: str) -> None:
        """
        Emergency disable a feature
        
        Args:
            feature_name: Feature to disable
        """
        payload = {
            "enabled": False,
            "emergency": True,
            "reason": "Emergency kill switch activated"
        }
        
        try:
            response = requests.post(
                f"{self.service_url}/api/v1/features/{feature_name}/emergency",
                headers=self.headers,
                json=payload
            )
            response.raise_for_status()
            
            logger.warning(f"Emergency kill switch activated for {feature_name}")
            
        except requests.exceptions.RequestException as e:
            logger.error(f"Failed to activate kill switch: {e}")
            raise


class FeatureFlagDecorator:
    """Decorator for feature flag gating"""
    
    def __init__(self, client: FeatureFlagsClient):
        self.client = client
    
    def flag(self, feature_name: str, default_return=None):
        """
        Decorator to gate functions with feature flags
        
        Args:
            feature_name: Name of the feature flag
            default_return: Value to return if feature is disabled
        """
        def decorator(func):
            def wrapper(*args, **kwargs):
                context = kwargs.get('_feature_context', {})
                
                if self.client.is_enabled(feature_name, context):
                    return func(*args, **kwargs)
                else:
                    logger.debug(f"Feature {feature_name} is disabled, skipping {func.__name__}")
                    return default_return
            
            return wrapper
        return decorator
    
    def variant(self, feature_name: str):
        """
        Decorator for A/B testing variants
        
        Args:
            feature_name: Name of the feature flag
        """
        def decorator(func):
            def wrapper(*args, **kwargs):
                context = kwargs.get('_feature_context', {})
                variant = self.client.get_variant(feature_name, context)
                
                # Inject variant into function
                kwargs['_variant'] = variant
                return func(*args, **kwargs)
            
            return wrapper
        return decorator


# Example usage patterns
class CleanPlatformFeatures:
    """Feature flags for Clean Platform"""
    
    # Feature flag names
    KAFKA_BATCH_PROCESSING = "kafka_batch_processing"
    ADVANCED_MONITORING = "advanced_monitoring"
    COST_OPTIMIZATION = "cost_optimization"
    CELL_ROUTING_V2 = "cell_routing_v2"
    AUTO_SCALING_ML = "auto_scaling_ml"
    
    def __init__(self, client: FeatureFlagsClient):
        self.client = client
        self.decorator = FeatureFlagDecorator(client)
    
    def is_kafka_batch_enabled(self, account_id: Optional[str] = None) -> bool:
        """Check if Kafka batch processing is enabled"""
        context = {"account_id": account_id} if account_id else {}
        return self.client.is_enabled(self.KAFKA_BATCH_PROCESSING, context)
    
    def get_monitoring_variant(self, user_id: str) -> Optional[str]:
        """Get monitoring variant for A/B testing"""
        return self.client.get_variant(
            self.ADVANCED_MONITORING,
            {"user_id": user_id}
        )


if __name__ == "__main__":
    # Example usage
    import os
    
    client = FeatureFlagsClient(
        service_url=os.environ.get("FEATURE_FLAGS_SERVICE_URL", "http://localhost:8080"),
        api_key=os.environ.get("FEATURE_FLAGS_API_KEY", "test-key"),
        environment="development"
    )
    
    # Check if feature is enabled
    if client.is_enabled("new_ui_design", {"user_id": "12345"}):
        print("New UI design is enabled for user")
    
    # Create gradual rollout
    rollout_stages = [
        {"percentage": 10, "duration_hours": 24},
        {"percentage": 25, "duration_hours": 48},
        {"percentage": 50, "duration_hours": 72},
        {"percentage": 100, "duration_hours": 0}  # Final stage
    ]
    
    client.create_gradual_rollout("new_feature", rollout_stages)