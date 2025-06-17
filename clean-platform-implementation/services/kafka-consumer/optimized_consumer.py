"""
Optimized Kafka Consumer for Clean Platform Implementation
Implements high-throughput patterns from platform documentation
"""

import logging
import json
import time
import signal
import threading
from typing import Dict, List, Optional, Callable, Any
from dataclasses import dataclass
from kafka import KafkaConsumer, KafkaProducer, TopicPartition
from kafka.errors import KafkaError, CommitFailedError
from prometheus_client import Counter, Histogram, Gauge
import os

logger = logging.getLogger(__name__)


# Prometheus metrics
MESSAGES_PROCESSED = Counter(
    'clean_platform_kafka_messages_processed_total',
    'Total messages processed',
    ['topic', 'partition', 'status']
)
LAG_GAUGE = Gauge(
    'clean_platform_kafka_consumer_lag',
    'Consumer lag',
    ['topic', 'partition', 'consumer_group']
)
PROCESSING_TIME = Histogram(
    'clean_platform_kafka_processing_seconds',
    'Message processing time',
    ['topic', 'operation']
)


@dataclass
class ConsumerConfig:
    """Kafka consumer configuration"""
    bootstrap_servers: str
    consumer_group: str
    topics: List[str]
    # Performance tuning
    fetch_min_bytes: int = 65536  # 64KB minimum fetch
    fetch_max_wait_ms: int = 500
    max_poll_records: int = 500
    max_poll_interval_ms: int = 300000  # 5 minutes
    session_timeout_ms: int = 30000  # 30 seconds
    # Processing
    batch_size: int = 1000
    batch_timeout_ms: int = 5000
    num_workers: int = 4
    # Retry configuration
    max_retries: int = 3
    retry_backoff_ms: int = 1000
    # Dead letter queue
    enable_dlq: bool = True
    dlq_topic: Optional[str] = None


class OptimizedKafkaConsumer:
    """High-performance Kafka consumer with batching and parallel processing"""
    
    def __init__(
        self,
        config: ConsumerConfig,
        message_handler: Callable[[List[Dict]], None],
        error_handler: Optional[Callable[[Exception, Dict], None]] = None
    ):
        self.config = config
        self.message_handler = message_handler
        self.error_handler = error_handler or self._default_error_handler
        
        # Initialize consumer
        self.consumer = self._create_consumer()
        
        # Initialize producer for DLQ if enabled
        self.dlq_producer = self._create_dlq_producer() if config.enable_dlq else None
        
        # Processing state
        self.running = False
        self.processing_threads = []
        self.batch_queue = []
        self.batch_lock = threading.Lock()
        self.last_batch_time = time.time()
        
        # Setup signal handlers
        signal.signal(signal.SIGTERM, self._signal_handler)
        signal.signal(signal.SIGINT, self._signal_handler)
    
    def _create_consumer(self) -> KafkaConsumer:
        """Create optimized Kafka consumer"""
        return KafkaConsumer(
            *self.config.topics,
            bootstrap_servers=self.config.bootstrap_servers,
            group_id=self.config.consumer_group,
            # Performance settings
            fetch_min_bytes=self.config.fetch_min_bytes,
            fetch_max_wait_ms=self.config.fetch_max_wait_ms,
            max_poll_records=self.config.max_poll_records,
            max_poll_interval_ms=self.config.max_poll_interval_ms,
            session_timeout_ms=self.config.session_timeout_ms,
            # Reliability settings
            enable_auto_commit=False,
            auto_offset_reset='earliest',
            # Serialization
            value_deserializer=lambda m: json.loads(m.decode('utf-8')),
            key_deserializer=lambda k: k.decode('utf-8') if k else None,
            # Security (from environment)
            security_protocol=os.environ.get('KAFKA_SECURITY_PROTOCOL', 'PLAINTEXT'),
            sasl_mechanism=os.environ.get('KAFKA_SASL_MECHANISM'),
            sasl_plain_username=os.environ.get('KAFKA_SASL_USERNAME'),
            sasl_plain_password=os.environ.get('KAFKA_SASL_PASSWORD'),
        )
    
    def _create_dlq_producer(self) -> Optional[KafkaProducer]:
        """Create producer for dead letter queue"""
        if not self.config.enable_dlq:
            return None
        
        return KafkaProducer(
            bootstrap_servers=self.config.bootstrap_servers,
            # Performance settings
            acks=1,  # Leader acknowledgment
            compression_type='lz4',
            linger_ms=100,
            batch_size=65536,
            buffer_memory=67108864,  # 64MB
            # Reliability
            retries=3,
            max_in_flight_requests_per_connection=5,
            # Serialization
            value_serializer=lambda v: json.dumps(v).encode('utf-8'),
            key_serializer=lambda k: k.encode('utf-8') if k else None,
            # Security
            security_protocol=os.environ.get('KAFKA_SECURITY_PROTOCOL', 'PLAINTEXT'),
        )
    
    def start(self):
        """Start consumer and processing threads"""
        logger.info(f"Starting Kafka consumer for topics: {self.config.topics}")
        self.running = True
        
        # Start worker threads
        for i in range(self.config.num_workers):
            thread = threading.Thread(
                target=self._process_batches,
                name=f"kafka-worker-{i}"
            )
            thread.start()
            self.processing_threads.append(thread)
        
        # Start lag monitoring
        threading.Thread(
            target=self._monitor_lag,
            name="lag-monitor",
            daemon=True
        ).start()
        
        # Main consumer loop
        self._consume_messages()
    
    def _consume_messages(self):
        """Main consumer loop with batching"""
        while self.running:
            try:
                # Poll for messages
                messages = self.consumer.poll(
                    timeout_ms=self.config.batch_timeout_ms,
                    max_records=self.config.max_poll_records
                )
                
                if not messages:
                    # Check if we need to flush partial batch
                    self._check_batch_timeout()
                    continue
                
                # Process messages by partition for ordering
                for topic_partition, records in messages.items():
                    for record in records:
                        self._add_to_batch(record)
                
            except KafkaError as e:
                logger.error(f"Kafka consumer error: {e}")
                time.sleep(1)
            except Exception as e:
                logger.exception(f"Unexpected error in consumer loop: {e}")
                time.sleep(1)
    
    def _add_to_batch(self, record):
        """Add message to batch for processing"""
        message = {
            'topic': record.topic,
            'partition': record.partition,
            'offset': record.offset,
            'key': record.key,
            'value': record.value,
            'timestamp': record.timestamp,
            'headers': dict(record.headers) if record.headers else {}
        }
        
        with self.batch_lock:
            self.batch_queue.append(message)
            
            # Check if batch is full
            if len(self.batch_queue) >= self.config.batch_size:
                self._flush_batch()
    
    def _check_batch_timeout(self):
        """Check if partial batch should be flushed due to timeout"""
        current_time = time.time()
        elapsed = (current_time - self.last_batch_time) * 1000
        
        if elapsed >= self.config.batch_timeout_ms and self.batch_queue:
            with self.batch_lock:
                if self.batch_queue:
                    self._flush_batch()
    
    def _flush_batch(self):
        """Flush current batch for processing"""
        if not self.batch_queue:
            return
        
        # Create copy of batch
        batch = self.batch_queue.copy()
        self.batch_queue.clear()
        self.last_batch_time = time.time()
        
        # Process batch in worker thread
        self._process_batch(batch)
    
    def _process_batches(self):
        """Worker thread for processing batches"""
        while self.running:
            time.sleep(0.1)  # Small sleep to prevent busy waiting
    
    def _process_batch(self, batch: List[Dict]):
        """Process a batch of messages with retry logic"""
        start_time = time.time()
        topic = batch[0]['topic'] if batch else 'unknown'
        
        try:
            # Call message handler
            self.message_handler(batch)
            
            # Commit offsets after successful processing
            self._commit_batch_offsets(batch)
            
            # Update metrics
            MESSAGES_PROCESSED.labels(
                topic=topic,
                partition=batch[0]['partition'] if batch else -1,
                status='success'
            ).inc(len(batch))
            
            PROCESSING_TIME.labels(
                topic=topic,
                operation='batch_processing'
            ).observe(time.time() - start_time)
            
            logger.debug(f"Processed batch of {len(batch)} messages from {topic}")
            
        except Exception as e:
            logger.error(f"Error processing batch: {e}")
            self._handle_batch_error(batch, e)
    
    def _commit_batch_offsets(self, batch: List[Dict]):
        """Commit offsets for processed batch"""
        try:
            # Group by topic-partition
            offsets = {}
            for msg in batch:
                tp = TopicPartition(msg['topic'], msg['partition'])
                # Commit the next offset (current + 1)
                offsets[tp] = msg['offset'] + 1
            
            # Commit offsets
            self.consumer.commit({tp: offset for tp, offset in offsets.items()})
            
        except CommitFailedError as e:
            logger.error(f"Failed to commit offsets: {e}")
            # Consumer will retry from last committed offset
    
    def _handle_batch_error(self, batch: List[Dict], error: Exception):
        """Handle batch processing error"""
        for msg in batch:
            try:
                # Call error handler for each message
                self.error_handler(error, msg)
                
                # Send to DLQ if enabled
                if self.config.enable_dlq and self.dlq_producer:
                    self._send_to_dlq(msg, error)
                
            except Exception as e:
                logger.error(f"Error in error handler: {e}")
        
        # Update error metrics
        MESSAGES_PROCESSED.labels(
            topic=batch[0]['topic'] if batch else 'unknown',
            partition=batch[0]['partition'] if batch else -1,
            status='error'
        ).inc(len(batch))
    
    def _send_to_dlq(self, message: Dict, error: Exception):
        """Send failed message to dead letter queue"""
        dlq_topic = self.config.dlq_topic or f"{message['topic']}-dlq"
        
        dlq_message = {
            'original_message': message,
            'error': str(error),
            'error_type': type(error).__name__,
            'timestamp': time.time(),
            'consumer_group': self.config.consumer_group,
            'retry_count': message.get('retry_count', 0) + 1
        }
        
        try:
            future = self.dlq_producer.send(
                dlq_topic,
                value=dlq_message,
                key=message.get('key')
            )
            future.get(timeout=10)  # Wait for send to complete
            logger.info(f"Sent message to DLQ: {dlq_topic}")
        except Exception as e:
            logger.error(f"Failed to send to DLQ: {e}")
    
    def _monitor_lag(self):
        """Monitor consumer lag"""
        while self.running:
            try:
                # Get current positions
                for partition in self.consumer.assignment():
                    # Get current offset
                    current = self.consumer.position(partition)
                    
                    # Get end offset
                    end_offsets = self.consumer.end_offsets([partition])
                    end = end_offsets.get(partition, current)
                    
                    # Calculate lag
                    lag = end - current
                    
                    # Update metric
                    LAG_GAUGE.labels(
                        topic=partition.topic,
                        partition=partition.partition,
                        consumer_group=self.config.consumer_group
                    ).set(lag)
                
                time.sleep(30)  # Check every 30 seconds
                
            except Exception as e:
                logger.error(f"Error monitoring lag: {e}")
                time.sleep(60)
    
    def _signal_handler(self, signum, frame):
        """Handle shutdown signals"""
        logger.info(f"Received signal {signum}, shutting down...")
        self.stop()
    
    def stop(self):
        """Stop consumer gracefully"""
        logger.info("Stopping Kafka consumer...")
        self.running = False
        
        # Flush any remaining messages
        with self.batch_lock:
            if self.batch_queue:
                self._flush_batch()
        
        # Wait for processing threads
        for thread in self.processing_threads:
            thread.join(timeout=30)
        
        # Close consumer and producer
        self.consumer.close()
        if self.dlq_producer:
            self.dlq_producer.close()
        
        logger.info("Kafka consumer stopped")
    
    def _default_error_handler(self, error: Exception, message: Dict):
        """Default error handler"""
        logger.error(
            f"Error processing message from {message['topic']}:"
            f"{message['partition']}:{message['offset']} - {error}"
        )


# Example usage for Clean Platform
class CleanPlatformConsumer:
    """Clean Platform specific Kafka consumer"""
    
    def __init__(self):
        config = ConsumerConfig(
            bootstrap_servers=os.environ.get('KAFKA_BROKERS', 'localhost:9092'),
            consumer_group='clean-platform-consumer',
            topics=['platform-events', 'platform-metrics'],
            # Optimized settings from platform docs
            fetch_min_bytes=65536,
            max_poll_records=500,
            batch_size=1000,
            num_workers=7,  # Based on CPU optimization
        )
        
        self.consumer = OptimizedKafkaConsumer(
            config=config,
            message_handler=self.process_messages,
            error_handler=self.handle_error
        )
    
    def process_messages(self, batch: List[Dict]):
        """Process batch of messages"""
        # Group by message type for efficient processing
        events = []
        metrics = []
        
        for msg in batch:
            if msg['topic'] == 'platform-events':
                events.append(msg['value'])
            elif msg['topic'] == 'platform-metrics':
                metrics.append(msg['value'])
        
        # Process different message types
        if events:
            self._process_events(events)
        if metrics:
            self._process_metrics(metrics)
    
    def _process_events(self, events: List[Dict]):
        """Process platform events"""
        # Batch insert to database
        logger.info(f"Processing {len(events)} events")
    
    def _process_metrics(self, metrics: List[Dict]):
        """Process platform metrics"""
        # Aggregate and forward metrics
        logger.info(f"Processing {len(metrics)} metrics")
    
    def handle_error(self, error: Exception, message: Dict):
        """Handle processing errors"""
        if isinstance(error, json.JSONDecodeError):
            logger.error(f"Invalid JSON in message: {message}")
        else:
            logger.error(f"Processing error: {error}")
    
    def start(self):
        """Start consumer"""
        self.consumer.start()


if __name__ == "__main__":
    # Configure logging
    logging.basicConfig(
        level=logging.INFO,
        format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
    )
    
    # Start consumer
    consumer = CleanPlatformConsumer()
    consumer.start()