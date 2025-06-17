"""
Distributed Tracing Configuration for Clean Platform
Implements OpenTelemetry with New Relic integration
"""

import os
import logging
from typing import Optional, Dict, Any
from functools import wraps
from contextlib import contextmanager

from opentelemetry import trace, baggage, metrics
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
from opentelemetry.exporter.otlp.proto.grpc.metric_exporter import OTLPMetricExporter
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.sdk.metrics import MeterProvider
from opentelemetry.sdk.metrics.export import PeriodicExportingMetricReader
from opentelemetry.sdk.resources import Resource
from opentelemetry.instrumentation.requests import RequestsInstrumentor
from opentelemetry.instrumentation.flask import FlaskInstrumentor
from opentelemetry.instrumentation.kafka import KafkaInstrumentor
from opentelemetry.instrumentation.redis import RedisInstrumentor
from opentelemetry.instrumentation.psycopg2 import Psycopg2Instrumentor
from opentelemetry.instrumentation.boto3sqs import Boto3SQSInstrumentor
from opentelemetry.propagate import set_global_textmap
from opentelemetry.trace.propagation.tracecontext import TraceContextTextMapPropagator
from opentelemetry.baggage.propagation import W3CBaggagePropagator
from opentelemetry.trace import Status, StatusCode
from opentelemetry.semconv.trace import SpanAttributes

logger = logging.getLogger(__name__)


class TracingConfig:
    """Configuration for distributed tracing"""
    
    def __init__(
        self,
        service_name: str,
        service_version: str = "1.0.0",
        environment: str = None,
        otlp_endpoint: str = None,
        insecure: bool = True,
        sample_rate: float = 1.0
    ):
        self.service_name = service_name
        self.service_version = service_version
        self.environment = environment or os.environ.get("ENVIRONMENT", "development")
        self.otlp_endpoint = otlp_endpoint or os.environ.get(
            "OTEL_EXPORTER_OTLP_ENDPOINT",
            "otel-collector.observability.svc.cluster.local:4317"
        )
        self.insecure = insecure
        self.sample_rate = sample_rate
        
        # Resource attributes
        self.resource = Resource.create({
            "service.name": service_name,
            "service.version": service_version,
            "service.environment": self.environment,
            "telemetry.sdk.language": "python",
            "telemetry.sdk.name": "opentelemetry",
            "cloud.provider": "aws",
            "cloud.platform": "aws_eks",
            "deployment.environment": self.environment,
            "team.name": os.environ.get("TEAM_NAME", "platform-team"),
            "cell.name": os.environ.get("GRAND_CENTRAL_CELL", "unknown"),
        })


class DistributedTracing:
    """Main distributed tracing implementation"""
    
    def __init__(self, config: TracingConfig):
        self.config = config
        self.tracer_provider = None
        self.meter_provider = None
        self.tracer = None
        self.meter = None
        self._initialized = False
    
    def initialize(self):
        """Initialize tracing and metrics providers"""
        if self._initialized:
            return
        
        try:
            # Setup trace provider
            self.tracer_provider = TracerProvider(
                resource=self.config.resource,
                # Add sampler if needed
            )
            
            # Setup OTLP exporter for traces
            otlp_trace_exporter = OTLPSpanExporter(
                endpoint=self.config.otlp_endpoint,
                insecure=self.config.insecure,
            )
            
            # Add span processor
            span_processor = BatchSpanProcessor(
                otlp_trace_exporter,
                max_queue_size=2048,
                max_export_batch_size=512,
                max_export_timeout_millis=30000,
            )
            self.tracer_provider.add_span_processor(span_processor)
            
            # Set global tracer provider
            trace.set_tracer_provider(self.tracer_provider)
            
            # Setup metrics provider
            metric_reader = PeriodicExportingMetricReader(
                exporter=OTLPMetricExporter(
                    endpoint=self.config.otlp_endpoint,
                    insecure=self.config.insecure,
                ),
                export_interval_millis=60000,  # Export every minute
            )
            
            self.meter_provider = MeterProvider(
                resource=self.config.resource,
                metric_readers=[metric_reader],
            )
            metrics.set_meter_provider(self.meter_provider)
            
            # Get tracer and meter
            self.tracer = trace.get_tracer(
                self.config.service_name,
                self.config.service_version
            )
            self.meter = metrics.get_meter(
                self.config.service_name,
                self.config.service_version
            )
            
            # Setup propagators
            set_global_textmap(
                TraceContextTextMapPropagator()
            )
            
            # Auto-instrument libraries
            self._setup_auto_instrumentation()
            
            self._initialized = True
            logger.info(f"Distributed tracing initialized for {self.config.service_name}")
            
        except Exception as e:
            logger.error(f"Failed to initialize tracing: {e}")
            raise
    
    def _setup_auto_instrumentation(self):
        """Setup automatic instrumentation for common libraries"""
        try:
            # HTTP requests
            RequestsInstrumentor().instrument(
                service_name=self.config.service_name,
                span_attributes={
                    "service.environment": self.config.environment
                }
            )
            
            # Flask
            # Note: Call this after Flask app is created
            # FlaskInstrumentor().instrument()
            
            # Kafka
            KafkaInstrumentor().instrument()
            
            # Redis
            RedisInstrumentor().instrument()
            
            # PostgreSQL
            Psycopg2Instrumentor().instrument()
            
            # AWS SQS
            Boto3SQSInstrumentor().instrument()
            
            logger.info("Auto-instrumentation setup complete")
            
        except Exception as e:
            logger.warning(f"Some auto-instrumentation failed: {e}")
    
    def instrument_flask_app(self, app):
        """Instrument Flask application"""
        FlaskInstrumentor().instrument_app(
            app,
            service_name=self.config.service_name,
            enable_commenter=True,
            commenter_options={
                "service.environment": self.config.environment
            }
        )
    
    @contextmanager
    def trace_operation(
        self,
        operation_name: str,
        attributes: Optional[Dict[str, Any]] = None,
        kind: trace.SpanKind = trace.SpanKind.INTERNAL
    ):
        """
        Context manager for tracing operations
        
        Usage:
            with tracing.trace_operation("process_data", {"batch_size": 100}):
                process_batch(data)
        """
        if not self._initialized:
            yield
            return
        
        with self.tracer.start_as_current_span(
            operation_name,
            kind=kind,
            attributes=attributes or {}
        ) as span:
            try:
                yield span
            except Exception as e:
                span.record_exception(e)
                span.set_status(Status(StatusCode.ERROR, str(e)))
                raise
    
    def trace_function(
        self,
        operation_name: Optional[str] = None,
        attributes: Optional[Dict[str, Any]] = None,
        record_exception: bool = True,
        set_status_on_exception: bool = True
    ):
        """
        Decorator for tracing functions
        
        Usage:
            @tracing.trace_function("process_message", {"queue": "data"})
            def process_message(msg):
                return msg.process()
        """
        def decorator(func):
            @wraps(func)
            def wrapper(*args, **kwargs):
                if not self._initialized:
                    return func(*args, **kwargs)
                
                name = operation_name or f"{func.__module__}.{func.__name__}"
                
                with self.tracer.start_as_current_span(
                    name,
                    attributes=attributes or {}
                ) as span:
                    # Add function parameters as attributes
                    span.set_attribute("function.name", func.__name__)
                    span.set_attribute("function.module", func.__module__)
                    
                    try:
                        result = func(*args, **kwargs)
                        span.set_status(Status(StatusCode.OK))
                        return result
                    except Exception as e:
                        if record_exception:
                            span.record_exception(e)
                        if set_status_on_exception:
                            span.set_status(
                                Status(StatusCode.ERROR, str(e))
                            )
                        raise
            
            return wrapper
        return decorator
    
    def add_baggage(self, key: str, value: str):
        """Add baggage item for context propagation"""
        ctx = baggage.set_baggage(key, value)
        trace.get_current_span().add_event(
            "baggage_added",
            {"baggage.key": key, "baggage.value": value}
        )
        return ctx
    
    def get_baggage(self, key: str) -> Optional[str]:
        """Get baggage item from context"""
        return baggage.get_baggage(key)
    
    def record_error(self, error: Exception, attributes: Optional[Dict[str, Any]] = None):
        """Record an error in the current span"""
        span = trace.get_current_span()
        if span:
            span.record_exception(error, attributes=attributes)
            span.set_status(Status(StatusCode.ERROR, str(error)))
    
    def create_counter(self, name: str, unit: str = "", description: str = ""):
        """Create a counter metric"""
        return self.meter.create_counter(
            name=name,
            unit=unit,
            description=description
        )
    
    def create_histogram(self, name: str, unit: str = "", description: str = ""):
        """Create a histogram metric"""
        return self.meter.create_histogram(
            name=name,
            unit=unit,
            description=description
        )
    
    def create_gauge(self, name: str, unit: str = "", description: str = ""):
        """Create a gauge metric"""
        return self.meter.create_observable_gauge(
            name=name,
            unit=unit,
            description=description
        )


# Singleton instance
_tracing_instance: Optional[DistributedTracing] = None


def setup_tracing(
    service_name: str,
    service_version: str = "1.0.0",
    environment: str = None
) -> DistributedTracing:
    """
    Setup and return the global tracing instance
    
    Args:
        service_name: Name of the service
        service_version: Version of the service
        environment: Deployment environment
    
    Returns:
        DistributedTracing instance
    """
    global _tracing_instance
    
    if _tracing_instance is None:
        config = TracingConfig(
            service_name=service_name,
            service_version=service_version,
            environment=environment
        )
        _tracing_instance = DistributedTracing(config)
        _tracing_instance.initialize()
    
    return _tracing_instance


def get_tracing() -> Optional[DistributedTracing]:
    """Get the global tracing instance"""
    return _tracing_instance


# Convenience decorators
def trace_operation(name: str, **attributes):
    """Convenience decorator for tracing operations"""
    tracing = get_tracing()
    if tracing:
        return tracing.trace_function(name, attributes)
    else:
        # No-op decorator if tracing not initialized
        def decorator(func):
            return func
        return decorator


# Example usage for Clean Platform services
if __name__ == "__main__":
    # Initialize tracing
    tracing = setup_tracing(
        service_name="clean-platform-data-collector",
        service_version="1.0.0",
        environment="production"
    )
    
    # Create metrics
    request_counter = tracing.create_counter(
        "clean_platform_requests_total",
        description="Total requests processed"
    )
    
    request_duration = tracing.create_histogram(
        "clean_platform_request_duration_seconds",
        unit="s",
        description="Request processing duration"
    )
    
    # Example traced function
    @tracing.trace_function("process_data_batch")
    def process_batch(batch_data):
        with tracing.trace_operation("validate_batch", {"batch_size": len(batch_data)}):
            # Validation logic
            pass
        
        with tracing.trace_operation("store_batch", {"batch_size": len(batch_data)}):
            # Storage logic
            pass
        
        # Record metrics
        request_counter.add(len(batch_data), {"status": "success"})
    
    # Example with Flask
    from flask import Flask
    app = Flask(__name__)
    tracing.instrument_flask_app(app)
    
    @app.route("/api/v1/data")
    @trace_operation("handle_data_request")
    def handle_data():
        return {"status": "ok"}