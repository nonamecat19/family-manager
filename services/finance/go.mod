module github.com/nnc/family-manager/services/finance

go 1.25.0

// The monorepo's modules are never published: go.work covers builds inside the repo, and
// these replaces keep `go mod tidy` and single-module builds working too.
replace (
	github.com/nnc/family-manager/libs/go/auth => ../../libs/go/auth
	github.com/nnc/family-manager/libs/go/database => ../../libs/go/database
	github.com/nnc/family-manager/libs/go/events => ../../libs/go/events
	github.com/nnc/family-manager/libs/go/logger => ../../libs/go/logger
	github.com/nnc/family-manager/libs/go/rpc => ../../libs/go/rpc
	github.com/nnc/family-manager/sdk/go => ../../sdk/go
)

require (
	connectrpc.com/connect v1.20.0
	github.com/jackc/pgx/v5 v5.10.0
	github.com/nnc/family-manager/libs/go/auth v0.0.0
	github.com/nnc/family-manager/libs/go/database v0.0.0
	github.com/nnc/family-manager/libs/go/events v0.0.0
	github.com/nnc/family-manager/libs/go/logger v0.0.0
	github.com/nnc/family-manager/libs/go/rpc v0.0.0
	github.com/nnc/family-manager/sdk/go v0.0.0
	github.com/spf13/viper v1.21.0
	google.golang.org/protobuf v1.36.12
)

require (
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.5.0 // indirect
	github.com/golang-jwt/jwt/v5 v5.3.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/klauspost/compress v1.19.2 // indirect
	github.com/nats-io/nats.go v1.53.1 // indirect
	github.com/nats-io/nkeys v0.4.15 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/pelletier/go-toml/v2 v2.3.1 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	github.com/sagikazarmark/locafero v0.11.0 // indirect
	github.com/sourcegraph/conc v0.3.1-0.20240121214520-5f936abd7ae8 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/crypto v0.55.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.41.0 // indirect
)
