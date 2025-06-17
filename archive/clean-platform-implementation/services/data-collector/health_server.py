"""
Separate health check server running on port 8081
This allows health checks to be independent of the main application
"""

import os
import logging
from flask import Flask, jsonify
import redis
from datetime import datetime
import threading

# Configure logging
logging.basicConfig(
    level=os.getenv('LOG_LEVEL', 'INFO'),
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

# Create health check app
health_app = Flask(__name__)

# Initialize Redis connection for health checks
redis_client = redis.from_url(os.getenv('REDIS_URL', 'redis://localhost:6379'))

@health_app.route('/healthz', methods=['GET'])
def healthz():
    """Kubernetes liveness probe endpoint"""
    try:
        # Basic liveness check - is the service running?
        return jsonify({"status": "ok", "timestamp": datetime.utcnow().isoformat()}), 200
    except Exception as e:
        logger.error(f"Liveness check failed: {e}")
        return jsonify({"status": "error", "message": str(e)}), 503

@health_app.route('/readyz', methods=['GET'])
def readyz():
    """Kubernetes readiness probe endpoint"""
    try:
        # Check if service is ready to accept traffic
        redis_client.ping()
        
        # Check if main app is responsive
        # This could check a shared health status
        
        return jsonify({
            "status": "ready",
            "timestamp": datetime.utcnow().isoformat(),
            "checks": {
                "redis": "ok",
                "app": "ok"
            }
        }), 200
    except redis.ConnectionError:
        logger.error("Redis connection failed")
        return jsonify({
            "status": "not ready",
            "message": "Redis connection failed",
            "timestamp": datetime.utcnow().isoformat()
        }), 503
    except Exception as e:
        logger.error(f"Readiness check failed: {e}")
        return jsonify({
            "status": "not ready",
            "message": str(e),
            "timestamp": datetime.utcnow().isoformat()
        }), 503

@health_app.route('/health', methods=['GET'])
def health():
    """Comprehensive health check endpoint"""
    try:
        # Check Redis
        redis_status = "ok"
        try:
            redis_client.ping()
            queue_length = redis_client.llen('data_queue')
        except:
            redis_status = "error"
            queue_length = -1
        
        # Check system resources
        import psutil
        cpu_percent = psutil.cpu_percent(interval=0.1)
        memory = psutil.virtual_memory()
        
        health_status = {
            "status": "healthy" if redis_status == "ok" else "degraded",
            "timestamp": datetime.utcnow().isoformat(),
            "service": "data-collector",
            "version": os.getenv('SERVICE_VERSION', 'unknown'),
            "checks": {
                "redis": {
                    "status": redis_status,
                    "queue_length": queue_length
                },
                "system": {
                    "cpu_percent": cpu_percent,
                    "memory_percent": memory.percent,
                    "memory_available_mb": memory.available // (1024 * 1024)
                }
            }
        }
        
        status_code = 200 if redis_status == "ok" else 503
        return jsonify(health_status), status_code
        
    except Exception as e:
        logger.error(f"Health check failed: {e}")
        return jsonify({
            "status": "unhealthy",
            "error": str(e),
            "timestamp": datetime.utcnow().isoformat()
        }), 503

@health_app.route('/', methods=['GET'])
def root():
    """Root endpoint for health server"""
    return jsonify({
        "service": "data-collector-health",
        "endpoints": ["/health", "/healthz", "/readyz"],
        "port": 8081
    }), 200

def run_health_server():
    """Run the health check server"""
    logger.info("Starting health check server on port 8081")
    health_app.run(host='0.0.0.0', port=8081, debug=False)

if __name__ == '__main__':
    run_health_server()