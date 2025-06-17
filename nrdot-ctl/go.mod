module github.com/newrelic/nrdot-host/nrdot-ctl

go 1.21.0

require (
	github.com/briandowns/spinner v1.23.0
	github.com/fatih/color v1.16.0
	github.com/olekukonko/tablewriter v0.0.5
	github.com/spf13/cobra v1.8.0
	github.com/spf13/viper v1.20.1
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/fsnotify/fsnotify v1.8.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.2.1 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.9 // indirect
	github.com/pelletier/go-toml/v2 v2.2.3 // indirect
	github.com/sagikazarmark/locafero v0.7.0 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/afero v1.12.0 // indirect
	github.com/spf13/cast v1.7.1 // indirect
	github.com/spf13/pflag v1.0.6 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/sys v0.29.0 // indirect
	golang.org/x/term v0.1.0 // indirect
	golang.org/x/text v0.21.0 // indirect
)

replace (
	github.com/newrelic/nrdot-host/nrdot-api-server => ../nrdot-api-server
	github.com/newrelic/nrdot-host/nrdot-schema => ../nrdot-schema
	github.com/newrelic/nrdot-host/nrdot-template-lib => ../nrdot-template-lib
)
