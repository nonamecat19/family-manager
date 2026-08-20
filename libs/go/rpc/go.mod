module github.com/nnc/family-manager/libs/go/rpc

go 1.25.0

require connectrpc.com/connect v1.18.1

require (
	golang.org/x/net v0.55.0 // indirect
	golang.org/x/text v0.39.0 // indirect
)

require (
	github.com/nnc/family-manager/libs/go/logger v0.0.0
	google.golang.org/protobuf v1.36.4 // indirect
)

replace github.com/nnc/family-manager/libs/go/logger => ../logger
