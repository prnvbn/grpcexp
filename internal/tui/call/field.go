package call

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
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
	name                   string
	kind                   fieldKind
	optional               bool
	present                bool
	optionalControlFocused bool
	width                  int

	textInput  textinput.Model
	enumPicker enumPicker
	fieldGroup *fieldGroup
	listField  *fieldList
	mapField   *fieldMap
	oneofField *fieldOneof

	validate func(string) error
}

const optionalControlWidth = 6

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
	f := newFieldFromProto(field, "")
	if f != nil {
		f.optional = field.HasOptionalKeyword()
		if f.optional && field.Kind() == protoreflect.StringKind {
			f.textInput.Placeholder = "Enter string..."
		}
	}
	return f
}

func newFieldFromProto(field protoreflect.FieldDescriptor, inputRole string) *Field {
	name := string(field.Name())
	kind := field.Kind()

	switch kind {
	case protoreflect.StringKind:
		return NewTextField(name, textFieldPlaceholder(field, inputRole), 0, nil)

	case protoreflect.BoolKind:
		return NewBoolField(name)

	case protoreflect.Int32Kind, protoreflect.Int64Kind,
		protoreflect.Sint32Kind, protoreflect.Sint64Kind,
		protoreflect.Sfixed32Kind, protoreflect.Sfixed64Kind:
		return NewTextField(name, textFieldPlaceholder(field, inputRole), 64, validateInt)

	case protoreflect.Uint32Kind, protoreflect.Uint64Kind,
		protoreflect.Fixed32Kind, protoreflect.Fixed64Kind:
		return NewTextField(name, textFieldPlaceholder(field, inputRole), 64, validateUint)

	case protoreflect.FloatKind, protoreflect.DoubleKind:
		return NewTextField(name, textFieldPlaceholder(field, inputRole), 64, validateFloat)

	case protoreflect.EnumKind:
		return NewEnumField(name, field)

	case protoreflect.BytesKind:
		return NewTextField(name, textFieldPlaceholder(field, inputRole), 512, nil)
	case protoreflect.MessageKind:
		msgDesc := field.Message()

		// support for well-known types
		switch msgDesc.FullName() {
		case "google.protobuf.Timestamp":
			return NewTextField(name, textFieldPlaceholder(field, inputRole), 64, validateTimestamp)
		case "google.protobuf.Duration":
			return NewTextField(name, textFieldPlaceholder(field, inputRole), 64, validateDuration)
		default:
			return NewFieldGroup(name, field)
		}
	default:
		return nil
	}
}

func textFieldPlaceholder(field protoreflect.FieldDescriptor, inputRole string) string {
	switch field.Kind() {
	case protoreflect.StringKind:
		if inputRole != "" {
			return fmt.Sprintf("Enter string %s...", inputRole)
		}
		return fmt.Sprintf("Enter %s...", field.Name())
	case protoreflect.Int32Kind, protoreflect.Int64Kind,
		protoreflect.Sint32Kind, protoreflect.Sint64Kind,
		protoreflect.Sfixed32Kind, protoreflect.Sfixed64Kind,
		protoreflect.Uint32Kind, protoreflect.Uint64Kind,
		protoreflect.Fixed32Kind, protoreflect.Fixed64Kind,
		protoreflect.FloatKind, protoreflect.DoubleKind:
		if inputRole != "" {
			return fmt.Sprintf("Enter %s %s...", field.Kind(), inputRole)
		}
		return fmt.Sprintf("Enter %s...", field.Kind())
	case protoreflect.BytesKind:
		if inputRole != "" {
			return fmt.Sprintf("Enter hex bytes %s (e.g., deadbeef)...", inputRole)
		}
		return "Enter hex bytes (e.g., deadbeef)..."
	case protoreflect.MessageKind:
		switch field.Message().FullName() {
		case "google.protobuf.Timestamp":
			if inputRole != "" {
				return fmt.Sprintf("Enter timestamp %s (RFC 3339, e.g., 2017-01-15T01:30:15.01Z)...", inputRole)
			}
			return "Enter RFC 3339 timestamp (e.g., 2017-01-15T01:30:15.01Z)..."
		case "google.protobuf.Duration":
			if inputRole != "" {
				return fmt.Sprintf("Enter duration %s (e.g., 10s)...", inputRole)
			}
			return "Enter duration (e.g., 10s)..."
		}
	}
	return fmt.Sprintf("Enter %s...", field.Name())
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

func (f *Field) checkbox(focused bool) string {
	if !f.optional {
		return ""
	}
	mark := unselectedStyle.Render(" ")
	if f.present {
		mark = selectedStyle.Render("x")
	}
	box := bracketStyle.Render("[") + mark + bracketStyle.Render("]")
	if focused {
		return " " + focusedLabelStyle.Render("> ") + box
	}
	return "   " + box
}

func (f *Field) inputView(prefix string) string {
	if f.optional && f.kind == FieldText && f.width > 0 {
		f.textInput.SetWidth(max(1, f.width-len(prefix)-len(f.name)-2-optionalControlWidth))
	}
	return f.View()
}

func (f *Field) AcceptsTextInput() bool {
	if f.optionalControlFocused {
		return false
	}
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
	f.optionalControlFocused = false
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
	if f.optional {
		f.Blur()
		f.optionalControlFocused = true
		return nil
	}
	return f.focusValueFromEnd()
}

func (f *Field) focusValueFromEnd() tea.Cmd {
	f.optionalControlFocused = false
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
	f.optionalControlFocused = false
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

func (f *Field) HandleKey(msg tea.KeyPressMsg) (tea.Cmd, bool) {
	if f.optionalControlFocused && msg.String() == "enter" {
		f.present = !f.present
		return nil, true
	}

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
	if f.optionalControlFocused {
		return nil
	}
	switch f.kind {
	case FieldText:
		var cmd tea.Cmd
		f.textInput, cmd = f.textInput.Update(msg)
		if f.optional && f.textInput.Value() != "" {
			f.present = true
		}
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
	f.width = width
	switch f.kind {
	case FieldText:
		f.textInput.SetWidth(width)
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
	inputFocused := focused && !f.optionalControlFocused
	if inputFocused {
		prefix = indent + "> "
	} else {
		prefix = indent + "  "
	}

	switch f.kind {
	case FieldGroup:
		if inputFocused {
			b.WriteString(focusedLabelStyle.Render(prefix + f.name + ":"))
		} else {
			b.WriteString(labelStyle.Render(prefix + f.name + ":"))
		}
		b.WriteString("\n")
		b.WriteString(f.fieldGroup.ViewWithDepth(depth + 1))
	case FieldList:
		if inputFocused {
			b.WriteString(focusedLabelStyle.Render(prefix + f.name + ":"))
		} else {
			b.WriteString(labelStyle.Render(prefix + f.name + ":"))
		}
		b.WriteString("\n")
		b.WriteString(f.listField.ViewWithDepth(depth + 1))
	case FieldMap:
		if inputFocused {
			b.WriteString(focusedLabelStyle.Render(prefix + f.name + ":"))
		} else {
			b.WriteString(labelStyle.Render(prefix + f.name + ":"))
		}
		b.WriteString("\n")
		b.WriteString(f.mapField.ViewWithDepth(depth + 1))
	case FieldOneof:
		if inputFocused {
			b.WriteString(focusedLabelStyle.Render(prefix + f.name + ":"))
		} else {
			b.WriteString(labelStyle.Render(prefix + f.name + ":"))
		}
		b.WriteString("\n")
		b.WriteString(f.oneofField.ViewWithDepth(depth + 1))
	default:
		if inputFocused {
			b.WriteString(focusedLabelStyle.Render(prefix + f.name + ": "))
		} else {
			b.WriteString(labelStyle.Render(prefix + f.name + ": "))
		}
		b.WriteString(f.inputView(prefix))
		b.WriteString(f.checkbox(focused && f.optionalControlFocused))
		b.WriteString("\n")
	}

	return b.String()
}
