"""
Circuit Breaker Implementation for Clean Platform
Provides fault tolerance and graceful degradation
"""

import time
import logging
import threading
from typing import Callable, Any, Optional, Dict, List
from enum import Enum
from datetime import datetime, timedelta
from functools import wraps
from collections import deque
import json

logger = logging.getLogger(__name__)


class CircuitState(Enum):
    """Circuit breaker states"""
    CLOSED = "closed"  # Normal operation
    OPEN = "open"      # Failing, rejecting calls
    HALF_OPEN = "half_open"  # Testing if service recovered


class CircuitBreakerError(Exception):
    """Raised when circuit breaker is open"""
    pass


class CircuitBreakerMetrics:
    """Tracks circuit breaker metrics"""
    
    def __init__(self, window_size: int = 100):
        self.window_size = window_size
        self.calls = deque(maxlen=window_size)
        self.lock = threading.Lock()
        
        # Counters
        self.total_calls = 0
        self.successful_calls = 0
        self.failed_calls = 0
        self.rejected_calls = 0
        
        # Timing
        self.last_failure_time = None
        self.last_success_time = None
        self.circuit_opened_at = None
    
    def record_success(self, duration: float):
        """Record successful call"""
        with self.lock:
            self.calls.append({
                'success': True,
                'timestamp': time.time(),
                'duration': duration
            })
            self.total_calls += 1
            self.successful_calls += 1
            self.last_success_time = datetime.now()
    
    def record_failure(self, error: Exception, duration: float):
        """Record failed call"""
        with self.lock:
            self.calls.append({
                'success': False,
                'timestamp': time.time(),
                'duration': duration,
                'error': str(error)
            })
            self.total_calls += 1
            self.failed_calls += 1
            self.last_failure_time = datetime.now()
    
    def record_rejection(self):
        """Record rejected call"""
        with self.lock:
            self.total_calls += 1
            self.rejected_calls += 1
    
    def get_failure_rate(self) -> float:
        """Calculate recent failure rate"""
        with self.lock:
            if not self.calls:
                return 0.0
            
            failures = sum(1 for call in self.calls if not call['success'])
            return failures / len(self.calls)
    
    def get_stats(self) -> Dict[str, Any]:
        """Get current statistics"""
        with self.lock:
            return {
                'total_calls': self.total_calls,
                'successful_calls': self.successful_calls,
                'failed_calls': self.failed_calls,
                'rejected_calls': self.rejected_calls,
                'failure_rate': self.get_failure_rate(),
                'last_failure': self.last_failure_time.isoformat() if self.last_failure_time else None,
                'last_success': self.last_success_time.isoformat() if self.last_success_time else None,
                'circuit_opened_at': self.circuit_opened_at.isoformat() if self.circuit_opened_at else None
            }


class CircuitBreaker:
    """
    Circuit breaker implementation with configurable thresholds
    """
    
    def __init__(
        self,
        name: str,
        failure_threshold: int = 5,
        recovery_timeout: int = 60,
        expected_exception: type = Exception,
        success_threshold: int = 2,
        failure_rate_threshold: float = 0.5,
        window_size: int = 100,
        fallback_function: Optional[Callable] = None
    ):
        """
        Initialize circuit breaker
        
        Args:
            name: Circuit breaker name
            failure_threshold: Number of failures before opening
            recovery_timeout: Seconds before attempting recovery
            expected_exception: Exception type to catch
            success_threshold: Successes needed to close from half-open
            failure_rate_threshold: Failure rate to open circuit
            window_size: Rolling window size for metrics
            fallback_function: Function to call when circuit is open
        """
        self.name = name
        self.failure_threshold = failure_threshold
        self.recovery_timeout = recovery_timeout
        self.expected_exception = expected_exception
        self.success_threshold = success_threshold
        self.failure_rate_threshold = failure_rate_threshold
        self.fallback_function = fallback_function
        
        # State management
        self._state = CircuitState.CLOSED
        self._failure_count = 0
        self._success_count = 0
        self._last_failure_time = None
        self._state_lock = threading.Lock()
        
        # Metrics
        self.metrics = CircuitBreakerMetrics(window_size)
        
        # Listeners for state changes
        self._state_change_listeners = []
        
        logger.info(f"Circuit breaker '{name}' initialized")
    
    @property
    def state(self) -> CircuitState:
        """Get current state"""
        with self._state_lock:
            # Check if we should transition from OPEN to HALF_OPEN
            if self._state == CircuitState.OPEN:
                if self._should_attempt_reset():
                    self._transition_to_half_open()
            
            return self._state
    
    def call(self, func: Callable, *args, **kwargs) -> Any:
        """
        Call function through circuit breaker
        
        Args:
            func: Function to call
            *args: Function arguments
            **kwargs: Function keyword arguments
            
        Returns:
            Function result or fallback result
            
        Raises:
            CircuitBreakerError: If circuit is open and no fallback
        """
        # Check state and potentially reject
        if self.state == CircuitState.OPEN:
            self.metrics.record_rejection()
            
            if self.fallback_function:
                logger.warning(f"Circuit breaker '{self.name}' is OPEN, using fallback")
                return self.fallback_function(*args, **kwargs)
            else:
                raise CircuitBreakerError(
                    f"Circuit breaker '{self.name}' is OPEN"
                )
        
        # Attempt the call
        start_time = time.time()
        try:
            result = func(*args, **kwargs)
            duration = time.time() - start_time
            self._on_success(duration)
            return result
            
        except self.expected_exception as e:
            duration = time.time() - start_time
            self._on_failure(e, duration)
            
            # If half-open, immediately go back to open
            if self.state == CircuitState.HALF_OPEN:
                if self.fallback_function:
                    return self.fallback_function(*args, **kwargs)
            
            raise
    
    def _on_success(self, duration: float):
        """Handle successful call"""
        with self._state_lock:
            self.metrics.record_success(duration)
            
            if self._state == CircuitState.HALF_OPEN:
                self._success_count += 1
                logger.debug(
                    f"Circuit breaker '{self.name}' success in HALF_OPEN "
                    f"({self._success_count}/{self.success_threshold})"
                )
                
                if self._success_count >= self.success_threshold:
                    self._transition_to_closed()
            else:
                self._failure_count = 0  # Reset failure count on success
    
    def _on_failure(self, error: Exception, duration: float):
        """Handle failed call"""
        with self._state_lock:
            self.metrics.record_failure(error, duration)
            self._last_failure_time = time.time()
            
            if self._state == CircuitState.CLOSED:
                self._failure_count += 1
                logger.warning(
                    f"Circuit breaker '{self.name}' failure "
                    f"({self._failure_count}/{self.failure_threshold}): {error}"
                )
                
                # Check if we should open the circuit
                if (self._failure_count >= self.failure_threshold or
                    self.metrics.get_failure_rate() >= self.failure_rate_threshold):
                    self._transition_to_open()
                    
            elif self._state == CircuitState.HALF_OPEN:
                logger.warning(
                    f"Circuit breaker '{self.name}' failure in HALF_OPEN, reopening"
                )
                self._transition_to_open()
    
    def _should_attempt_reset(self) -> bool:
        """Check if enough time has passed to attempt reset"""
        return (
            self._last_failure_time and
            time.time() - self._last_failure_time >= self.recovery_timeout
        )
    
    def _transition_to_open(self):
        """Transition to OPEN state"""
        logger.error(f"Circuit breaker '{self.name}' transitioning to OPEN")
        self._state = CircuitState.OPEN
        self._failure_count = 0
        self._success_count = 0
        self.metrics.circuit_opened_at = datetime.now()
        self._notify_state_change(CircuitState.OPEN)
    
    def _transition_to_half_open(self):
        """Transition to HALF_OPEN state"""
        logger.info(f"Circuit breaker '{self.name}' transitioning to HALF_OPEN")
        self._state = CircuitState.HALF_OPEN
        self._success_count = 0
        self._failure_count = 0
        self._notify_state_change(CircuitState.HALF_OPEN)
    
    def _transition_to_closed(self):
        """Transition to CLOSED state"""
        logger.info(f"Circuit breaker '{self.name}' transitioning to CLOSED")
        self._state = CircuitState.CLOSED
        self._failure_count = 0
        self._success_count = 0
        self.metrics.circuit_opened_at = None
        self._notify_state_change(CircuitState.CLOSED)
    
    def _notify_state_change(self, new_state: CircuitState):
        """Notify listeners of state change"""
        for listener in self._state_change_listeners:
            try:
                listener(self.name, new_state)
            except Exception as e:
                logger.error(f"Error notifying state change listener: {e}")
    
    def add_state_change_listener(self, listener: Callable[[str, CircuitState], None]):
        """Add listener for state changes"""
        self._state_change_listeners.append(listener)
    
    def reset(self):
        """Manually reset the circuit breaker"""
        with self._state_lock:
            logger.info(f"Manually resetting circuit breaker '{self.name}'")
            self._transition_to_closed()
    
    def get_stats(self) -> Dict[str, Any]:
        """Get circuit breaker statistics"""
        stats = self.metrics.get_stats()
        stats.update({
            'name': self.name,
            'state': self.state.value,
            'failure_threshold': self.failure_threshold,
            'recovery_timeout': self.recovery_timeout,
            'failure_rate_threshold': self.failure_rate_threshold
        })
        return stats


class CircuitBreakerDecorator:
    """Decorator for applying circuit breaker to functions"""
    
    def __init__(
        self,
        name: Optional[str] = None,
        failure_threshold: int = 5,
        recovery_timeout: int = 60,
        expected_exception: type = Exception,
        fallback_function: Optional[Callable] = None
    ):
        self.name = name
        self.failure_threshold = failure_threshold
        self.recovery_timeout = recovery_timeout
        self.expected_exception = expected_exception
        self.fallback_function = fallback_function
        self._breakers = {}
    
    def __call__(self, func: Callable) -> Callable:
        # Use function name if no name provided
        breaker_name = self.name or f"{func.__module__}.{func.__name__}"
        
        # Create circuit breaker for this function
        if breaker_name not in self._breakers:
            self._breakers[breaker_name] = CircuitBreaker(
                name=breaker_name,
                failure_threshold=self.failure_threshold,
                recovery_timeout=self.recovery_timeout,
                expected_exception=self.expected_exception,
                fallback_function=self.fallback_function
            )
        
        breaker = self._breakers[breaker_name]
        
        @wraps(func)
        def wrapper(*args, **kwargs):
            return breaker.call(func, *args, **kwargs)
        
        # Attach breaker for inspection
        wrapper.circuit_breaker = breaker
        
        return wrapper


# Convenience decorator
circuit_breaker = CircuitBreakerDecorator


# Global circuit breaker registry
class CircuitBreakerRegistry:
    """Registry for managing multiple circuit breakers"""
    
    def __init__(self):
        self._breakers: Dict[str, CircuitBreaker] = {}
        self._lock = threading.Lock()
    
    def register(self, breaker: CircuitBreaker):
        """Register a circuit breaker"""
        with self._lock:
            self._breakers[breaker.name] = breaker
    
    def get(self, name: str) -> Optional[CircuitBreaker]:
        """Get circuit breaker by name"""
        return self._breakers.get(name)
    
    def get_all_stats(self) -> List[Dict[str, Any]]:
        """Get stats for all circuit breakers"""
        with self._lock:
            return [breaker.get_stats() for breaker in self._breakers.values()]
    
    def reset_all(self):
        """Reset all circuit breakers"""
        with self._lock:
            for breaker in self._breakers.values():
                breaker.reset()


# Global registry instance
registry = CircuitBreakerRegistry()


# Example usage for Clean Platform
if __name__ == "__main__":
    import requests
    import random
    
    # Example 1: Basic circuit breaker
    @circuit_breaker(
        name="external_api",
        failure_threshold=3,
        recovery_timeout=30
    )
    def call_external_api(endpoint: str):
        response = requests.get(f"https://api.example.com/{endpoint}", timeout=5)
        response.raise_for_status()
        return response.json()
    
    # Example 2: Circuit breaker with fallback
    def get_cached_data():
        return {"data": "cached", "stale": True}
    
    @circuit_breaker(
        name="database_query",
        failure_threshold=5,
        recovery_timeout=60,
        fallback_function=lambda: get_cached_data()
    )
    def query_database():
        # Simulate database query
        if random.random() > 0.7:  # 30% failure rate
            raise Exception("Database connection failed")
        return {"data": "fresh", "stale": False}
    
    # Example 3: Manual circuit breaker usage
    db_breaker = CircuitBreaker(
        name="postgres_main",
        failure_threshold=3,
        recovery_timeout=120,
        expected_exception=ConnectionError
    )
    
    def get_user(user_id: int):
        def _query():
            # Database query logic
            pass
        
        try:
            return db_breaker.call(_query)
        except CircuitBreakerError:
            # Return default or cached data
            return {"id": user_id, "name": "Unknown", "cached": True}
    
    # Example 4: State change monitoring
    def on_state_change(breaker_name: str, new_state: CircuitState):
        logger.warning(f"Circuit breaker '{breaker_name}' changed to {new_state.value}")
        # Could send alert, update dashboard, etc.
    
    db_breaker.add_state_change_listener(on_state_change)
    
    # Register breakers
    registry.register(db_breaker)
    
    # Get stats
    print(json.dumps(registry.get_all_stats(), indent=2))