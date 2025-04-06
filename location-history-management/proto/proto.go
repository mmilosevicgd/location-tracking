package proto

import (
	context "context"
	"log"

	grpc "google.golang.org/grpc"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

type GRPCClient interface {
	Close() error
	UpdateUserLocation(ctx context.Context, in *LocationInfo, opts ...grpc.CallOption) (*emptypb.Empty, error)
}

type LHMGRPCClient struct {
	connection *grpc.ClientConn
	client     LocationHistoryManagementClient
}

// Close closes the grpc connection
func (c *LHMGRPCClient) Close() error {
	return c.connection.Close()
}

// UpdateUserLocation updates the user location using the grpc client
func (c *LHMGRPCClient) UpdateUserLocation(ctx context.Context, in *LocationInfo, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	return c.client.UpdateUserLocation(ctx, in, opts...)
}

// CreateClient creates a new grpc client for the location history management service
func CreateClient(target string, opts ...grpc.DialOption) (*LHMGRPCClient, error) {
	connection, err := grpc.NewClient(target, opts...)

	if err != nil {
		return nil, err
	}

	return &LHMGRPCClient{
		connection: connection,
		client:     NewLocationHistoryManagementClient(connection),
	}, nil
}

// MustCreateClient creates a new grpc client for the location history management service and panics if it fails
func MustCreateClient(uri string, opts ...grpc.DialOption) *LHMGRPCClient {
	client, err := CreateClient(uri, opts...)

	if err != nil {
		log.Fatalf("failed to create location history management client for uri '%s' and options '%v': %v\n", uri, opts, err)
	}

	return client
}
