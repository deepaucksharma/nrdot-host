"""
Service Discovery (Biosecurity) Client for Clean Platform Implementation
Implements Vault-backed service discovery patterns
"""

import logging
import json
import os
from typing import Dict, List, Optional, Any, Tuple
from dataclasses import dataclass
from enum import Enum
import hvac
from functools import lru_cache
from datetime import datetime, timedelta
import threading
import time

logger = logging.getLogger(__name__)


class ServiceType(Enum):
    """Types of services in discovery"""
    DATABASE = "database"
    CACHE = "cache"
    API = "api"
    KAFKA = "kafka"
    INTERNAL = "internal"
    EXTERNAL = "external"


@dataclass
class ServiceEndpoint:
    """Service endpoint information"""
    host: str
    port: int
    protocol: str = "https"
    region: Optional[str] = None
    cell: Optional[str] = None
    metadata: Optional[Dict[str, Any]] = None


@dataclass
class ServiceCredentials:
    """Service credentials"""
    username: Optional[str] = None
    password: Optional[str] = None
    api_key: Optional[str] = None
    token: Optional[str] = None
    certificate: Optional[str] = None
    private_key: Optional[str] = None
    metadata: Optional[Dict[str, Any]] = None


class BiosecurityClient:
    """Client for Vault-backed service discovery"""
    
    def __init__(
        self,
        vault_addr: str = None,
        vault_token: str = None,
        team_name: str = None,
        environment: str = None,
        cache_ttl_seconds: int = 300
    ):
        self.vault_addr = vault_addr or os.environ.get(
            'VAULT_ADDR',
            'https://vault-prd1a.r10.us.nr-ops.net:8200'
        )
        self.vault_token = vault_token or os.environ.get('VAULT_TOKEN')
        self.team_name = team_name or os.environ.get('TEAM_NAME')
        self.environment = environment or os.environ.get('ENVIRONMENT', 'development')
        self.cache_ttl = cache_ttl_seconds
        
        # Initialize Vault client
        self.vault = hvac.Client(
            url=self.vault_addr,
            token=self.vault_token
        )
        
        if not self.vault.is_authenticated():
            raise ValueError("Vault authentication failed")
        
        # Cache management
        self._cache = {}
        self._cache_timestamps = {}
        self._cache_lock = threading.Lock()
        
        # Start cache cleanup thread
        self._start_cache_cleanup()
    
    def _get_discovery_path(self, service_name: str, service_type: ServiceType) -> str:
        """
        Get Vault path for service discovery
        
        Standard pattern: terraform/{team}/{environment}/{cell}/{service}/{service}-{type}
        """
        cell = os.environ.get('GRAND_CENTRAL_CELL', '*')
        
        return (
            f"terraform/{self.team_name}/{self.environment}/"
            f"{cell}/{service_name}/{service_name}-{service_type.value}"
        )
    
    @lru_cache(maxsize=100)
    def discover_service(
        self,
        service_name: str,
        service_type: ServiceType
    ) -> Tuple[ServiceEndpoint, ServiceCredentials]:
        """
        Discover service endpoint and credentials
        
        Args:
            service_name: Name of the service
            service_type: Type of service
        
        Returns:
            Tuple of (endpoint, credentials)
        """
        cache_key = f"{service_name}:{service_type.value}"
        
        # Check cache
        with self._cache_lock:
            if cache_key in self._cache:
                cached_time = self._cache_timestamps[cache_key]
                if datetime.now() - cached_time < timedelta(seconds=self.cache_ttl):
                    return self._cache[cache_key]
        
        # Fetch from Vault
        path = self._get_discovery_path(service_name, service_type)
        
        try:
            response = self.vault.read(path)
            if not response or 'data' not in response:
                raise ValueError(f"No data found at path: {path}")
            
            data = response['data']
            
            # Parse endpoint
            endpoint = self._parse_endpoint(data)
            
            # Parse credentials
            credentials = self._parse_credentials(data)
            
            # Cache result
            result = (endpoint, credentials)
            with self._cache_lock:
                self._cache[cache_key] = result
                self._cache_timestamps[cache_key] = datetime.now()
            
            logger.info(f"Discovered service: {service_name} ({service_type.value})")
            return result
            
        except Exception as e:
            logger.error(f"Failed to discover service {service_name}: {e}")
            raise
    
    def _parse_endpoint(self, data: Dict[str, Any]) -> ServiceEndpoint:
        """Parse endpoint from Vault data"""
        # Common patterns for endpoints
        endpoint_str = (
            data.get('endpoint') or
            data.get('host') or
            data.get('address') or
            data.get('url')
        )
        
        if not endpoint_str:
            raise ValueError("No endpoint found in service data")
        
        # Parse host and port
        if ':' in endpoint_str and not endpoint_str.startswith('http'):
            host, port = endpoint_str.rsplit(':', 1)
            port = int(port)
        else:
            # Try to extract from URL
            if endpoint_str.startswith(('http://', 'https://')):
                protocol = 'https' if endpoint_str.startswith('https') else 'http'
                url_parts = endpoint_str.replace(f'{protocol}://', '').split('/')
                host_port = url_parts[0]
                
                if ':' in host_port:
                    host, port = host_port.rsplit(':', 1)
                    port = int(port)
                else:
                    host = host_port
                    port = 443 if protocol == 'https' else 80
            else:
                host = endpoint_str
                port = data.get('port', 443)
        
        return ServiceEndpoint(
            host=host,
            port=port,
            protocol=data.get('protocol', 'https'),
            region=data.get('region'),
            cell=data.get('cell'),
            metadata={
                k: v for k, v in data.items()
                if k not in ['endpoint', 'host', 'port', 'username', 'password']
            }
        )
    
    def _parse_credentials(self, data: Dict[str, Any]) -> ServiceCredentials:
        """Parse credentials from Vault data"""
        return ServiceCredentials(
            username=data.get('username') or data.get('user'),
            password=data.get('password') or data.get('pass'),
            api_key=data.get('api_key') or data.get('apikey'),
            token=data.get('token') or data.get('auth_token'),
            certificate=data.get('certificate') or data.get('cert'),
            private_key=data.get('private_key') or data.get('key'),
            metadata={
                'auth_type': data.get('auth_type', 'basic'),
                'ttl': data.get('ttl'),
                'renewable': data.get('renewable', False)
            }
        )
    
    def get_database_connection(self, db_name: str) -> Dict[str, Any]:
        """
        Get database connection details
        
        Args:
            db_name: Database name
        
        Returns:
            Connection configuration
        """
        endpoint, creds = self.discover_service(db_name, ServiceType.DATABASE)
        
        return {
            'host': endpoint.host,
            'port': endpoint.port,
            'database': endpoint.metadata.get('database', db_name),
            'username': creds.username,
            'password': creds.password,
            'ssl': endpoint.metadata.get('ssl', True),
            'options': endpoint.metadata.get('options', {})
        }
    
    def get_redis_connection(self, cache_name: str) -> Dict[str, Any]:
        """
        Get Redis connection details
        
        Args:
            cache_name: Cache cluster name
        
        Returns:
            Redis connection configuration
        """
        endpoint, creds = self.discover_service(cache_name, ServiceType.CACHE)
        
        return {
            'host': endpoint.host,
            'port': endpoint.port,
            'password': creds.password,
            'ssl': endpoint.protocol == 'rediss',
            'db': endpoint.metadata.get('db', 0),
            'decode_responses': True
        }
    
    def get_kafka_config(self, cluster_name: str = "default") -> Dict[str, Any]:
        """
        Get Kafka cluster configuration
        
        Args:
            cluster_name: Kafka cluster name
        
        Returns:
            Kafka configuration
        """
        # Special handling for Kafka - uses secret key refs
        kafka_path = f"secret_key_ref:kpt-kafka-client-config"
        
        try:
            # Read Kafka configuration
            response = self.vault.read(f"secret/{self.team_name}/kpt-kafka-client-config")
            if not response or 'data' not in response:
                # Fallback to environment variables
                return {
                    'bootstrap_servers': os.environ.get('KAFKA_BROKERS'),
                    'security_protocol': os.environ.get('KAFKA_SECURITY_PROTOCOL', 'PLAINTEXT'),
                    'sasl_mechanism': os.environ.get('KAFKA_SASL_MECHANISM'),
                    'sasl_plain_username': os.environ.get('KAFKA_SASL_USERNAME'),
                    'sasl_plain_password': os.environ.get('KAFKA_SASL_PASSWORD'),
                }
            
            data = response['data'].get('data', response['data'])
            
            return {
                'bootstrap_servers': data.get('BOOTSTRAP_BROKERS_DEFAULT'),
                'security_protocol': data.get('SECURITY_PROTOCOL_DEFAULT', 'PLAINTEXT'),
                'sasl_mechanism': data.get('SASL_MECHANISM'),
                'sasl_plain_username': data.get('SASL_USERNAME'),
                'sasl_plain_password': data.get('SASL_PASSWORD'),
                'ssl_ca_location': data.get('SSL_CA_LOCATION'),
                'ssl_certfile': data.get('SSL_CERTFILE'),
                'ssl_keyfile': data.get('SSL_KEYFILE'),
            }
            
        except Exception as e:
            logger.error(f"Failed to get Kafka config: {e}")
            raise
    
    def get_api_client(self, service_name: str) -> Dict[str, Any]:
        """
        Get API client configuration
        
        Args:
            service_name: API service name
        
        Returns:
            API client configuration
        """
        endpoint, creds = self.discover_service(service_name, ServiceType.API)
        
        base_url = f"{endpoint.protocol}://{endpoint.host}"
        if endpoint.port not in [80, 443]:
            base_url += f":{endpoint.port}"
        
        headers = {}
        
        # Add authentication headers
        if creds.api_key:
            headers['Authorization'] = f"Bearer {creds.api_key}"
        elif creds.token:
            headers['X-Auth-Token'] = creds.token
        elif creds.username and creds.password:
            import base64
            auth = base64.b64encode(f"{creds.username}:{creds.password}".encode()).decode()
            headers['Authorization'] = f"Basic {auth}"
        
        return {
            'base_url': base_url,
            'headers': headers,
            'timeout': endpoint.metadata.get('timeout', 30),
            'verify_ssl': endpoint.metadata.get('verify_ssl', True),
            'retry_count': endpoint.metadata.get('retry_count', 3)
        }
    
    def refresh_credentials(self, service_name: str, service_type: ServiceType):
        """
        Force refresh of cached credentials
        
        Args:
            service_name: Service name
            service_type: Service type
        """
        cache_key = f"{service_name}:{service_type.value}"
        
        with self._cache_lock:
            if cache_key in self._cache:
                del self._cache[cache_key]
                del self._cache_timestamps[cache_key]
        
        # Clear LRU cache
        self.discover_service.cache_clear()
        
        logger.info(f"Refreshed credentials for {service_name}")
    
    def _start_cache_cleanup(self):
        """Start background thread for cache cleanup"""
        def cleanup():
            while True:
                time.sleep(60)  # Check every minute
                now = datetime.now()
                
                with self._cache_lock:
                    expired_keys = [
                        key for key, timestamp in self._cache_timestamps.items()
                        if now - timestamp > timedelta(seconds=self.cache_ttl * 2)
                    ]
                    
                    for key in expired_keys:
                        del self._cache[key]
                        del self._cache_timestamps[key]
                
                if expired_keys:
                    logger.debug(f"Cleaned up {len(expired_keys)} expired cache entries")
        
        thread = threading.Thread(target=cleanup, daemon=True)
        thread.start()


# Helper class for connection management
class ServiceConnectionManager:
    """Manages service connections with automatic discovery"""
    
    def __init__(self, biosecurity_client: BiosecurityClient):
        self.biosecurity = biosecurity_client
        self._connections = {}
    
    def get_postgres_connection(self, db_name: str):
        """Get PostgreSQL connection with automatic discovery"""
        import psycopg2
        from psycopg2.pool import ThreadedConnectionPool
        
        if db_name in self._connections:
            return self._connections[db_name]
        
        config = self.biosecurity.get_database_connection(db_name)
        
        # Create connection pool
        pool = ThreadedConnectionPool(
            minconn=1,
            maxconn=20,
            host=config['host'],
            port=config['port'],
            database=config['database'],
            user=config['username'],
            password=config['password'],
            sslmode='require' if config['ssl'] else 'disable'
        )
        
        self._connections[db_name] = pool
        return pool
    
    def get_redis_client(self, cache_name: str):
        """Get Redis client with automatic discovery"""
        import redis
        
        if cache_name in self._connections:
            return self._connections[cache_name]
        
        config = self.biosecurity.get_redis_connection(cache_name)
        
        client = redis.Redis(**config)
        self._connections[cache_name] = client
        return client
    
    def close_all(self):
        """Close all connections"""
        for name, conn in self._connections.items():
            try:
                if hasattr(conn, 'closeall'):
                    conn.closeall()  # PostgreSQL pool
                elif hasattr(conn, 'close'):
                    conn.close()  # Redis client
            except Exception as e:
                logger.error(f"Error closing connection {name}: {e}")
        
        self._connections.clear()


if __name__ == "__main__":
    # Example usage
    client = BiosecurityClient(
        team_name="platform-team",
        environment="production"
    )
    
    # Discover database
    try:
        db_config = client.get_database_connection("clean-platform-db")
        print(f"Database host: {db_config['host']}")
        print(f"Database port: {db_config['port']}")
    except Exception as e:
        print(f"Failed to discover database: {e}")
    
    # Discover Redis
    try:
        redis_config = client.get_redis_connection("clean-platform-cache")
        print(f"Redis host: {redis_config['host']}")
        print(f"Redis port: {redis_config['port']}")
    except Exception as e:
        print(f"Failed to discover Redis: {e}")
    
    # Get Kafka configuration
    try:
        kafka_config = client.get_kafka_config()
        print(f"Kafka brokers: {kafka_config['bootstrap_servers']}")
    except Exception as e:
        print(f"Failed to get Kafka config: {e}")