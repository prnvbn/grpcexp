package grpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fullstorydev/grpcurl"
	oldproto "github.com/golang/protobuf/proto" //nolint:staticcheck // grpcurl uses the legacy proto API
	"github.com/jhump/protoreflect/desc"        //nolint:staticcheck // Deprecated package but required by grpcurl
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type UnaryResult struct {
	Body     string
	Headers  metadata.MD
	Trailers metadata.MD
	Status   *status.Status

	formattedStatus string
}

func (r UnaryResult) StatusText() string {
	return r.formattedStatus
}

func (r UnaryResult) Failed() bool {
	return r.Status.Code() != codes.OK
}

func (r UnaryResult) HeadersText() string {
	return grpcurl.MetadataToString(r.Headers)
}

func (r UnaryResult) TrailersText() string {
	return grpcurl.MetadataToString(r.Trailers)
}

type unaryEventHandler struct {
	formatter grpcurl.Formatter
	body      bytes.Buffer
	headers   metadata.MD
	trailers  metadata.MD
	status    *status.Status
	formatErr error
}

var _ grpcurl.InvocationEventHandler = &unaryEventHandler{}

func (h *unaryEventHandler) OnResolveMethod(_ *desc.MethodDescriptor) {}

func (h *unaryEventHandler) OnSendHeaders(_ metadata.MD) {}

func (h *unaryEventHandler) OnReceiveHeaders(md metadata.MD) {
	h.headers = md.Copy()
}

func (h *unaryEventHandler) OnReceiveResponse(resp oldproto.Message) {
	response, err := h.formatter(resp)
	if err != nil {
		h.formatErr = fmt.Errorf("failed to format unary response: %w", err)
		return
	}
	h.body.WriteString(response)
	h.body.WriteByte('\n')
}

func (h *unaryEventHandler) OnReceiveTrailers(stat *status.Status, md metadata.MD) {
	h.trailers = md.Copy()
	h.status = stat
}

func (c *Client) InvokeRPC(ctx context.Context, methodFullName string, request map[string]any) (UnaryResult, error) {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return UnaryResult{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	requestData := bytes.NewReader(jsonData)
	rf, formatter, err := grpcurl.RequestParserAndFormatter(grpcurl.FormatJSON, c.source, requestData, grpcurl.FormatOptions{})
	if err != nil {
		return UnaryResult{}, fmt.Errorf("failed to create request parser: %w", err)
	}

	handler := &unaryEventHandler{
		formatter: formatter,
	}

	err = grpcurl.InvokeRPC(ctx, c.source, c.conn, methodFullName, c.config.Headers, handler, rf.Next)
	if err != nil {
		return UnaryResult{}, fmt.Errorf("RPC invocation failed: %w", err)
	}
	if handler.formatErr != nil {
		return UnaryResult{}, handler.formatErr
	}

	var statusBuf bytes.Buffer
	grpcurl.PrintStatus(&statusBuf, handler.status, formatter)

	return UnaryResult{
		Body:            handler.body.String(),
		Headers:         handler.headers,
		Trailers:        handler.trailers,
		Status:          handler.status,
		formattedStatus: strings.TrimRight(statusBuf.String(), "\n"),
	}, nil
}
