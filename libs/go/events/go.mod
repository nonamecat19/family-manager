module github.com/nnc/family-manager/libs/go/events

go 1.25.0

require (
	github.com/nats-io/nats.go v1.53.1
	google.golang.org/protobuf v1.36.12
)

require (
	github.com/klauspost/compress v1.19.2 // indirect
	github.com/nats-io/nkeys v0.4.15 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/nnc/family-manager/libs/go/logger v0.0.0
	golang.org/x/crypto v0.55.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
)

replace github.com/nnc/family-manager/libs/go/logger => ../logger
