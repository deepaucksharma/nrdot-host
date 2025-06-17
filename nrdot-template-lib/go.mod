module github.com/newrelic/nrdot-host/nrdot-template-lib

go 1.21

require (
	github.com/newrelic/nrdot-host/nrdot-schema v0.0.0
	github.com/stretchr/testify v1.10.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20180127040702-4e3ac2762d5f // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/xeipuuv/gojsonschema v1.2.0 // indirect
)

replace github.com/newrelic/nrdot-host/nrdot-schema => ../nrdot-schema
