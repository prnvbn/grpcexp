package grpc

import (
	"context"

	"github.com/fullstorydev/grpcurl"
	"github.com/jhump/protoreflect/grpcreflect"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type Config struct {
	Target    string
	Creds     credentials.TransportCredentials
	UserAgent string
}

type Client struct {
	client *grpcreflect.Client
}

func NewClient(ctx context.Context, config Config) (*Client, error) {
	opts := []grpc.DialOption{
		grpc.WithUserAgent(config.UserAgent),
	}
	cc, err := grpcurl.BlockingDial(ctx, "", config.Target, config.Creds, opts...)
	if err != nil {
		return nil, err
	}

	refClient := grpcreflect.NewClientAuto(ctx, cc)
	refClient.AllowMissingFileDescriptors()
	return &Client{client: refClient}, nil
}

func (c *Client) ListServices() ([]string, error) {
	return c.client.ListServices()
}
