FROM python:3.11-slim

WORKDIR /app

# Install dependencies
RUN pip install flask requests opentelemetry-api opentelemetry-sdk \
    opentelemetry-instrumentation-flask opentelemetry-exporter-otlp \
    opentelemetry-instrumentation-requests

COPY app.py .

EXPOSE 5000

CMD ["python", "app.py"]