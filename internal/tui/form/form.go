package form

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/prnvbn/grpcexp/internal/grpc"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var _ tea.Model = &Form{}

type formState int

const (
	formStateInput formState = iota
	formStateCalling
	formStateResult
)

type Form struct {
	method protoreflect.MethodDescriptor
	root   *fieldGroup
	state  formState
	width  int
	height int

	unsupportedFields []string
	client            *grpc.Client

	response      string
	responseErr   error
	submitFocused bool
}

type rpcResultMsg struct {
	response string
	err      error
}

func NewForm(method protoreflect.MethodDescriptor, client *grpc.Client) Form {
	f := Form{
		method: method,
		client: client,
	}

	inputMsgDesc := method.Input()
	f.root = f.buildFieldGroup(inputMsgDesc.Fields())

	if f.root.Empty() {
		f.submitFocused = true
	} else {
		f.root.FocusFirst()
	}

	return f
}

func (f *Form) Init() tea.Cmd {
	return nil
}

func (f *Form) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case rpcResultMsg:
		f.state = formStateResult
		f.response = msg.response
		f.responseErr = msg.err
		return f, nil
	case tea.KeyMsg:
		switch f.state {
		case formStateResult:
			switch msg.String() {
			case "y":
				content := f.response
				if f.responseErr != nil {
					content = f.responseErr.Error()
				}
				clipboard.WriteAll(content)
			case "q":
				return f, tea.Quit
			}
			return f, nil
		case formStateCalling:
			return f, nil
		case formStateInput:
			model, cmd, handled := f.handleKey(msg)
			if handled {
				return model, cmd
			}
		default:
			panic(fmt.Sprintf("unknown state - non exhaustive switch for key msg: %d", f.state))
		}
	}

	return f, f.root.Update(msg)
}

func (f *Form) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	switch msg.String() {
	case "j", "k":
		if f.submitFocused || !f.root.AcceptsTextInput() {
			if msg.String() == "j" {
				f.nextField()
			} else {
				f.prevField()
			}
			return f, nil, true
		}
	case "tab", "down":
		f.nextField()
		return f, nil, true
	case "enter":
		if f.submitFocused {
			f.state = formStateCalling
			return f, f.invokeRPC(), true
		}
		f.nextField()
		return f, nil, true
	case "shift+tab", "up":
		f.prevField()
		return f, nil, true
	case "left", "h", "right", "l":
		cmd, handled := f.root.HandleKey(msg)
		if handled {
			return f, cmd, true
		}
	}
	return f, nil, false
}

func (f *Form) nextField() {
	if f.submitFocused {
		return
	}
	if !f.root.NextField() {
		f.root.Blur()
		f.submitFocused = true
	}
}

func (f *Form) prevField() {
	if f.submitFocused {
		f.submitFocused = false
		f.root.FocusLast()
		return
	}
	f.root.PrevField()
}

func (f *Form) View() string {
	var b strings.Builder

	header := fmt.Sprintf("%s(%s) -> %s", f.method.FullName(), f.method.Input().FullName(), f.method.Output().FullName())
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n\n")

	switch f.state {
	case formStateCalling:
		b.WriteString(labelStyle.Render("Calling..."))
		b.WriteString("\n")
	case formStateResult:
		if f.responseErr != nil {
			b.WriteString(headerStyle.Render("Error"))
			b.WriteString("\n\n")
			b.WriteString(labelStyle.Render(f.responseErr.Error()))
		} else {
			b.WriteString(headerStyle.Render("Response"))
			b.WriteString("\n\n")
			b.WriteString(f.response)
		}
		b.WriteString("\n\n")
		b.WriteString(labelStyle.Render("esc: back • y: copy response • q: quit"))
	case formStateInput:
		b.WriteString(f.renderFields())
	default:
		panic(fmt.Sprintf("unknown state: %d", f.state))
	}

	return b.String()
}

func (f *Form) renderFields() string {
	var b strings.Builder

	if f.root.Empty() {
		b.WriteString(labelStyle.Render("No input fields."))
		b.WriteString("\n")
	} else {
		b.WriteString(f.root.ViewWithDepth(0))
	}

	if len(f.unsupportedFields) > 0 {
		b.WriteString(labelStyle.Render(fmt.Sprintf("(unsupported: %s)",
			strings.Join(f.unsupportedFields, ", "))))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	if f.submitFocused {
		b.WriteString(focusedLabelStyle.Render("> [Submit]"))
	} else {
		b.WriteString(labelStyle.Render("  [Submit]"))
	}
	b.WriteString("\n\n")
	b.WriteString(labelStyle.Render("↑/↓/enter: navigate • ←/→: options"))

	return b.String()
}

func (f *Form) SetSize(width, height int) {
	f.width = width
	f.height = height
	f.root.SetWidth(width - 10)
}

func (f *Form) invokeRPC() tea.Cmd {
	methodFullName := string(f.method.FullName())
	request := f.root.Value()
	client := f.client

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		response, err := client.InvokeRPC(ctx, methodFullName, request)
		return rpcResultMsg{response: response, err: err}
	}
}

func (f *Form) buildFieldGroup(fields protoreflect.FieldDescriptors) *fieldGroup {
	g := &fieldGroup{
		name:       "",
		fields:     make([]Field, 0),
		focusIndex: 0,
		focused:    false,
	}

	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		fieldName := string(field.Name())

		if field.IsList() || field.IsMap() {
			f.unsupportedFields = append(f.unsupportedFields, fieldName)
			continue
		}

		formField := NewFieldFromProto(field)
		if formField != nil {
			g.fields = append(g.fields, *formField)
		}
	}

	return g
}
