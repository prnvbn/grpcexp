package grpc

import (
	"bytes"
	"context"
	"encoding/json"
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
	Protoset  string
}

type Client struct {
	source grpcurl.DescriptorSource
	conn   *grpc.ClientConn
}

func NewClient(ctx context.Context, config Config) (*Client, error) {
	opts := []grpc.DialOption{
		grpc.WithUserAgent(config.UserAgent),
	}
	cc, err := grpcurl.BlockingDial(ctx, "", config.Target, config.Creds, opts...)
	if err != nil {
		return nil, err
	}

	var source grpcurl.DescriptorSource
	if config.Protoset != "" {
		source, err = grpcurl.DescriptorSourceFromProtoSets(config.Protoset)
		if err != nil {
			return nil, fmt.Errorf("failed to load protoset file: %w", err)
		}
	} else {
		refClient := grpcreflect.NewClientAuto(ctx, cc)
		refClient.AllowMissingFileDescriptors()
		source = grpcurl.DescriptorSourceFromServer(ctx, refClient)
	}

	return &Client{source: source, conn: cc}, nil
}

func (c *Client) InvokeRPC(ctx context.Context, methodFullName string, request map[string]any) (string, error) {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	requestData := bytes.NewReader(jsonData)
	rf, formatter, err := grpcurl.RequestParserAndFormatter(grpcurl.FormatJSON, c.source, requestData, grpcurl.FormatOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to create request parser: %w", err)
	}

	var responseBuf bytes.Buffer

	handler := &grpcurl.DefaultEventHandler{
		Out:            &responseBuf,
		Formatter:      formatter,
		VerbosityLevel: 0,
	}

	err = grpcurl.InvokeRPC(ctx, c.source, c.conn, methodFullName, nil, handler, rf.Next)
	if err != nil {
		return "", fmt.Errorf("RPC invocation failed: %w", err)
	}

	if handler.Status.Code() != 0 {
		return "", fmt.Errorf("RPC error: %s", handler.Status.Message())
	}

	return responseBuf.String(), nil
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
