"""
Data Collector Service
Handles data ingestion, validation, and queuing for processing
"""

import os
import json
import logging
from datetime import datetime
from flask import Flask, request, jsonify
from prometheus_client import Counter, Histogram, generate_latest
import redis
from marshmallow import Schema, fields, ValidationError

# Configure logging
logging.basicConfig(
    level=os.getenv('LOG_LEVEL', 'INFO'),
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

# Initialize Flask app
app = Flask(__name__)

# Initialize Redis connection
redis_client = redis.from_url(os.getenv('REDIS_URL', 'redis://localhost:6379'))

# Prometheus metrics
requests_total = Counter('data_collector_requests_total', 'Total requests', ['method', 'endpoint', 'status'])
request_duration = Histogram('data_collector_request_duration_seconds', 'Request duration', ['method', 'endpoint'])
data_processed = Counter('data_collector_processed_total', 'Total data points processed', ['data_type'])

# Data validation schema
class DataSchema(Schema):
    timestamp = fields.DateTime(required=True)
    data_type = fields.Str(required=True)
    value = fields.Float(required=True)
    metadata = fields.Dict()

data_schema = DataSchema()

@app.route('/health', methods=['GET'])
def health():
    """Legacy health check endpoint"""
    try:
        # Check Redis connection
        redis_client.ping()
        return jsonify({
            "status": "healthy",
            "timestamp": datetime.utcnow().isoformat(),
            "service": "data-collector"
        }), 200
    except Exception as e:
        logger.error(f"Health check failed: {e}")
        return jsonify({
            "status": "unhealthy",
            "error": str(e)
        }), 503

@app.route('/healthz', methods=['GET'])
def healthz():
    """Kubernetes liveness probe endpoint"""
    try:
        # Basic liveness check - is the service running?
        return jsonify({"status": "ok"}), 200
    except Exception as e:
        logger.error(f"Liveness check failed: {e}")
        return jsonify({"status": "error", "message": str(e)}), 503

@app.route('/readyz', methods=['GET'])
def readyz():
    """Kubernetes readiness probe endpoint"""
    try:
        # Check if service is ready to accept traffic
        redis_client.ping()
        return jsonify({"status": "ready"}), 200
    except Exception as e:
        logger.error(f"Readiness check failed: {e}")
        return jsonify({"status": "not ready", "message": str(e)}), 503

@app.route('/metrics', methods=['GET'])
def metrics():
    """Prometheus metrics endpoint"""
    return generate_latest()

@app.route('/collect', methods=['POST'])
def collect_data():
    """Main data collection endpoint"""
    with request_duration.labels(method='POST', endpoint='/collect').time():
        try:
            # Validate content type
            if request.content_type != 'application/json':
                requests_total.labels(method='POST', endpoint='/collect', status='400').inc()
                return jsonify({"error": "Content-Type must be application/json"}), 400
            
            # Parse and validate data
            data = request.get_json()
            if isinstance(data, list):
                validated_data = [data_schema.load(item) for item in data]
            else:
                validated_data = [data_schema.load(data)]
            
            # Queue data for processing
            for item in validated_data:
                # Add server timestamp
                item['received_at'] = datetime.utcnow().isoformat()
                
                # Push to Redis queue
                redis_client.lpush('data_queue', json.dumps(item, default=str))
                
                # Update metrics
                data_processed.labels(data_type=item['data_type']).inc()
            
            requests_total.labels(method='POST', endpoint='/collect', status='202').inc()
            logger.info(f"Collected {len(validated_data)} data points")
            
            return jsonify({
                "status": "accepted",
                "count": len(validated_data),
                "timestamp": datetime.utcnow().isoformat()
            }), 202
            
        except ValidationError as e:
            requests_total.labels(method='POST', endpoint='/collect', status='400').inc()
            logger.warning(f"Validation error: {e.messages}")
            return jsonify({"error": "Validation failed", "details": e.messages}), 400
            
        except json.JSONDecodeError:
            requests_total.labels(method='POST', endpoint='/collect', status='400').inc()
            return jsonify({"error": "Invalid JSON"}), 400
            
        except Exception as e:
            requests_total.labels(method='POST', endpoint='/collect', status='500').inc()
            logger.error(f"Error collecting data: {e}")
            return jsonify({"error": "Internal server error"}), 500

@app.route('/stats', methods=['GET'])
def stats():
    """Get collection statistics"""
    try:
        queue_length = redis_client.llen('data_queue')
        return jsonify({
            "queue_length": queue_length,
            "timestamp": datetime.utcnow().isoformat()
        }), 200
    except Exception as e:
        logger.error(f"Error getting stats: {e}")
        return jsonify({"error": "Internal server error"}), 500

@app.errorhandler(404)
def not_found(e):
    requests_total.labels(method=request.method, endpoint='unknown', status='404').inc()
    return jsonify({"error": "Not found"}), 404

@app.errorhandler(500)
def internal_error(e):
    requests_total.labels(method=request.method, endpoint=request.path, status='500').inc()
    logger.error(f"Internal error: {e}")
    return jsonify({"error": "Internal server error"}), 500

if __name__ == '__main__':
    # Development server
    # Note: In production, health checks run on separate port 8081 via gunicorn config
    app.run(host='0.0.0.0', port=8080, debug=os.getenv('DEBUG', 'false').lower() == 'true')