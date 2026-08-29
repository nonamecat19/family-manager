module github.com/nnc/family-manager/libs/go/events

go 1.25.0

require (
	github.com/nats-io/nats.go v1.38.0
	google.golang.org/protobuf v1.36.12
)

require (
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/nats-io/nkeys v0.4.9 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/nnc/family-manager/libs/go/logger v0.0.0
	golang.org/x/crypto v0.51.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
	golang.org/x/text v0.39.0 // indirect
)

replace github.com/nnc/family-manager/libs/go/logger => ../logger
