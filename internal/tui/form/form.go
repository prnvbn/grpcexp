package form

import (
	"context"
	"fmt"
	"strings"
	"time"

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
	method     protoreflect.MethodDescriptor
	fields     []Field
	focusIndex int
	state      formState
	width      int
	height     int

	unsupportedFields []string
	client            *grpc.Client

	response    string
	responseErr error
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
	f.buildFields()

	if len(f.fields) > 0 {
		f.focusField(0)
	}

	return f
}

func (f *Form) Init() tea.Cmd {
	if len(f.fields) > 0 && f.fields[0].kind == FieldText {
		return f.fields[0].textInput.Focus()
	}
	return nil
}

func (f *Form) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if f.state == formStateResult {
		return f, nil
	}

	switch msg := msg.(type) {
	case rpcResultMsg:
		f.state = formStateResult
		f.response = msg.response
		f.responseErr = msg.err
		return f, nil
	case tea.KeyMsg:
		if f.state == formStateCalling {
			return f, nil
		}
		if model, cmd, handled := f.handleKey(msg); handled {
			return model, cmd
		}
	}

	return f, f.updateFocusedField(msg)
}

func (f *Form) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	switch msg.String() {
	case "j":
		if f.focusIndex > 0 && f.fields[f.focusIndex].kind != FieldText {
			f.nextField()
			return f, nil, true
		}
	case "tab", "down":
		f.nextField()
		return f, nil, true
	case "k":
		if f.focusIndex > 0 && f.fields[f.focusIndex].kind != FieldText {
			f.prevField()
			return f, nil, true
		}
	case "shift+tab", "up":
		f.prevField()
		return f, nil, true
	case "left", "h":
		if len(f.fields) > 0 {
			field := &f.fields[f.focusIndex]
			if field.kind == FieldEnum || field.kind == FieldBool {
				field.enumPicker.Prev()
				return f, nil, true
			}
		}
	case "right", "l":
		if len(f.fields) > 0 {
			field := &f.fields[f.focusIndex]
			if field.kind == FieldEnum || field.kind == FieldBool {
				field.enumPicker.Next()
				return f, nil, true
			}
		}
	case "enter":
		if f.focusIndex == len(f.fields)-1 || len(f.fields) == 0 {
			f.state = formStateCalling
			return f, f.invokeRPC(), true
		}
		f.nextField()
		return f, nil, true
	}
	return f, nil, false
}

func (f *Form) updateFocusedField(msg tea.Msg) tea.Cmd {
	if len(f.fields) == 0 {
		return nil
	}

	field := &f.fields[f.focusIndex]
	switch field.kind {
	case FieldText:
		var cmd tea.Cmd
		field.textInput, cmd = field.textInput.Update(msg)
		return cmd
	}
	return nil
}

func (f *Form) nextField() {
	if len(f.fields) == 0 {
		return
	}
	f.blurField(f.focusIndex)
	f.focusIndex = (f.focusIndex + 1) % len(f.fields)
	f.focusField(f.focusIndex)
}

func (f *Form) prevField() {
	if len(f.fields) == 0 {
		return
	}
	f.blurField(f.focusIndex)
	f.focusIndex--
	if f.focusIndex < 0 {
		f.focusIndex = len(f.fields) - 1
	}
	f.focusField(f.focusIndex)
}

func (f *Form) focusField(idx int) {
	if idx < 0 || idx >= len(f.fields) {
		return
	}
	field := &f.fields[idx]
	switch field.kind {
	case FieldText:
		field.textInput.Focus()
	}
}

func (f *Form) blurField(idx int) {
	if idx < 0 || idx >= len(f.fields) {
		return
	}
	field := &f.fields[idx]
	switch field.kind {
	case FieldText:
		field.textInput.Blur()
	}
}

func (f *Form) View() string {
	var b strings.Builder

	b.WriteString(headerStyle.Render(fmt.Sprintf("Method: %s", f.method.Name())))
	b.WriteString("\n")
	b.WriteString(labelStyle.Render(fmt.Sprintf("Request: %s", f.method.Input().FullName())))
	b.WriteString("\n")
	b.WriteString(labelStyle.Render(fmt.Sprintf("Response: %s", f.method.Output().FullName())))
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
	case formStateInput:
		b.WriteString(f.renderFields())
	default:
		panic(fmt.Sprintf("unknown state - non exhaustive switch for form state: %d", f.state))
	}

	return b.String()
}

func (f *Form) renderFields() string {
	var b strings.Builder

	if len(f.fields) == 0 {
		b.WriteString(labelStyle.Render("This method has no supported input fields."))
		b.WriteString("\n")
	}

	for i, field := range f.fields {
		isFocused := i == f.focusIndex

		label := field.name
		if isFocused {
			b.WriteString(focusedLabelStyle.Render("> " + label))
		} else {
			b.WriteString(labelStyle.Render("  " + label))
		}
		b.WriteString("\n")

		switch field.kind {
		case FieldText:
			b.WriteString("  ")
			b.WriteString(field.textInput.View())
		case FieldEnum, FieldBool:
			b.WriteString("  ")
			b.WriteString(field.enumPicker.View())
		}
		b.WriteString("\n\n")
	}

	if len(f.unsupportedFields) > 0 {
		b.WriteString(labelStyle.Render(fmt.Sprintf("Unsupported fields: %s",
			strings.Join(f.unsupportedFields, ", "))))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(labelStyle.Render("tab/↓/j: next • shift+tab/↑/k: prev • left/h: prev opt • right/l: next opt • enter: submit"))

	return b.String()
}

func (f *Form) SetSize(width, height int) {
	f.width = width
	f.height = height

	for i := range f.fields {
		if f.fields[i].kind == FieldText {
			f.fields[i].textInput.Width = width - 10
		}
	}
}

func (f *Form) submittedValues() map[string]any {
	mp := make(map[string]any)
	for _, field := range f.fields {
		var val any
		switch field.kind {
		case FieldText:
			val = field.textInput.Value()
		case FieldEnum, FieldBool:
			if item := field.enumPicker.SelectedItem(); item != nil {
				val = item.value
			}
		}
		mp[field.name] = val
	}
	return mp
}

func (f *Form) invokeRPC() tea.Cmd {
	methodFullName := string(f.method.FullName())
	request := f.submittedValues()
	client := f.client

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		response, err := client.InvokeRPC(ctx, methodFullName, request)
		return rpcResultMsg{response: response, err: err}
	}
}

func (f *Form) buildFields() {
	inputMsgDesc := f.method.Input()
	fields := inputMsgDesc.Fields()

	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)

		// todo! add support
		if field.IsList() || field.IsMap() || field.Kind() == protoreflect.MessageKind {
			f.unsupportedFields = append(f.unsupportedFields, string(field.Name()))
			continue
		}

		formField := f.createFormField(field)
		if formField != nil {
			f.fields = append(f.fields, *formField)
		}
	}
}

func (f *Form) createFormField(field protoreflect.FieldDescriptor) *Field {
	name := string(field.Name())

	switch field.Kind() {
	case protoreflect.StringKind:
		return NewTextField(name, fmt.Sprintf("Enter %s...", name), 256, nil)

	case protoreflect.BoolKind:
		return NewBoolField(name)

	case protoreflect.Int32Kind, protoreflect.Int64Kind,
		protoreflect.Sint32Kind, protoreflect.Sint64Kind,
		protoreflect.Sfixed32Kind, protoreflect.Sfixed64Kind:
		return NewTextField(name, "Enter integer...", 64, validateInt)

	case protoreflect.Uint32Kind, protoreflect.Uint64Kind,
		protoreflect.Fixed32Kind, protoreflect.Fixed64Kind:
		return NewTextField(name, "Enter positive integer...", 64, validateUint)

	case protoreflect.FloatKind, protoreflect.DoubleKind:
		return NewTextField(name, "Enter number...", 64, validateFloat)

	case protoreflect.EnumKind:
		return NewEnumField(name, field)

	case protoreflect.BytesKind:
		return NewTextField(name, "Enter hex bytes (e.g., deadbeef)...", 512, nil)

	default:
		return nil
	}
}
