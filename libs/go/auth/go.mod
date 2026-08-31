module github.com/nnc/family-manager/libs/go/auth

go 1.25.0

require (
	connectrpc.com/connect v1.20.0
	github.com/golang-jwt/jwt/v5 v5.3.1
)

require (
	github.com/nnc/family-manager/libs/go/logger v0.0.0
	google.golang.org/protobuf v1.36.12 // indirect
)

replace github.com/nnc/family-manager/libs/go/logger => ../logger
