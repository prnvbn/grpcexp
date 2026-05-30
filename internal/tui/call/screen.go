package call

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/prnvbn/grpcexp/internal/grpc"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type Screen interface {
	tea.Model
	SetSize(width, height int)
	AcceptsTextInput() bool
	Cancel()
}

func NewScreen(method protoreflect.MethodDescriptor, client *grpc.Client) Screen {
	if method.IsStreamingClient() || method.IsStreamingServer() {
		return NewStream(method, client)
	}
	return NewUnary(method, client)
}
