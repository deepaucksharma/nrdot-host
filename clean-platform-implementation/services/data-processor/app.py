import os
import json
import logging
import time
from datetime import datetime, timezone
from typing import Dict, Any, List, Optional
import asyncio
from concurrent.futures import ThreadPoolExecutor

from flask import Flask, request, jsonify, Response
from flask_cors import CORS
from prometheus_client import Counter, Histogram, Gauge, generate_latest
import redis
from sqlalchemy import create_engine, text
from sqlalchemy.pool import QueuePool
import pandas as pd
import numpy as np
from marshmallow import Schema, fields, ValidationError
from celery import Celery
from pythonjsonlogger import jsonlogger

# Configure structured logging
logHandler = logging.StreamHandler()
formatter = jsonlogger.JsonFormatter()
logHandler.setFormatter(formatter)
logger = logging.getLogger()
logger.addHandler(logHandler)
logger.setLevel(os.getenv('LOG_LEVEL', 'INFO').upper())

# Initialize Flask app
app = Flask(__name__)
CORS(app)

# Configuration
DATABASE_URL = os.getenv('DATABASE_URL', 'postgresql://postgres:postgres@localhost:5432/platform')
REDIS_URL = os.getenv('REDIS_URL', 'redis://localhost:6379')
CELERY_BROKER_URL = os.getenv('CELERY_BROKER_URL', REDIS_URL)
CELERY_RESULT_BACKEND = os.getenv('CELERY_RESULT_BACKEND', REDIS_URL)

# Initialize connections
redis_client = redis.from_url(REDIS_URL, decode_responses=True)
db_engine = create_engine(
    DATABASE_URL,
    poolclass=QueuePool,
    pool_size=20,
    max_overflow=40,
    pool_pre_ping=True,
    pool_recycle=3600
)

# Initialize Celery
celery_app = Celery('data_processor', broker=CELERY_BROKER_URL, backend=CELERY_RESULT_BACKEND)
celery_app.conf.update(
    task_serializer='json',
    accept_content=['json'],
    result_serializer='json',
    timezone='UTC',
    enable_utc=True,
    task_track_started=True,
    task_time_limit=300,
    task_soft_time_limit=270,
    worker_prefetch_multiplier=1,
    worker_max_tasks_per_child=1000,
)

# Metrics
process_counter = Counter('data_processor_requests_total', 'Total requests', ['method', 'endpoint', 'status'])
process_histogram = Histogram('data_processor_request_duration_seconds', 'Request duration', ['method', 'endpoint'])
active_tasks = Gauge('data_processor_active_tasks', 'Number of active processing tasks')
processing_errors = Counter('data_processor_errors_total', 'Total processing errors', ['error_type'])
cache_hits = Counter('data_processor_cache_hits_total', 'Cache hits')
cache_misses = Counter('data_processor_cache_misses_total', 'Cache misses')

# Thread pool for async operations
executor = ThreadPoolExecutor(max_workers=10)

# Validation schemas
class ProcessRequestSchema(Schema):
    data_id = fields.Str(required=True)
    processing_type = fields.Str(required=True)
    parameters = fields.Dict(missing={})
    priority = fields.Int(missing=5, validate=lambda x: 1 <= x <= 10)

class QueryRequestSchema(Schema):
    query_type = fields.Str(required=True)
    filters = fields.Dict(missing={})
    start_time = fields.DateTime(missing=None)
    end_time = fields.DateTime(missing=None)
    limit = fields.Int(missing=100, validate=lambda x: 1 <= x <= 1000)
    offset = fields.Int(missing=0, validate=lambda x: x >= 0)

process_schema = ProcessRequestSchema()
query_schema = QueryRequestSchema()

# Celery tasks
@celery_app.task(bind=True, name='process_data')
def process_data_task(self, data_id: str, processing_type: str, parameters: Dict[str, Any]) -> Dict[str, Any]:
    """Background task for data processing"""
    active_tasks.inc()
    start_time = time.time()
    
    try:
        logger.info(f"Processing data: {data_id}, type: {processing_type}")
        
        # Simulate different processing types
        if processing_type == 'aggregate':
            result = aggregate_data(data_id, parameters)
        elif processing_type == 'transform':
            result = transform_data(data_id, parameters)
        elif processing_type == 'analyze':
            result = analyze_data(data_id, parameters)
        else:
            raise ValueError(f"Unknown processing type: {processing_type}")
        
        # Store result
        result_key = f"result:{data_id}:{processing_type}"
        redis_client.setex(result_key, 3600, json.dumps(result))
        
        duration = time.time() - start_time
        logger.info(f"Processing completed: {data_id}, duration: {duration:.2f}s")
        
        return {
            'status': 'completed',
            'data_id': data_id,
            'processing_type': processing_type,
            'duration': duration,
            'result_key': result_key
        }
        
    except Exception as e:
        logger.error(f"Processing failed: {data_id}", exc_info=True)
        processing_errors.labels(error_type=type(e).__name__).inc()
        raise
    finally:
        active_tasks.dec()

def aggregate_data(data_id: str, parameters: Dict[str, Any]) -> Dict[str, Any]:
    """Aggregate data based on parameters"""
    # Fetch data from database
    with db_engine.connect() as conn:
        query = text("""
            SELECT timestamp, value, tags
            FROM data_points
            WHERE data_id = :data_id
            AND timestamp >= NOW() - INTERVAL '1 hour'
            ORDER BY timestamp
        """)
        result = conn.execute(query, {'data_id': data_id})
        df = pd.DataFrame(result.fetchall(), columns=['timestamp', 'value', 'tags'])
    
    if df.empty:
        return {'aggregates': {}, 'count': 0}
    
    # Perform aggregations
    aggregates = {
        'count': len(df),
        'sum': float(df['value'].sum()),
        'mean': float(df['value'].mean()),
        'std': float(df['value'].std()),
        'min': float(df['value'].min()),
        'max': float(df['value'].max()),
        'percentiles': {
            'p50': float(df['value'].quantile(0.5)),
            'p90': float(df['value'].quantile(0.9)),
            'p95': float(df['value'].quantile(0.95)),
            'p99': float(df['value'].quantile(0.99))
        }
    }
    
    # Time-based aggregations if requested
    if parameters.get('time_bucket'):
        bucket = parameters['time_bucket']
        df['timestamp'] = pd.to_datetime(df['timestamp'])
        df.set_index('timestamp', inplace=True)
        time_series = df['value'].resample(bucket).agg(['mean', 'sum', 'count'])
        aggregates['time_series'] = time_series.to_dict('index')
    
    return {'aggregates': aggregates, 'parameters': parameters}

def transform_data(data_id: str, parameters: Dict[str, Any]) -> Dict[str, Any]:
    """Transform data based on parameters"""
    # This would contain actual transformation logic
    transformations = parameters.get('transformations', [])
    
    # Example transformations
    results = {
        'data_id': data_id,
        'transformations_applied': transformations,
        'output_format': parameters.get('output_format', 'json'),
        'timestamp': datetime.now(timezone.utc).isoformat()
    }
    
    return results

def analyze_data(data_id: str, parameters: Dict[str, Any]) -> Dict[str, Any]:
    """Analyze data using ML/statistical methods"""
    analysis_type = parameters.get('analysis_type', 'basic')
    
    # This would contain actual analysis logic
    results = {
        'data_id': data_id,
        'analysis_type': analysis_type,
        'insights': [],
        'recommendations': [],
        'confidence_scores': {},
        'timestamp': datetime.now(timezone.utc).isoformat()
    }
    
    return results

# API endpoints
@app.route('/health')
def health():
    """Health check endpoint"""
    checks = {
        'status': 'healthy',
        'timestamp': datetime.now(timezone.utc).isoformat(),
        'version': os.getenv('VERSION', '1.0.0')
    }
    
    # Check Redis
    try:
        redis_client.ping()
        checks['redis'] = 'healthy'
    except Exception:
        checks['redis'] = 'unhealthy'
        checks['status'] = 'degraded'
    
    # Check Database
    try:
        with db_engine.connect() as conn:
            conn.execute(text('SELECT 1'))
        checks['database'] = 'healthy'
    except Exception:
        checks['database'] = 'unhealthy'
        checks['status'] = 'unhealthy'
    
    status_code = 200 if checks['status'] == 'healthy' else 503
    return jsonify(checks), status_code

@app.route('/ready')
def ready():
    """Readiness check endpoint"""
    try:
        # Check all dependencies
        redis_client.ping()
        with db_engine.connect() as conn:
            conn.execute(text('SELECT 1'))
        
        # Check Celery
        i = celery_app.control.inspect()
        stats = i.stats()
        if not stats:
            raise Exception("No Celery workers available")
        
        return jsonify({'status': 'ready'}), 200
    except Exception as e:
        logger.error(f"Readiness check failed: {e}")
        return jsonify({'status': 'not ready', 'error': str(e)}), 503

@app.route('/metrics')
def metrics():
    """Prometheus metrics endpoint"""
    return Response(generate_latest(), mimetype='text/plain')

@app.route('/process', methods=['POST'])
def process():
    """Process data endpoint"""
    with process_histogram.labels(method='POST', endpoint='/process').time():
        try:
            # Validate request
            data = process_schema.load(request.json)
            
            # Check cache first
            cache_key = f"processing:{data['data_id']}:{data['processing_type']}"
            cached_result = redis_client.get(cache_key)
            
            if cached_result:
                cache_hits.inc()
                process_counter.labels(method='POST', endpoint='/process', status='200').inc()
                return jsonify(json.loads(cached_result)), 200
            
            cache_misses.inc()
            
            # Submit to Celery
            task = process_data_task.apply_async(
                args=[data['data_id'], data['processing_type'], data['parameters']],
                priority=data['priority']
            )
            
            # Store task info
            task_info = {
                'task_id': task.id,
                'data_id': data['data_id'],
                'processing_type': data['processing_type'],
                'status': 'queued',
                'submitted_at': datetime.now(timezone.utc).isoformat()
            }
            redis_client.setex(f"task:{task.id}", 3600, json.dumps(task_info))
            
            process_counter.labels(method='POST', endpoint='/process', status='202').inc()
            return jsonify({
                'task_id': task.id,
                'status': 'queued',
                'status_url': f'/process/status/{task.id}'
            }), 202
            
        except ValidationError as e:
            process_counter.labels(method='POST', endpoint='/process', status='400').inc()
            return jsonify({'error': 'Validation error', 'details': e.messages}), 400
        except Exception as e:
            logger.error(f"Process request failed: {e}", exc_info=True)
            processing_errors.labels(error_type=type(e).__name__).inc()
            process_counter.labels(method='POST', endpoint='/process', status='500').inc()
            return jsonify({'error': 'Internal server error'}), 500

@app.route('/process/status/<task_id>')
def process_status(task_id: str):
    """Get processing task status"""
    try:
        # Get task info from Redis
        task_info = redis_client.get(f"task:{task_id}")
        if not task_info:
            return jsonify({'error': 'Task not found'}), 404
        
        task_data = json.loads(task_info)
        
        # Get Celery task status
        task = celery_app.AsyncResult(task_id)
        
        if task.state == 'PENDING':
            task_data['status'] = 'queued'
        elif task.state == 'STARTED':
            task_data['status'] = 'processing'
        elif task.state == 'SUCCESS':
            task_data['status'] = 'completed'
            task_data['result'] = task.result
        elif task.state == 'FAILURE':
            task_data['status'] = 'failed'
            task_data['error'] = str(task.info)
        else:
            task_data['status'] = task.state.lower()
        
        return jsonify(task_data), 200
        
    except Exception as e:
        logger.error(f"Status check failed: {e}", exc_info=True)
        return jsonify({'error': 'Internal server error'}), 500

@app.route('/query', methods=['POST'])
def query():
    """Query processed data"""
    with process_histogram.labels(method='POST', endpoint='/query').time():
        try:
            # Validate request
            data = query_schema.load(request.json)
            
            # Build cache key
            cache_key = f"query:{json.dumps(data, sort_keys=True)}"
            cached_result = redis_client.get(cache_key)
            
            if cached_result:
                cache_hits.inc()
                process_counter.labels(method='POST', endpoint='/query', status='200').inc()
                return jsonify(json.loads(cached_result)), 200
            
            cache_misses.inc()
            
            # Execute query
            results = execute_query(data)
            
            # Cache results
            redis_client.setex(cache_key, 300, json.dumps(results))
            
            process_counter.labels(method='POST', endpoint='/query', status='200').inc()
            return jsonify(results), 200
            
        except ValidationError as e:
            process_counter.labels(method='POST', endpoint='/query', status='400').inc()
            return jsonify({'error': 'Validation error', 'details': e.messages}), 400
        except Exception as e:
            logger.error(f"Query failed: {e}", exc_info=True)
            processing_errors.labels(error_type=type(e).__name__).inc()
            process_counter.labels(method='POST', endpoint='/query', status='500').inc()
            return jsonify({'error': 'Internal server error'}), 500

def execute_query(query_params: Dict[str, Any]) -> Dict[str, Any]:
    """Execute query against processed data"""
    query_type = query_params['query_type']
    filters = query_params['filters']
    limit = query_params['limit']
    offset = query_params['offset']
    
    # Build SQL query based on query type
    if query_type == 'aggregated':
        base_query = """
            SELECT 
                data_id,
                processing_type,
                result,
                created_at
            FROM processed_results
            WHERE 1=1
        """
    elif query_type == 'raw':
        base_query = """
            SELECT 
                data_id,
                timestamp,
                value,
                tags
            FROM data_points
            WHERE 1=1
        """
    else:
        raise ValueError(f"Unknown query type: {query_type}")
    
    # Add filters
    params = {}
    conditions = []
    
    if query_params.get('start_time'):
        conditions.append("timestamp >= :start_time")
        params['start_time'] = query_params['start_time']
    
    if query_params.get('end_time'):
        conditions.append("timestamp <= :end_time")
        params['end_time'] = query_params['end_time']
    
    for key, value in filters.items():
        conditions.append(f"{key} = :{key}")
        params[key] = value
    
    if conditions:
        base_query += " AND " + " AND ".join(conditions)
    
    # Add pagination
    base_query += f" ORDER BY timestamp DESC LIMIT :limit OFFSET :offset"
    params['limit'] = limit
    params['offset'] = offset
    
    # Execute query
    with db_engine.connect() as conn:
        result = conn.execute(text(base_query), params)
        rows = result.fetchall()
        columns = result.keys()
    
    # Format results
    data = [dict(zip(columns, row)) for row in rows]
    
    # Convert datetime objects to ISO format
    for item in data:
        for key, value in item.items():
            if isinstance(value, datetime):
                item[key] = value.isoformat()
    
    return {
        'query_type': query_type,
        'count': len(data),
        'limit': limit,
        'offset': offset,
        'data': data
    }

@app.route('/stream')
def stream():
    """WebSocket endpoint for real-time updates"""
    # This would be implemented with actual WebSocket library
    return jsonify({'message': 'WebSocket endpoint - use ws:// protocol'}), 200

@app.errorhandler(404)
def not_found(e):
    return jsonify({'error': 'Resource not found'}), 404

@app.errorhandler(500)
def internal_error(e):
    logger.error(f"Internal server error: {e}", exc_info=True)
    return jsonify({'error': 'Internal server error'}), 500

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=8080, debug=False)