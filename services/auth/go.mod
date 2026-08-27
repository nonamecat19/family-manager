module github.com/nnc/family-manager/services/auth

go 1.25.0

// The monorepo's modules are never published: go.work covers builds inside the repo, and
// these replaces keep `go mod tidy` and single-module builds working too.
replace (
	github.com/nnc/family-manager/libs/go/auth => ../../libs/go/auth
	github.com/nnc/family-manager/libs/go/database => ../../libs/go/database
	github.com/nnc/family-manager/libs/go/events => ../../libs/go/events
	github.com/nnc/family-manager/libs/go/logger => ../../libs/go/logger
	github.com/nnc/family-manager/sdk/go => ../../sdk/go
)

require (
	connectrpc.com/connect v1.18.1
	github.com/golang-jwt/jwt/v5 v5.2.1
	github.com/jackc/pgx/v5 v5.9.2
	github.com/nnc/family-manager/libs/go/auth v0.0.0
	github.com/nnc/family-manager/libs/go/database v0.0.0
	github.com/nnc/family-manager/libs/go/logger v0.0.0
	github.com/nnc/family-manager/sdk/go v0.0.0
	github.com/spf13/viper v1.19.0
	golang.org/x/crypto v0.51.0
)

require (
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/nnc/family-manager/libs/go/rpc v0.0.0
	github.com/pelletier/go-toml/v2 v2.2.2 // indirect
	github.com/sagikazarmark/locafero v0.4.0 // indirect
	github.com/sagikazarmark/slog-shim v0.1.0 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/afero v1.11.0 // indirect
	github.com/spf13/cast v1.6.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/multierr v1.9.0 // indirect
	golang.org/x/exp v0.0.0-20230905200255-921286631fa9 // indirect
	golang.org/x/sync v0.21.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
	golang.org/x/text v0.39.0 // indirect
	google.golang.org/protobuf v1.36.4 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/nnc/family-manager/libs/go/rpc => ../../libs/go/rpc
