package grpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"unicode"

	"github.com/fullstorydev/grpcurl"
	oldproto "github.com/golang/protobuf/proto" //nolint:staticcheck // grpcurl uses the legacy proto API
	"github.com/jhump/protoreflect/desc"        //nolint:staticcheck // Deprecated package but required by grpcurl
	"github.com/jhump/protoreflect/grpcreflect"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type Config struct {
	Target      string
	Creds       credentials.TransportCredentials
	UserAgent   string
	Protoset    string
	ProtoFiles  []string
	ImportPaths []string
	Headers     []string
}

type Client struct {
	source grpcurl.DescriptorSource
	conn   *grpc.ClientConn
	config Config
}

func NewClient(ctx context.Context, config Config) (*Client, error) {
	if err := validateConfig(config); err != nil {
		return nil, err
	}

	source, err := loadFileDescriptorSource(config)
	if err != nil {
		return nil, err
	}

	opts := []grpc.DialOption{
		grpc.WithUserAgent(config.UserAgent),
	}
	cc, err := grpcurl.BlockingDial(ctx, "", config.Target, config.Creds, opts...)
	if err != nil {
		return nil, err
	}

	if source == nil {
		refCtx := context.Background()
		if len(config.Headers) > 0 {
			refCtx = metadata.NewOutgoingContext(refCtx, grpcurl.MetadataFromHeaders(config.Headers))
		}
		refClient := grpcreflect.NewClientAuto(refCtx, cc)
		refClient.AllowMissingFileDescriptors()
		source = grpcurl.DescriptorSourceFromServer(refCtx, refClient)
	}

	return &Client{source: source, conn: cc, config: config}, nil
}

func validateConfig(config Config) error {
	if config.Protoset != "" && len(config.ProtoFiles) > 0 {
		return fmt.Errorf("use either --protoset or --proto, not both")
	}
	if len(config.ImportPaths) > 0 && len(config.ProtoFiles) == 0 {
		return fmt.Errorf("--import-path requires at least one --proto")
	}
	return validateHeaders(config.Headers)
}

func loadFileDescriptorSource(config Config) (grpcurl.DescriptorSource, error) {
	if config.Protoset != "" {
		source, err := grpcurl.DescriptorSourceFromProtoSets(config.Protoset)
		if err != nil {
			return nil, fmt.Errorf("failed to load protoset file: %w", err)
		}
		return source, nil
	}
	if len(config.ProtoFiles) > 0 {
		source, err := grpcurl.DescriptorSourceFromProtoFiles(config.ImportPaths, config.ProtoFiles...)
		if err != nil {
			return nil, fmt.Errorf("failed to load proto source files: %w", err)
		}
		return source, nil
	}
	return nil, nil
}

func validateHeaders(headers []string) error {
	for _, header := range headers {
		name, _, ok := strings.Cut(header, ":")
		if !ok || strings.TrimSpace(name) == "" {
			return fmt.Errorf("invalid header %q: expected 'name: value'", header)
		}
	}
	return nil
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

	err = grpcurl.InvokeRPC(ctx, c.source, c.conn, methodFullName, c.config.Headers, handler, rf.Next)
	if err != nil {
		return "", fmt.Errorf("RPC invocation failed: %w", err)
	}

	if handler.Status.Code() != 0 {
		return "", fmt.Errorf("RPC error: %s", handler.Status.Message())
	}

	return responseBuf.String(), nil
}

func (c *Client) InvokeStreaming(ctx context.Context, methodFullName string, requests <-chan map[string]any, events chan<- StreamEvent) error {
	_, formatter, err := grpcurl.RequestParserAndFormatter(
		grpcurl.FormatJSON,
		c.source,
		bytes.NewReader(nil),
		grpcurl.FormatOptions{},
	)
	if err != nil {
		return fmt.Errorf("failed to create response formatter: %w", err)
	}

	handler := &streamEventHandler{
		formatter: formatter,
		events:    events,
	}

	requestSupplier := func(msg oldproto.Message) error {
		request, ok := <-requests
		if !ok {
			return io.EOF
		}

		jsonData, err := json.Marshal(request)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}

		rf, _, err := grpcurl.RequestParserAndFormatter(grpcurl.FormatJSON, c.source, bytes.NewReader(jsonData), grpcurl.FormatOptions{})
		if err != nil {
			return fmt.Errorf("failed to create request parser: %w", err)
		}

		return rf.Next(msg)
	}

	if err := grpcurl.InvokeRPC(ctx, c.source, c.conn, methodFullName, c.config.Headers, handler, requestSupplier); err != nil {
		events <- StreamEvent{Kind: StreamEventError, Err: fmt.Errorf("RPC invocation failed: %w", err)}
		return err
	}

	if handler.status != nil && handler.status.Code() != codes.OK {
		err := fmt.Errorf("RPC error: %s", handler.status.Message())
		events <- StreamEvent{Kind: StreamEventError, Err: err}
		return err
	}

	events <- StreamEvent{Kind: StreamEventClosed}
	return nil
}

// GRPCURLCommand returns a shell-safe grpcurl command for the current client session.
func (c *Client) GRPCURLCommand(methodFullName string, request map[string]any) (string, error) {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	args := []string{"grpcurl"}
	if c.config.Creds.Info().SecurityProtocol == "insecure" {
		args = append(args, "-plaintext")
	}
	if c.config.Protoset != "" {
		args = append(args, "-protoset", c.config.Protoset)
	}
	for _, importPath := range c.config.ImportPaths {
		args = append(args, "-import-path", importPath)
	}
	for _, protoFile := range c.config.ProtoFiles {
		args = append(args, "-proto", protoFile)
	}
	if c.config.UserAgent != "" {
		args = append(args, "-user-agent", c.config.UserAgent)
	}
	for _, header := range c.config.Headers {
		args = append(args, "-H", header)
	}
	args = append(args, "-d", string(jsonData), c.config.Target, methodFullName)

	for i, arg := range args {
		args[i] = shellQuote(arg)
	}
	return strings.Join(args, " "), nil
}

func shellQuote(s string) string {
	if s != "" && strings.IndexFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && !strings.ContainsRune("_+-=.,/:@", r)
	}) == -1 {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
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
		return nil, fmt.Errorf("service descriptor not found for %s", fullyQualifiedName)
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
