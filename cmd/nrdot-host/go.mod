module github.com/newrelic/nrdot-host/cmd/nrdot-host

go 1.21

require (
	github.com/gorilla/mux v1.8.1
	github.com/newrelic/nrdot-host/nrdot-common v0.0.0-00010101000000-000000000000
	github.com/newrelic/nrdot-host/nrdot-supervisor v0.0.0-00010101000000-000000000000
	go.uber.org/zap v1.27.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/cenkalti/backoff/v4 v4.2.1 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/golang-jwt/jwt/v5 v5.2.2 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.19.0 // indirect
	github.com/newrelic/nrdot-host/nrdot-api-server v0.0.0-00010101000000-000000000000 // indirect
	github.com/newrelic/nrdot-host/nrdot-config-engine v0.0.0-00010101000000-000000000000 // indirect
	github.com/newrelic/nrdot-host/nrdot-schema v0.0.0 // indirect
	github.com/newrelic/nrdot-host/nrdot-telemetry-client v0.0.0-00010101000000-000000000000 // indirect
	github.com/newrelic/nrdot-host/nrdot-template-lib v0.0.0-00010101000000-000000000000 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20180127040702-4e3ac2762d5f // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/xeipuuv/gojsonschema v1.2.0 // indirect
	go.opentelemetry.io/otel v1.24.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.24.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.24.0 // indirect
	go.opentelemetry.io/otel/metric v1.24.0 // indirect
	go.opentelemetry.io/otel/sdk v1.24.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.24.0 // indirect
	go.opentelemetry.io/otel/trace v1.24.0 // indirect
	go.opentelemetry.io/proto/otlp v1.1.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/net v0.20.0 // indirect
	golang.org/x/sys v0.17.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240123012728-ef4313101c80 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240123012728-ef4313101c80 // indirect
	google.golang.org/grpc v1.62.0 // indirect
	google.golang.org/protobuf v1.32.0 // indirect
)

replace (
	github.com/newrelic/nrdot-host/nrdot-api-server => ../../nrdot-api-server
	github.com/newrelic/nrdot-host/nrdot-common => ../../nrdot-common
	github.com/newrelic/nrdot-host/nrdot-config-engine => ../../nrdot-config-engine
	github.com/newrelic/nrdot-host/nrdot-schema => ../../nrdot-schema
	github.com/newrelic/nrdot-host/nrdot-supervisor => ../../nrdot-supervisor
	github.com/newrelic/nrdot-host/nrdot-telemetry-client => ../../nrdot-telemetry-client
	github.com/newrelic/nrdot-host/nrdot-template-lib => ../../nrdot-template-lib
)
