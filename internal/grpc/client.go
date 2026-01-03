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
	"google.golang.org/protobuf/reflect/protoreflect"
)

type Config struct {
	Target    string
	Creds     credentials.TransportCredentials
	UserAgent string
}

type Client struct {
	source grpcurl.DescriptorSource
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
	source := grpcurl.DescriptorSourceFromServer(ctx, refClient) //todo: add support for files as well?

	return &Client{source: source}, nil
}

func (c *Client) ListServices() ([]string, error) {
	svcNames, err := c.source.ListServices()
	if err != nil {
		return nil, err
	}
	sort.Strings(svcNames)
	return svcNames, nil
}

func (c *Client) ListMethods(fullyQualifiedName string) ([]protoreflect.MethodDescriptor, error) {
	descriptor, err := c.source.FindSymbol(fullyQualifiedName)
	if err != nil {
		return nil, err
	}

	sd, ok := descriptor.(*desc.ServiceDescriptor)
	if !ok {
		return nil, fmt.Errorf("Service Descriptor not found for %s", fullyQualifiedName)
	}

	methods := make([]protoreflect.MethodDescriptor, 0, len(sd.GetMethods()))
	for _, method := range sd.GetMethods() {
		methods = append(methods, method.UnwrapMethod())
	}

	sort.Slice(methods, func(i, j int) bool {
		return methods[i].FullName() < methods[j].FullName()
	})

	return methods, nil
}
