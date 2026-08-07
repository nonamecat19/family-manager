package grpc

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server implements the AuthService gRPC interface.
// Replace the embedded placeholder once protobuf is generated.
type Server struct{}

func New() *Server {
	return &Server{}
}

func Register(s *grpc.Server, srv *Server) {
	// Once proto is generated, replace with:
	// authpb.RegisterAuthServiceServer(s, srv)
	_ = s
	_ = srv
}

func (s *Server) Login(_ context.Context, _ interface{}) (interface{}, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (s *Server) Register(_ context.Context, _ interface{}) (interface{}, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}
