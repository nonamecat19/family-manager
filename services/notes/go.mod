module github.com/nnc/family-manager/services/notes

go 1.25.0

// The monorepo's modules are never published: go.work covers builds inside the repo, and
// these replaces keep `go mod tidy` and single-module builds working too.
replace (
	github.com/nnc/family-manager/libs/go/auth => ../../libs/go/auth
	github.com/nnc/family-manager/libs/go/database => ../../libs/go/database
	github.com/nnc/family-manager/libs/go/events => ../../libs/go/events
	github.com/nnc/family-manager/libs/go/logger => ../../libs/go/logger
	github.com/nnc/family-manager/libs/go/storage => ../../libs/go/storage
	github.com/nnc/family-manager/sdk/go => ../../sdk/go
)

require (
	github.com/jackc/pgx/v5 v5.10.0
	github.com/spf13/viper v1.21.0
)

require (
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/go-viper/mapstructure/v2 v2.5.0 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
)

require (
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/pelletier/go-toml/v2 v2.3.1 // indirect
	github.com/sagikazarmark/locafero v0.11.0 // indirect
	github.com/sourcegraph/conc v0.3.1-0.20240121214520-5f936abd7ae8 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.41.0 // indirect
)

replace github.com/nnc/family-manager/libs/go/rpc => ../../libs/go/rpc
