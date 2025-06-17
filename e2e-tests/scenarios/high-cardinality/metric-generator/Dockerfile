FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY . .
RUN go mod init metric-generator && \
    go get go.opentelemetry.io/otel && \
    go get go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc && \
    go get go.opentelemetry.io/otel/metric && \
    go get go.opentelemetry.io/otel/sdk/metric && \
    go build -o generator generator.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/generator .
CMD ["./generator"]