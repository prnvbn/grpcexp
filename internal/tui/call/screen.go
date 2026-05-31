package call

import (
	"fmt"

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

func callHeader(method protoreflect.MethodDescriptor) string {
	input := string(method.Input().FullName())
	if method.IsStreamingClient() {
		input = "stream " + input
	}

	output := string(method.Output().FullName())
	if method.IsStreamingServer() {
		output = "stream " + output
	}

	header := fmt.Sprintf("%s(%s) -> %s", method.FullName(), input, output)
	return headerStyle.Render(header)
}
