package form

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type fieldKind int

const (
	FieldText fieldKind = iota
	FieldBool
	FieldEnum
	FieldGroup
	FieldList
	FieldMap
	FieldOneof
)

type Field struct {
	name string
	kind fieldKind

	textInput  textinput.Model
	enumPicker enumPicker
	fieldGroup *fieldGroup
	listField  *fieldList
	mapField   *fieldMap
	oneofField *fieldOneof

	validate func(string) error
}

func NewTextField(name string, placeholder string, charLimit int, validate func(string) error) *Field {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.CharLimit = charLimit
	ti.Prompt = ""
	return &Field{
		name:      name,
		kind:      FieldText,
		textInput: ti,
		validate:  validate,
	}
}

func NewBoolField(name string) *Field {
	items := []enumItem{
		{name: "false", value: "false"},
		{name: "true", value: "true"},
	}

	return &Field{
		name:       name,
		kind:       FieldBool,
		enumPicker: newEnumPicker(items),
	}
}

func NewEnumField(name string, field protoreflect.FieldDescriptor) *Field {
	enumDesc := field.Enum()
	values := enumDesc.Values()

	items := make([]enumItem, values.Len())
	for i := 0; i < values.Len(); i++ {
		enumVal := values.Get(i)
		items[i] = enumItem{
			name:  string(enumVal.Name()),
			value: fmt.Sprintf("%d", enumVal.Number()),
		}
	}

	return &Field{
		name:       name,
		kind:       FieldEnum,
		enumPicker: newEnumPicker(items),
	}
}

func NewFieldGroup(name string, field protoreflect.FieldDescriptor) *Field {
	fields := field.Message().Fields()
	fg := NewfieldGroup(name, fields)
	return &Field{
		name:       name,
		kind:       FieldGroup,
		fieldGroup: fg,
	}
}

func NewListField(name string, field protoreflect.FieldDescriptor) *Field {
	lf := newListField(name, field)
	return &Field{
		name:      name,
		kind:      FieldList,
		listField: lf,
	}
}

func NewMapField(name string, field protoreflect.FieldDescriptor) *Field {
	mf := newMapField(name, field)
	return &Field{
		name:     name,
		kind:     FieldMap,
		mapField: mf,
	}
}

func NewOneofField(name string, oneof protoreflect.OneofDescriptor) *Field {
	of := newFieldOneof(name, oneof)
	return &Field{
		name:       name,
		kind:       FieldOneof,
		oneofField: of,
	}
}

func validateInt(s string) error {
	if s == "" {
		return nil
	}
	_, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return fmt.Errorf("must be a valid integer")
	}
	return nil
}

func validateUint(s string) error {
	if s == "" {
		return nil
	}
	_, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return fmt.Errorf("must be a valid positive integer")
	}
	return nil
}

func validateFloat(s string) error {
	if s == "" {
		return nil
	}
	_, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fmt.Errorf("must be a valid number")
	}
	return nil
}

func validateDuration(s string) error {
	if s == "" {
		return nil
	}
	_, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("must be a valid duration (e.g., 10s)")
	}
	return nil
}

func validateTimestamp(s string) error {
	if s == "" {
		return nil
	}
	_, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		// Try RFC3339 format without nanoseconds
		_, err = time.Parse(time.RFC3339, s)
		if err != nil {
			return fmt.Errorf("must be a valid RFC 3339 timestamp (e.g., 2017-01-15T01:30:15.01Z)")
		}
	}
	return nil
}

func NewFieldFromProto(field protoreflect.FieldDescriptor) *Field {
	name := string(field.Name())
	kind := field.Kind()

	switch kind {
	case protoreflect.StringKind:
		return NewTextField(name, fmt.Sprintf("Enter %s...", name), 0, nil)

	case protoreflect.BoolKind:
		return NewBoolField(name)

	case protoreflect.Int32Kind, protoreflect.Int64Kind,
		protoreflect.Sint32Kind, protoreflect.Sint64Kind,
		protoreflect.Sfixed32Kind, protoreflect.Sfixed64Kind:
		return NewTextField(name, fmt.Sprintf("Enter %s...", kind.String()), 64, validateInt)

	case protoreflect.Uint32Kind, protoreflect.Uint64Kind,
		protoreflect.Fixed32Kind, protoreflect.Fixed64Kind:
		return NewTextField(name, fmt.Sprintf("Enter %s...", kind.String()), 64, validateUint)

	case protoreflect.FloatKind, protoreflect.DoubleKind:
		return NewTextField(name, fmt.Sprintf("Enter %s...", kind.String()), 64, validateFloat)

	case protoreflect.EnumKind:
		return NewEnumField(name, field)

	case protoreflect.BytesKind:
		return NewTextField(name, "Enter hex bytes (e.g., deadbeef)...", 512, nil)
	case protoreflect.MessageKind:
		msgDesc := field.Message()

		// support for well-known types
		switch msgDesc.FullName() {
		case "google.protobuf.Timestamp":
			return NewTextField(name, "Enter RFC 3339 timestamp (e.g., 2017-01-15T01:30:15.01Z)...", 64, validateTimestamp)
		case "google.protobuf.Duration":
			return NewTextField(name, "Enter duration (e.g., 10s)...", 64, validateDuration)
		default:
			return NewFieldGroup(name, field)
		}
	default:
		return nil
	}
}

func (f *Field) Value() any {
	switch f.kind {
	case FieldText:
		return f.textInput.Value()
	case FieldEnum, FieldBool:
		return f.enumPicker.Value()
	case FieldGroup:
		return f.fieldGroup.Value()
	case FieldList:
		return f.listField.Value()
	case FieldMap:
		return f.mapField.Value()
	case FieldOneof:
		return f.oneofField.Value()
	default:
		panic(fmt.Sprintf("unknown field kind: %d", f.kind))
	}
}

func (f *Field) View() string {
	switch f.kind {
	case FieldText:
		return f.textInput.View()
	case FieldEnum, FieldBool:
		return f.enumPicker.View()
	case FieldGroup:
		return f.fieldGroup.View()
	case FieldList:
		return f.listField.View()
	case FieldMap:
		return f.mapField.View()
	case FieldOneof:
		return f.oneofField.View()
	default:
		panic(fmt.Sprintf("unknown field kind: %d", f.kind))
	}
}

func (f *Field) Name() string {
	return f.name
}

func (f *Field) AcceptsTextInput() bool {
	switch f.kind {
	case FieldText:
		return true
	case FieldList:
		if f.listField != nil {
			return f.listField.AcceptsTextInput()
		}
	case FieldMap:
		if f.mapField != nil {
			return f.mapField.AcceptsTextInput()
		}
	case FieldOneof:
		if f.oneofField != nil {
			return f.oneofField.AcceptsTextInput()
		}
	}
	return false
}

func (f *Field) Focus() tea.Cmd {
	switch f.kind {
	case FieldText:
		return f.textInput.Focus()
	case FieldGroup:
		if f.fieldGroup != nil {
			f.fieldGroup.FocusFirst()
		}
	case FieldList:
		if f.listField != nil {
			return f.listField.FocusFirst()
		}
	case FieldMap:
		if f.mapField != nil {
			return f.mapField.FocusFirst()
		}
	case FieldOneof:
		if f.oneofField != nil {
			return f.oneofField.FocusFirst()
		}
	}
	return nil
}

func (f *Field) FocusFromEnd() tea.Cmd {
	switch f.kind {
	case FieldText:
		return f.textInput.Focus()
	case FieldGroup:
		if f.fieldGroup != nil {
			f.fieldGroup.FocusLast()
		}
	case FieldList:
		if f.listField != nil {
			return f.listField.FocusLast()
		}
	case FieldMap:
		if f.mapField != nil {
			return f.mapField.FocusLast()
		}
	case FieldOneof:
		if f.oneofField != nil {
			return f.oneofField.FocusLast()
		}
	}
	return nil
}

func (f *Field) Blur() {
	switch f.kind {
	case FieldText:
		f.textInput.Blur()
	case FieldGroup:
		if f.fieldGroup != nil {
			f.fieldGroup.Blur()
		}
	case FieldList:
		if f.listField != nil {
			f.listField.Blur()
		}
	case FieldMap:
		if f.mapField != nil {
			f.mapField.Blur()
		}
	case FieldOneof:
		if f.oneofField != nil {
			f.oneofField.Blur()
		}
	}
}

func (f *Field) Next() bool {
	switch f.kind {
	case FieldGroup:
		if f.fieldGroup != nil {
			return f.fieldGroup.NextField()
		}
	case FieldList:
		if f.listField != nil {
			return f.listField.NextField()
		}
	case FieldMap:
		if f.mapField != nil {
			return f.mapField.NextField()
		}
	case FieldOneof:
		if f.oneofField != nil {
			return f.oneofField.NextField()
		}
	}
	return false
}

func (f *Field) Prev() bool {
	switch f.kind {
	case FieldGroup:
		if f.fieldGroup != nil {
			return f.fieldGroup.PrevField()
		}
	case FieldList:
		if f.listField != nil {
			return f.listField.PrevField()
		}
	case FieldMap:
		if f.mapField != nil {
			return f.mapField.PrevField()
		}
	case FieldOneof:
		if f.oneofField != nil {
			return f.oneofField.PrevField()
		}
	}
	return false
}

func (f *Field) HandleKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	switch f.kind {
	case FieldEnum, FieldBool:
		switch msg.String() {
		case "left", "right":
			f.enumPicker.Update(msg)
			return nil, true
		}
	case FieldGroup:
		if f.fieldGroup != nil {
			return f.fieldGroup.HandleKey(msg)
		}
	case FieldList:
		if f.listField != nil {
			return f.listField.HandleKey(msg)
		}
	case FieldMap:
		if f.mapField != nil {
			return f.mapField.HandleKey(msg)
		}
	case FieldOneof:
		if f.oneofField != nil {
			return f.oneofField.HandleKey(msg)
		}
	}
	return nil, false
}

func (f *Field) Update(msg tea.Msg) tea.Cmd {
	switch f.kind {
	case FieldText:
		var cmd tea.Cmd
		f.textInput, cmd = f.textInput.Update(msg)
		return cmd
	case FieldGroup:
		if f.fieldGroup != nil {
			return f.fieldGroup.Update(msg)
		}
	case FieldList:
		if f.listField != nil {
			return f.listField.Update(msg)
		}
	case FieldMap:
		if f.mapField != nil {
			return f.mapField.Update(msg)
		}
	case FieldOneof:
		if f.oneofField != nil {
			return f.oneofField.Update(msg)
		}
	}
	return nil
}

func (f *Field) SetWidth(width int) {
	switch f.kind {
	case FieldText:
		f.textInput.Width = width
	case FieldGroup:
		f.fieldGroup.SetWidth(width)
	case FieldList:
		if f.listField != nil {
			f.listField.SetWidth(width)
		}
	case FieldMap:
		if f.mapField != nil {
			f.mapField.SetWidth(width)
		}
	case FieldOneof:
		if f.oneofField != nil {
			f.oneofField.SetWidth(width)
		}
	}
}

func (f *Field) RenderWithFocus(focused bool, depth int) string {
	var b strings.Builder
	indent := strings.Repeat("  ", depth)

	var prefix string
	if focused {
		prefix = indent + "> "
	} else {
		prefix = indent + "  "
	}

	switch f.kind {
	case FieldGroup:
		if focused {
			b.WriteString(focusedLabelStyle.Render(prefix + f.name + ":"))
		} else {
			b.WriteString(labelStyle.Render(prefix + f.name + ":"))
		}
		b.WriteString("\n")
		b.WriteString(f.fieldGroup.ViewWithDepth(depth + 1))
	case FieldList:
		if focused {
			b.WriteString(focusedLabelStyle.Render(prefix + f.name + ":"))
		} else {
			b.WriteString(labelStyle.Render(prefix + f.name + ":"))
		}
		b.WriteString("\n")
		b.WriteString(f.listField.ViewWithDepth(depth + 1))
	case FieldMap:
		if focused {
			b.WriteString(focusedLabelStyle.Render(prefix + f.name + ":"))
		} else {
			b.WriteString(labelStyle.Render(prefix + f.name + ":"))
		}
		b.WriteString("\n")
		b.WriteString(f.mapField.ViewWithDepth(depth + 1))
	case FieldOneof:
		if focused {
			b.WriteString(focusedLabelStyle.Render(prefix + f.name + ":"))
		} else {
			b.WriteString(labelStyle.Render(prefix + f.name + ":"))
		}
		b.WriteString("\n")
		b.WriteString(f.oneofField.ViewWithDepth(depth + 1))
	default:
		if focused {
			b.WriteString(focusedLabelStyle.Render(prefix + f.name + ": "))
		} else {
			b.WriteString(labelStyle.Render(prefix + f.name + ": "))
		}
		b.WriteString(f.View())
		b.WriteString("\n")
	}

	return b.String()
}
