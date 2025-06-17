FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY . .
RUN go mod init frontend && \
    go get go.opentelemetry.io/otel && \
    go get go.opentelemetry.io/otel/exporters/otlp/otlptrace && \
    go get go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc && \
    go get go.opentelemetry.io/otel/sdk/trace && \
    go get go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp && \
    go build -o frontend main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/frontend .
COPY --from=builder /app/static ./static
EXPOSE 8080
CMD ["./frontend"]