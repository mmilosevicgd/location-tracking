package proto

import (
	context "context"

	grpc "google.golang.org/grpc"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

type MockGRPCClient struct {
}

func (m *MockGRPCClient) Close() error {
	return nil
}

func (m *MockGRPCClient) UpdateUserLocation(ctx context.Context, in *LocationInfo, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

// CreateMockGRPCClient creates a new mock grpc client
func CreateMockGRPCClient() *MockGRPCClient {
	return &MockGRPCClient{}
}
