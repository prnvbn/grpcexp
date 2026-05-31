package call

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/prnvbn/grpcexp/internal/grpc"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var _ Screen = &Unary{}

type unaryState int

const (
	unaryStateInput unaryState = iota
	unaryStateCalling
	unaryStateResult
)

type Unary struct {
	method  protoreflect.MethodDescriptor
	builder *Builder
	client  *grpc.Client
	state   unaryState

	response    string
	responseErr error
}

type rpcResultMsg struct {
	response string
	err      error
}

func NewUnary(method protoreflect.MethodDescriptor, client *grpc.Client) *Unary {
	return &Unary{
		method:  method,
		builder: NewBuilder(method.Input()),
		client:  client,
	}
}

func (f *Unary) Init() tea.Cmd {
	return nil
}

func (f *Unary) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case rpcResultMsg:
		f.state = unaryStateResult
		f.response = msg.response
		f.responseErr = msg.err
		return f, nil
	case tea.KeyMsg:
		switch f.state {
		case unaryStateResult:
			return f, f.handleResultKey(msg)
		case unaryStateCalling:
			return f, nil
		case unaryStateInput:
			if msg.String() == "ctrl+y" {
				f.copyGRPCURLCommand()
				return f, nil
			}
			cmd, handled := f.builder.HandleKey(msg, func() tea.Cmd {
				f.state = unaryStateCalling
				return f.invokeRPC()
			})
			if handled {
				return f, cmd
			}
		default:
			panic(fmt.Sprintf("unknown unary state: %d", f.state))
		}
	}

	if f.state == unaryStateInput {
		return f, f.builder.Update(msg)
	}
	return f, nil
}

func (f *Unary) View() string {
	var out strings.Builder

	out.WriteString(callHeader(f.method))
	out.WriteString("\n\n")

	switch f.state {
	case unaryStateCalling:
		out.WriteString(labelStyle.Render("Calling..."))
		out.WriteString("\n")
	case unaryStateResult:
		if f.responseErr != nil {
			out.WriteString(headerStyle.Render("Error"))
			out.WriteString("\n\n")
			out.WriteString(labelStyle.Render(f.responseErr.Error()))
		} else {
			out.WriteString(headerStyle.Render("Response"))
			out.WriteString("\n\n")
			out.WriteString(f.response)
		}
		out.WriteString("\n\n")
		out.WriteString(labelStyle.Render("esc: back • r: resubmit • y: copy response • ctrl+y: copy grpcurl • q: quit"))
	case unaryStateInput:
		out.WriteString(f.builder.View("Submit", true, false))
		out.WriteString("\n\n")
		out.WriteString(labelStyle.Render("up/down/tab: navigate • left/right: options • ctrl+y: copy grpcurl"))
	default:
		panic(fmt.Sprintf("unknown unary state: %d", f.state))
	}

	return out.String()
}

func (f *Unary) SetSize(width, _ int) {
	f.builder.SetWidth(width - 10)
}

func (f *Unary) AcceptsTextInput() bool {
	return f.state == unaryStateInput && f.builder.AcceptsTextInput()
}

func (f *Unary) Cancel() {}

func (f *Unary) handleResultKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "r":
		f.state = unaryStateInput
		f.builder.ResetToSubmit()
	case "y":
		content := f.response
		if f.responseErr != nil {
			content = f.responseErr.Error()
		}
		if err := clipboard.WriteAll(content); err != nil {
			fmt.Fprintf(os.Stderr, "error writing to clipboard: %v\n", err)
		}
	case "ctrl+y":
		f.copyGRPCURLCommand()
	case "q":
		return tea.Quit
	}
	return nil
}

func (f *Unary) invokeRPC() tea.Cmd {
	methodFullName := string(f.method.FullName())
	request := f.builder.Value()
	client := f.client

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		response, err := client.InvokeRPC(ctx, methodFullName, request)
		return rpcResultMsg{response: response, err: err}
	}
}

func (f *Unary) copyGRPCURLCommand() {
	command, err := f.client.GRPCURLCommand(string(f.method.FullName()), f.builder.Value())
	if err != nil {
		fmt.Fprintf(os.Stderr, "error building grpcurl command: %v\n", err)
		return
	}
	if err := clipboard.WriteAll(command); err != nil {
		fmt.Fprintf(os.Stderr, "error writing to clipboard: %v\n", err)
	}
}
