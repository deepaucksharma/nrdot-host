module github.com/newrelic/nrdot-host/nrdot-api-server

go 1.21

require (
	github.com/gorilla/mux v1.8.1
	github.com/stretchr/testify v1.9.0
	go.uber.org/zap v1.27.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
    github.com/newrelic/nrdot-host/nrdot-api-server => ../nrdot-api-server
    github.com/newrelic/nrdot-host/nrdot-common => ../nrdot-common
    github.com/newrelic/nrdot-host/nrdot-config-engine => ../nrdot-config-engine
    github.com/newrelic/nrdot-host/nrdot-schema => ../nrdot-schema
    github.com/newrelic/nrdot-host/nrdot-telemetry-client => ../nrdot-telemetry-client
    github.com/newrelic/nrdot-host/nrdot-template-lib => ../nrdot-template-lib
)
