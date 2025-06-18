FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY . .
RUN go mod init backend && \
    go get go.opentelemetry.io/otel && \
    go get go.opentelemetry.io/otel/exporters/otlp/otlptrace && \
    go get go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc && \
    go get go.opentelemetry.io/otel/sdk/trace && \
    go get go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp && \
    go get github.com/lib/pq && \
    go get github.com/go-redis/redis/v8 && \
    go build -o backend main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/backend .
EXPOSE 8081
CMD ["./backend"]