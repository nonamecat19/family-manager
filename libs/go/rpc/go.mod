module github.com/nnc/family-manager/libs/go/rpc

go 1.23

require connectrpc.com/connect v1.18.1

require (
	github.com/nnc/family-manager/libs/go/logger v0.0.0
	google.golang.org/protobuf v1.34.2 // indirect
)

replace github.com/nnc/family-manager/libs/go/logger => ../logger
