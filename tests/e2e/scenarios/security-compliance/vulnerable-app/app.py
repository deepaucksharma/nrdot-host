import os
import logging
import json
import random
from flask import Flask, jsonify, request
from datetime import datetime

from opentelemetry import trace
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
from opentelemetry.instrumentation.flask import FlaskInstrumentor
from opentelemetry.instrumentation.requests import RequestsInstrumentor
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.sdk.resources import Resource

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Initialize Flask app
app = Flask(__name__)

# Configure OpenTelemetry
resource = Resource.create({
    "service.name": os.getenv("OTEL_SERVICE_NAME", "vulnerable-app"),
    "service.version": "1.0.0",
    "deployment.environment": "security-test"
})

provider = TracerProvider(resource=resource)
processor = BatchSpanProcessor(
    OTLPSpanExporter(endpoint=os.getenv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317"))
)
provider.add_span_processor(processor)
trace.set_tracer_provider(provider)
tracer = trace.get_tracer(__name__)

# Instrument Flask
FlaskInstrumentor().instrument_app(app)
RequestsInstrumentor().instrument()

# Get secrets from environment
DB_PASSWORD = os.getenv("DATABASE_PASSWORD")
API_KEY = os.getenv("API_KEY")
AWS_ACCESS_KEY = os.getenv("AWS_ACCESS_KEY_ID")
AWS_SECRET_KEY = os.getenv("AWS_SECRET_ACCESS_KEY")
GITHUB_TOKEN = os.getenv("GITHUB_TOKEN")
STRIPE_KEY = os.getenv("STRIPE_SECRET_KEY")
JWT_SECRET = os.getenv("JWT_SECRET")

@app.route('/')
def home():
    """Home endpoint that logs sensitive information"""
    with tracer.start_as_current_span("home") as span:
        # Log with secrets (should be redacted by NRDOT)
        logger.info(f"User accessed home page with API key: {API_KEY}")
        
        # Add secrets to span attributes (should be redacted)
        span.set_attribute("api.key", API_KEY)
        span.set_attribute("user.authenticated", True)
        
        return jsonify({
            "message": "Welcome to vulnerable app",
            "timestamp": datetime.utcnow().isoformat()
        })

@app.route('/login', methods=['POST'])
def login():
    """Login endpoint that processes credentials"""
    with tracer.start_as_current_span("login") as span:
        data = request.get_json() or {}
        username = data.get('username', 'anonymous')
        password = data.get('password', '')
        
        # Log login attempt with password (should be redacted)
        logger.info(f"Login attempt for user {username} with password: {password}")
        
        # Simulate database check with exposed password
        db_query = f"SELECT * FROM users WHERE username='{username}' AND password='{password}'"
        logger.info(f"Executing query: {db_query}")
        logger.info(f"Using database password: {DB_PASSWORD}")
        
        # Add to trace
        span.set_attribute("db.statement", db_query)
        span.set_attribute("db.password", DB_PASSWORD)
        span.set_attribute("user.name", username)
        
        # Simulate authentication
        if password == "secret123":
            token = f"jwt_{JWT_SECRET}_{username}"
            logger.info(f"Generated JWT token: {token}")
            
            return jsonify({
                "status": "success",
                "token": token,
                "message": f"Welcome {username}!"
            })
        else:
            span.set_status(trace.Status(trace.StatusCode.ERROR, "Invalid credentials"))
            return jsonify({"status": "error", "message": "Invalid credentials"}), 401

@app.route('/api/payment', methods=['POST'])
def payment():
    """Payment endpoint that handles sensitive payment data"""
    with tracer.start_as_current_span("payment") as span:
        data = request.get_json() or {}
        card_number = data.get('card_number', '4111111111111111')
        amount = data.get('amount', 100)
        
        # Log payment processing with sensitive data
        logger.info(f"Processing payment of ${amount} with card: {card_number}")
        logger.info(f"Using Stripe key: {STRIPE_KEY}")
        
        # Add to trace
        span.set_attribute("payment.amount", amount)
        span.set_attribute("payment.card_number", card_number)
        span.set_attribute("payment.stripe_key", STRIPE_KEY)
        
        # Simulate payment processing
        if random.random() > 0.1:  # 90% success rate
            transaction_id = f"txn_{random.randint(10000, 99999)}"
            logger.info(f"Payment successful: {transaction_id}")
            
            return jsonify({
                "status": "success",
                "transaction_id": transaction_id,
                "amount": amount
            })
        else:
            span.set_status(trace.Status(trace.StatusCode.ERROR, "Payment failed"))
            logger.error("Payment processing failed")
            return jsonify({"status": "error", "message": "Payment failed"}), 500

@app.route('/api/aws')
def aws_endpoint():
    """Endpoint that exposes AWS credentials"""
    with tracer.start_as_current_span("aws-operation") as span:
        # Log AWS operations with credentials
        logger.info(f"Accessing AWS with key: {AWS_ACCESS_KEY}")
        logger.info(f"AWS Secret: {AWS_SECRET_KEY}")
        
        # Add to trace
        span.set_attribute("aws.access_key_id", AWS_ACCESS_KEY)
        span.set_attribute("aws.secret_access_key", AWS_SECRET_KEY)
        span.set_attribute("aws.region", "us-east-1")
        
        return jsonify({
            "message": "AWS operation completed",
            "region": "us-east-1",
            "bucket": "my-secure-bucket"
        })

@app.route('/api/github')
def github_endpoint():
    """Endpoint that uses GitHub token"""
    with tracer.start_as_current_span("github-api") as span:
        # Log GitHub API call
        logger.info(f"Calling GitHub API with token: {GITHUB_TOKEN}")
        
        # Add to trace
        span.set_attribute("github.token", GITHUB_TOKEN)
        span.set_attribute("github.api_call", "repos/user/repo")
        
        return jsonify({
            "message": "GitHub API called",
            "repos": ["repo1", "repo2", "repo3"]
        })

@app.route('/health')
def health():
    """Health check endpoint"""
    return jsonify({
        "status": "healthy",
        "service": "vulnerable-app",
        "timestamp": datetime.utcnow().isoformat()
    })

@app.route('/generate-logs')
def generate_logs():
    """Generate various log entries with secrets"""
    with tracer.start_as_current_span("generate-logs") as span:
        # Generate different types of logs with secrets
        logger.info(f"Database connection string: postgresql://user:{DB_PASSWORD}@localhost/db")
        logger.warning(f"API rate limit approaching for key: {API_KEY}")
        logger.error(f"Failed to authenticate with JWT secret: {JWT_SECRET}")
        
        # Log structured data with secrets
        logger.info(json.dumps({
            "event": "api_call",
            "api_key": API_KEY,
            "aws_credentials": {
                "access_key": AWS_ACCESS_KEY,
                "secret_key": AWS_SECRET_KEY
            },
            "timestamp": datetime.utcnow().isoformat()
        }))
        
        # Add various secret patterns to trace
        span.set_attribute("config.database_url", f"postgres://admin:{DB_PASSWORD}@db:5432/prod")
        span.set_attribute("config.redis_url", "redis://:password123@redis:6379")
        span.set_attribute("config.elasticsearch_url", "https://elastic:changeme@es:9200")
        
        return jsonify({
            "message": "Logs generated",
            "count": 4
        })

if __name__ == '__main__':
    # Log startup with secrets (for testing)
    logger.info("Starting vulnerable app...")
    logger.info(f"Environment variables loaded:")
    logger.info(f"  DATABASE_PASSWORD: {DB_PASSWORD}")
    logger.info(f"  API_KEY: {API_KEY}")
    logger.info(f"  AWS_ACCESS_KEY_ID: {AWS_ACCESS_KEY}")
    logger.info(f"  GITHUB_TOKEN: {GITHUB_TOKEN}")
    
    app.run(host='0.0.0.0', port=5000, debug=True)