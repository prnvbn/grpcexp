package grpc

import (
	"fmt"

	"github.com/fullstorydev/grpcurl"
	"github.com/jhump/protoreflect/desc" //nolint:staticcheck // Required by grpcurl's descriptor API
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/protoadapt"
)

type StreamEventKind int

const (
	StreamEventResponse StreamEventKind = iota
	StreamEventError
	StreamEventClosed
)

type StreamEvent struct {
	Kind    StreamEventKind
	Message string
	Err     error
}

var _ grpcurl.InvocationEventHandler = &streamEventHandler{}

type streamEventHandler struct {
	formatter grpcurl.Formatter
	events    chan<- StreamEvent
	status    *status.Status
	count     int
}

func (h *streamEventHandler) OnResolveMethod(_ *desc.MethodDescriptor) {}

func (h *streamEventHandler) OnSendHeaders(_ metadata.MD) {}

func (h *streamEventHandler) OnReceiveHeaders(_ metadata.MD) {}

func (h *streamEventHandler) OnReceiveResponse(resp protoadapt.MessageV1) {
	h.count++
	respStr, err := h.formatter(resp)
	if err != nil {
		h.events <- StreamEvent{Kind: StreamEventError, Err: fmt.Errorf("failed to format response message %d: %w", h.count, err)}
		return
	}
	h.events <- StreamEvent{Kind: StreamEventResponse, Message: respStr}
}

func (h *streamEventHandler) OnReceiveTrailers(stat *status.Status, _ metadata.MD) {
	h.status = stat
}
