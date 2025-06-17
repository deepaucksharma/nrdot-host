"""
Rate Limiting Client for Clean Platform Implementation
Integrates with the platform rate limiting service
"""

import logging
import requests
import time
from typing import Dict, List, Optional, Tuple, Any
from dataclasses import dataclass
from enum import Enum
from functools import wraps
import threading
from collections import defaultdict
from datetime import datetime, timedelta

logger = logging.getLogger(__name__)


class RateLimitType(Enum):
    """Types of rate limits"""
    ACCOUNT = "account"
    USER = "user"
    API_KEY = "api_key"
    IP_ADDRESS = "ip_address"
    SERVICE = "service"
    GLOBAL = "global"


class LimitScope(Enum):
    """Scope of rate limits"""
    SECOND = "second"
    MINUTE = "minute"
    HOUR = "hour"
    DAY = "day"


@dataclass
class RateLimitConfig:
    """Rate limit configuration"""
    limit_type: RateLimitType
    scope: LimitScope
    limit: int
    burst: Optional[int] = None
    override_accounts: Optional[Dict[str, int]] = None


class RateLimitingClient:
    """Client for platform rate limiting service"""
    
    def __init__(
        self,
        service_url: str,
        api_key: str,
        service_name: str,
        fail_open: bool = True
    ):
        self.service_url = service_url.rstrip('/')
        self.api_key = api_key
        self.service_name = service_name
        self.fail_open = fail_open
        self.headers = {
            "Authorization": f"Bearer {api_key}",
            "Content-Type": "application/json",
            "X-Service-Name": service_name
        }
        # Local cache for rate limit checks
        self._local_cache = defaultdict(lambda: {"count": 0, "reset_time": time.time()})
        self._cache_lock = threading.Lock()
    
    def check_rate_limit(
        self,
        identifier: str,
        limit_type: RateLimitType,
        operation: Optional[str] = None
    ) -> Tuple[bool, Dict[str, Any]]:
        """
        Check if request is within rate limits
        
        Args:
            identifier: The identifier to check (account_id, user_id, etc.)
            limit_type: Type of rate limit to check
            operation: Optional operation name for granular limits
        
        Returns:
            Tuple of (allowed, metadata)
        """
        try:
            params = {
                "identifier": identifier,
                "limit_type": limit_type.value,
                "service": self.service_name
            }
            
            if operation:
                params["operation"] = operation
            
            response = requests.get(
                f"{self.service_url}/api/v1/check",
                headers=self.headers,
                params=params,
                timeout=1.0  # Fast timeout for rate limit checks
            )
            response.raise_for_status()
            
            result = response.json()
            allowed = result.get("allowed", True)
            metadata = {
                "limit": result.get("limit"),
                "remaining": result.get("remaining"),
                "reset_time": result.get("reset_time"),
                "retry_after": result.get("retry_after")
            }
            
            if not allowed:
                logger.warning(
                    f"Rate limit exceeded for {identifier} "
                    f"({limit_type.value}): {metadata}"
                )
            
            return allowed, metadata
            
        except requests.exceptions.RequestException as e:
            logger.error(f"Failed to check rate limit: {e}")
            # Fail open - allow request if service is unavailable
            if self.fail_open:
                return True, {"error": str(e), "fail_open": True}
            return False, {"error": str(e)}
    
    def consume_token(
        self,
        identifier: str,
        limit_type: RateLimitType,
        tokens: int = 1,
        operation: Optional[str] = None
    ) -> Tuple[bool, Dict[str, Any]]:
        """
        Consume tokens from rate limit bucket
        
        Args:
            identifier: The identifier
            limit_type: Type of rate limit
            tokens: Number of tokens to consume
            operation: Optional operation name
        
        Returns:
            Tuple of (success, metadata)
        """
        payload = {
            "identifier": identifier,
            "limit_type": limit_type.value,
            "service": self.service_name,
            "tokens": tokens
        }
        
        if operation:
            payload["operation"] = operation
        
        try:
            response = requests.post(
                f"{self.service_url}/api/v1/consume",
                headers=self.headers,
                json=payload,
                timeout=1.0
            )
            response.raise_for_status()
            
            result = response.json()
            success = result.get("success", False)
            metadata = {
                "remaining": result.get("remaining"),
                "reset_time": result.get("reset_time"),
                "burst_remaining": result.get("burst_remaining")
            }
            
            return success, metadata
            
        except requests.exceptions.RequestException as e:
            logger.error(f"Failed to consume tokens: {e}")
            if self.fail_open:
                return True, {"error": str(e), "fail_open": True}
            return False, {"error": str(e)}
    
    def get_limits(self, identifier: str, limit_type: RateLimitType) -> Dict[str, Any]:
        """
        Get current rate limits for an identifier
        
        Args:
            identifier: The identifier
            limit_type: Type of rate limit
        
        Returns:
            Current limits and usage
        """
        try:
            response = requests.get(
                f"{self.service_url}/api/v1/limits",
                headers=self.headers,
                params={
                    "identifier": identifier,
                    "limit_type": limit_type.value,
                    "service": self.service_name
                }
            )
            response.raise_for_status()
            
            return response.json()
            
        except requests.exceptions.RequestException as e:
            logger.error(f"Failed to get limits: {e}")
            return {"error": str(e)}
    
    def create_override(
        self,
        identifier: str,
        limit_type: RateLimitType,
        new_limit: int,
        duration_hours: Optional[int] = None,
        reason: str = ""
    ) -> Dict[str, Any]:
        """
        Create a rate limit override
        
        Args:
            identifier: The identifier to override
            limit_type: Type of rate limit
            new_limit: New limit value
            duration_hours: Duration of override (None for permanent)
            reason: Reason for override
        
        Returns:
            Override details
        """
        payload = {
            "identifier": identifier,
            "limit_type": limit_type.value,
            "service": self.service_name,
            "new_limit": new_limit,
            "reason": reason
        }
        
        if duration_hours:
            payload["expires_at"] = (
                datetime.utcnow() + timedelta(hours=duration_hours)
            ).isoformat()
        
        try:
            response = requests.post(
                f"{self.service_url}/api/v1/overrides",
                headers=self.headers,
                json=payload
            )
            response.raise_for_status()
            
            result = response.json()
            logger.info(
                f"Created rate limit override for {identifier} "
                f"({limit_type.value}): {new_limit}"
            )
            return result
            
        except requests.exceptions.RequestException as e:
            logger.error(f"Failed to create override: {e}")
            raise
    
    def local_rate_limit(
        self,
        identifier: str,
        limit: int,
        window_seconds: int = 60
    ) -> bool:
        """
        Local rate limiting fallback
        
        Args:
            identifier: The identifier
            limit: Request limit
            window_seconds: Time window in seconds
        
        Returns:
            True if request is allowed
        """
        with self._cache_lock:
            now = time.time()
            cache_key = f"{identifier}:{window_seconds}"
            cache_entry = self._local_cache[cache_key]
            
            # Reset counter if window has passed
            if now > cache_entry["reset_time"]:
                cache_entry["count"] = 0
                cache_entry["reset_time"] = now + window_seconds
            
            # Check limit
            if cache_entry["count"] >= limit:
                return False
            
            # Increment counter
            cache_entry["count"] += 1
            return True


class RateLimitDecorator:
    """Decorator for rate limiting functions"""
    
    def __init__(self, client: RateLimitingClient):
        self.client = client
    
    def limit(
        self,
        limit_type: RateLimitType,
        get_identifier=None,
        operation: Optional[str] = None,
        fallback_limit: int = 100,
        fallback_window: int = 60
    ):
        """
        Rate limit decorator
        
        Args:
            limit_type: Type of rate limit
            get_identifier: Function to extract identifier from args/kwargs
            operation: Operation name
            fallback_limit: Local fallback limit
            fallback_window: Local fallback window in seconds
        """
        def decorator(func):
            @wraps(func)
            def wrapper(*args, **kwargs):
                # Extract identifier
                if get_identifier:
                    identifier = get_identifier(*args, **kwargs)
                else:
                    # Default: try to get from kwargs
                    identifier = kwargs.get('account_id') or \
                                kwargs.get('user_id') or \
                                kwargs.get('api_key') or \
                                'default'
                
                # Check rate limit
                allowed, metadata = self.client.check_rate_limit(
                    identifier,
                    limit_type,
                    operation or func.__name__
                )
                
                # If service is down, use local rate limiting
                if metadata.get('fail_open') and fallback_limit:
                    allowed = self.client.local_rate_limit(
                        identifier,
                        fallback_limit,
                        fallback_window
                    )
                
                if not allowed:
                    retry_after = metadata.get('retry_after', 60)
                    raise RateLimitExceededException(
                        f"Rate limit exceeded. Retry after {retry_after} seconds",
                        retry_after=retry_after,
                        metadata=metadata
                    )
                
                # Execute function
                return func(*args, **kwargs)
            
            return wrapper
        return decorator


class RateLimitExceededException(Exception):
    """Exception raised when rate limit is exceeded"""
    
    def __init__(self, message: str, retry_after: int = 60, metadata: Dict = None):
        super().__init__(message)
        self.retry_after = retry_after
        self.metadata = metadata or {}


# Platform-specific rate limit configurations
class CleanPlatformRateLimits:
    """Rate limit configurations for Clean Platform"""
    
    # API limits
    API_LIMITS = {
        "data_ingestion": RateLimitConfig(
            limit_type=RateLimitType.ACCOUNT,
            scope=LimitScope.MINUTE,
            limit=1000,
            burst=2000,
            override_accounts={
                "premium_tier": 10000,
                "enterprise_tier": 50000
            }
        ),
        "query_api": RateLimitConfig(
            limit_type=RateLimitType.USER,
            scope=LimitScope.MINUTE,
            limit=100,
            burst=150
        ),
        "admin_api": RateLimitConfig(
            limit_type=RateLimitType.API_KEY,
            scope=LimitScope.HOUR,
            limit=1000
        )
    }
    
    # Service limits
    SERVICE_LIMITS = {
        "kafka_producer": RateLimitConfig(
            limit_type=RateLimitType.SERVICE,
            scope=LimitScope.SECOND,
            limit=10000
        ),
        "database_writes": RateLimitConfig(
            limit_type=RateLimitType.SERVICE,
            scope=LimitScope.SECOND,
            limit=5000
        )
    }
    
    @classmethod
    def get_limit_config(cls, operation: str) -> Optional[RateLimitConfig]:
        """Get rate limit configuration for an operation"""
        return cls.API_LIMITS.get(operation) or cls.SERVICE_LIMITS.get(operation)


# Example usage
if __name__ == "__main__":
    import os
    
    client = RateLimitingClient(
        service_url=os.environ.get("RATE_LIMITING_SERVICE_URL", "http://localhost:8080"),
        api_key=os.environ.get("RATE_LIMITING_API_KEY", "test-key"),
        service_name="clean-platform",
        fail_open=True
    )
    
    # Check rate limit
    allowed, metadata = client.check_rate_limit(
        identifier="account-123",
        limit_type=RateLimitType.ACCOUNT,
        operation="data_ingestion"
    )
    
    if allowed:
        print(f"Request allowed. Remaining: {metadata.get('remaining')}")
    else:
        print(f"Rate limit exceeded. Retry after: {metadata.get('retry_after')}s")
    
    # Using decorator
    decorator = RateLimitDecorator(client)
    
    @decorator.limit(
        limit_type=RateLimitType.ACCOUNT,
        operation="data_ingestion",
        fallback_limit=100
    )
    def ingest_data(account_id: str, data: dict):
        print(f"Ingesting data for account {account_id}")
    
    try:
        ingest_data(account_id="account-123", data={"test": "data"})
    except RateLimitExceededException as e:
        print(f"Rate limited: {e}, retry after {e.retry_after}s")