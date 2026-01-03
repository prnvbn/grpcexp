package grpc

import (
	"context"
	"fmt"
	"sort"

	"github.com/fullstorydev/grpcurl"
	"github.com/jhump/protoreflect/desc"
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
	*grpcreflect.Client
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
	return &Client{refClient}, nil
}

func (c *Client) ListServices() ([]string, error) {
	services, err := c.Client.ListServices()
	if err != nil {
		return nil, err
	}
	sort.Strings(services)
	return services, nil
}

func (c *Client) ListMethods(fullyQualifiedName string) ([]string, error) {
	file, err := c.FileContainingSymbol(fullyQualifiedName)
	if err != nil {
		return nil, err
	}

	descriptor := file.FindSymbol(fullyQualifiedName)
	sd, ok := descriptor.(*desc.ServiceDescriptor)
	if !ok {
		return nil, fmt.Errorf("Service Descriptor not found for %s", fullyQualifiedName)
	}

	methods := make([]string, 0, len(sd.GetMethods()))
	for _, method := range sd.GetMethods() {
		methods = append(methods, method.GetFullyQualifiedName())
	}
	sort.Strings(methods)
	return methods, nil
}
