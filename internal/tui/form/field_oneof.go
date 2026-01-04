package form

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type oneofFocusState int

const (
	oneofFocusPicker oneofFocusState = iota
	oneofFocusField
)

type fieldOneof struct {
	name          string
	picker        enumPicker
	fields        []Field
	selectedIndex int
	focusState    oneofFocusState
	focused       bool
}

func newFieldOneof(name string, oneof protoreflect.OneofDescriptor) *fieldOneof {
	protoFields := oneof.Fields()

	items := make([]enumItem, protoFields.Len())
	fields := make([]Field, protoFields.Len())

	for i := 0; i < protoFields.Len(); i++ {
		protoField := protoFields.Get(i)
		fieldName := string(protoField.Name())

		items[i] = enumItem{
			name:  fieldName,
			value: fieldName,
		}

		field := NewFieldFromProto(protoField)
		if field != nil {
			fields[i] = *field
		}
	}

	return &fieldOneof{
		name:          name,
		picker:        newEnumPicker(items),
		fields:        fields,
		selectedIndex: 0,
		focusState:    oneofFocusPicker,
		focused:       false,
	}
}

func (o *fieldOneof) selectedField() *Field {
	if o.selectedIndex < 0 || o.selectedIndex >= len(o.fields) {
		return nil
	}
	return &o.fields[o.selectedIndex]
}

func (o *fieldOneof) Value() map[string]any {
	result := make(map[string]any)
	field := o.selectedField()
	if field != nil {
		result[field.name] = field.Value()
	}
	return result
}

func (o *fieldOneof) FocusFirst() tea.Cmd {
	o.focused = true
	o.focusState = oneofFocusPicker
	return nil
}

func (o *fieldOneof) FocusLast() tea.Cmd {
	o.focused = true
	o.focusState = oneofFocusField
	field := o.selectedField()
	if field != nil {
		return field.FocusFromEnd()
	}
	return nil
}

func (o *fieldOneof) Blur() {
	o.focused = false
	if o.focusState == oneofFocusField {
		field := o.selectedField()
		if field != nil {
			field.Blur()
		}
	}
}

func (o *fieldOneof) NextField() bool {
	if !o.focused {
		return false
	}

	switch o.focusState {
	case oneofFocusPicker:
		o.focusState = oneofFocusField
		field := o.selectedField()
		if field != nil {
			field.Focus()
		}
		return true
	case oneofFocusField:
		field := o.selectedField()
		if field != nil {
			if field.Next() {
				return true
			}
			field.Blur()
		}
		return false
	}
	return false
}

func (o *fieldOneof) PrevField() bool {
	if !o.focused {
		return false
	}

	switch o.focusState {
	case oneofFocusField:
		field := o.selectedField()
		if field != nil {
			if field.Prev() {
				return true
			}
			field.Blur()
		}
		o.focusState = oneofFocusPicker
		return true
	case oneofFocusPicker:
		return false
	}
	return false
}

func (o *fieldOneof) AcceptsTextInput() bool {
	if o.focusState == oneofFocusField {
		field := o.selectedField()
		if field != nil {
			return field.AcceptsTextInput()
		}
	}
	return false
}

func (o *fieldOneof) HandleKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	if !o.focused {
		return nil, false
	}

	switch o.focusState {
	case oneofFocusPicker:
		switch msg.String() {
		case "left", "right":
			oldIndex := o.picker.selected
			o.picker.Update(msg)
			newIndex := o.picker.selected
			if oldIndex != newIndex {
				o.selectedIndex = newIndex
			}
			return nil, true
		}
	case oneofFocusField:
		field := o.selectedField()
		if field != nil {
			return field.HandleKey(msg)
		}
	}
	return nil, false
}

func (o *fieldOneof) Update(msg tea.Msg) tea.Cmd {
	if !o.focused {
		return nil
	}

	if o.focusState == oneofFocusField {
		field := o.selectedField()
		if field != nil {
			return field.Update(msg)
		}
	}
	return nil
}

func (o *fieldOneof) SetWidth(width int) {
	for i := range o.fields {
		o.fields[i].SetWidth(width)
	}
}

func (o *fieldOneof) View() string {
	return o.ViewWithDepth(1)
}

func (o *fieldOneof) ViewWithDepth(depth int) string {
	var b strings.Builder
	indent := strings.Repeat("  ", depth)

	field := o.selectedField()
	if field == nil {
		return ""
	}

	isFocused := o.focused && o.focusState == oneofFocusField

	var prefix string
	if isFocused {
		prefix = indent + "> "
	} else {
		prefix = indent + "  "
	}

	switch field.kind {
	case FieldGroup:
		if isFocused {
			b.WriteString(focusedLabelStyle.Render(prefix + field.name + ":"))
		} else {
			b.WriteString(labelStyle.Render(prefix + field.name + ":"))
		}
		b.WriteString("\n")
		b.WriteString(field.fieldGroup.ViewWithDepth(depth + 1))
	default:
		if isFocused {
			b.WriteString(focusedLabelStyle.Render(prefix + field.name + ": "))
		} else {
			b.WriteString(labelStyle.Render(prefix + field.name + ": "))
		}
		b.WriteString(field.View())
		b.WriteString("\n")
	}

	return b.String()
}
