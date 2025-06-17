"""
Gunicorn configuration for data-collector service
Runs main app on port 8080 and health checks on port 8081
"""

import os
import multiprocessing

# Server socket
bind = ["0.0.0.0:8080", "0.0.0.0:8081"]

# Worker processes
workers = int(os.getenv('GUNICORN_WORKERS', multiprocessing.cpu_count() * 2 + 1))
worker_class = 'sync'
worker_connections = 1000

# Threading
threads = int(os.getenv('GUNICORN_THREADS', 4))

# Timeout
timeout = 30
keepalive = 2

# Logging
accesslog = '-'
errorlog = '-'
loglevel = os.getenv('LOG_LEVEL', 'info').lower()
access_log_format = '%(h)s %(l)s %(u)s %(t)s "%(r)s" %(s)s %(b)s "%(f)s" "%(a)s" %(D)s'

# Process naming
proc_name = 'data-collector'

# Server mechanics
daemon = False
pidfile = None
umask = 0
user = None
group = None
tmp_upload_dir = None

# SSL (if needed)
keyfile = None
certfile = None

# Stats
statsd_host = os.getenv('STATSD_HOST', None)
if statsd_host:
    statsd_prefix = 'gunicorn.data-collector'

# Server hooks
def pre_fork(server, worker):
    """Called just before a worker is forked."""
    server.log.info(f"Worker spawning (pid: {worker.pid})")

def post_fork(server, worker):
    """Called just after a worker has been forked."""
    server.log.info(f"Worker spawned (pid: {worker.pid})")

def worker_int(worker):
    """Called just after a worker exited on SIGINT or SIGQUIT."""
    worker.log.info(f"Worker interrupted (pid: {worker.pid})")

def pre_exec(server):
    """Called just before a new master process is forked."""
    server.log.info("Forked child, re-executing.")

def when_ready(server):
    """Called just after the server is started."""
    server.log.info("Server is ready. Spawning workers")

def worker_abort(worker):
    """Called when a worker received the SIGABRT signal."""
    worker.log.info(f"Worker aborted (pid: {worker.pid})")

def pre_request(worker, req):
    """Called just before a worker processes the request."""
    worker.log.debug(f"{req.method} {req.path}")

def post_request(worker, req, environ, resp):
    """Called after a worker processes the request."""
    worker.log.debug(f"Request processed: {req.method} {req.path} - {resp.status}")

def child_exit(server, worker):
    """Called just after a worker has been exited."""
    server.log.info(f"Worker exited (pid: {worker.pid})")

def on_exit(server):
    """Called just before master process exits."""
    server.log.info("Shutting down: Master")