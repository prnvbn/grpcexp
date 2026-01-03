package form

import (
	"encoding/json"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/prnvbn/grpcexp/internal/grpc"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var _ tea.Model = &Form{}

type Form struct {
	method     protoreflect.MethodDescriptor
	fields     []Field
	focusIndex int
	submitted  bool
	width      int
	height     int

	unsupportedFields []string
	client            *grpc.Client
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
	if len(f.fields) > 0 && f.fields[0].Kind == FieldText {
		return f.fields[0].textInput.Focus()
	}
	return nil
}

func (f *Form) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if f.submitted {
		return f, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "down":
			f.nextField()
			return f, nil
		case "shift+tab", "up":
			f.prevField()
			return f, nil
		case "enter":
			// Submit if on last field, otherwise move to next
			if f.focusIndex == len(f.fields)-1 {
				f.submitted = true
				return f, nil
			}
			f.nextField()
			return f, nil
		}
	}

	// Forward message to focused field
	return f, f.updateFocusedField(msg)
}

func (f *Form) updateFocusedField(msg tea.Msg) tea.Cmd {
	if len(f.fields) == 0 {
		return nil
	}

	field := &f.fields[f.focusIndex]
	switch field.Kind {
	case FieldText:
		var cmd tea.Cmd
		field.textInput, cmd = field.textInput.Update(msg)
		return cmd
	case FieldEnum, FieldBool:
		var cmd tea.Cmd
		field.enumList, cmd = field.enumList.Update(msg)
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
	switch field.Kind {
	case FieldText:
		field.textInput.Focus()
	}
}

func (f *Form) blurField(idx int) {
	if idx < 0 || idx >= len(f.fields) {
		return
	}
	field := &f.fields[idx]
	switch field.Kind {
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

	if f.submitted {
		b.WriteString(headerStyle.Render("Form Submitted!"))
		b.WriteString("\n")
		b.WriteString(f.renderSubmittedValues())
	} else {
		b.WriteString(f.renderFields())
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

		switch field.Kind {
		case FieldText:
			b.WriteString("  ")
			b.WriteString(field.textInput.View())
		case FieldEnum, FieldBool:
			// Show selected value inline instead of full list
			item := field.enumList.SelectedItem()
			enumItem, ok := item.(EnumItem)
			if !ok {
				continue
			}
			b.WriteString("  ")
			b.WriteString(fmt.Sprintf("< %s >", enumItem.Name))
		}
		b.WriteString("\n\n")
	}

	if len(f.unsupportedFields) > 0 {
		b.WriteString(labelStyle.Render(fmt.Sprintf("Unsupported fields: %s",
			strings.Join(f.unsupportedFields, ", "))))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(labelStyle.Render("tab: next • shift+tab: prev • enter: submit"))

	return b.String()
}

func (f *Form) SetSize(width, height int) {
	f.width = width
	f.height = height

	for i := range f.fields {
		if f.fields[i].Kind == FieldText {
			f.fields[i].textInput.Width = width - 10
		}
	}
}

func (f *Form) renderSubmittedValues() string {
	var b strings.Builder
	b.WriteString("Values:\n")

	mp := make(map[string]string)
	for _, field := range f.fields {
		var valStr string
		switch field.Kind {
		case FieldText:
			valStr = field.textInput.Value()
		case FieldEnum, FieldBool:
			if item := field.enumList.SelectedItem(); item != nil {
				if enumItem, ok := item.(EnumItem); ok {
					valStr = enumItem.Value
				}
			}
		}

		mp[field.name] = valStr

	}

	enc := json.NewEncoder(&b)
	enc.SetIndent("", "  ")
	err := enc.Encode(mp)
	if err != nil {
		return fmt.Sprintf("Error encoding submitted values: %v", err)
	}

	return b.String()
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
		return NewTextField(name, "Enter integer...", 64, ValidateInt)

	case protoreflect.Uint32Kind, protoreflect.Uint64Kind,
		protoreflect.Fixed32Kind, protoreflect.Fixed64Kind:
		return NewTextField(name, "Enter positive integer...", 64, ValidateUint)

	case protoreflect.FloatKind, protoreflect.DoubleKind:
		return NewTextField(name, "Enter number...", 64, ValidateFloat)

	case protoreflect.EnumKind:
		return NewEnumField(name, field)

	case protoreflect.BytesKind:
		return NewTextField(name, "Enter hex bytes (e.g., deadbeef)...", 512, nil)

	default:
		return nil
	}
}
