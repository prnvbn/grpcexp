package form

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type fieldGroup struct {
	name       string
	fields     []Field
	focusIndex int
	focused    bool
}

func NewfieldGroup(name string, protoFields protoreflect.FieldDescriptors) *fieldGroup {
	fields := make([]Field, 0)
	for i := 0; i < protoFields.Len(); i++ {
		protoField := protoFields.Get(i)
		field := NewFieldFromProto(protoField)
		if field != nil {
			fields = append(fields, *field)
		}
	}

	return &fieldGroup{
		name:       name,
		fields:     fields,
		focusIndex: 0,
		focused:    false,
	}
}

func (g *fieldGroup) Empty() bool {
	return len(g.fields) == 0
}

func (g *fieldGroup) Value() map[string]any {
	fields := make(map[string]any)
	for _, field := range g.fields {
		if field.kind == FieldOneof {
			value := field.oneofField.Value()
			for k, v := range value {
				fields[k] = v
			}
			continue
		}
		fields[field.name] = field.Value()
	}
	return fields
}

func (g *fieldGroup) focusedField() *Field {
	if len(g.fields) == 0 || g.focusIndex < 0 || g.focusIndex >= len(g.fields) {
		return nil
	}
	return &g.fields[g.focusIndex]
}

func (g *fieldGroup) focusChild(idx int) {
	if idx < 0 || idx >= len(g.fields) {
		return
	}
	field := &g.fields[idx]
	switch field.kind {
	case FieldText:
		field.textInput.Focus()
	case FieldGroup:
		field.fieldGroup.FocusFirst()
	case FieldList:
		field.listField.FocusFirst()
	case FieldMap:
		field.mapField.FocusFirst()
	case FieldOneof:
		field.oneofField.FocusFirst()
	}
}

func (g *fieldGroup) blurChild(idx int) {
	if idx < 0 || idx >= len(g.fields) {
		return
	}
	field := &g.fields[idx]
	switch field.kind {
	case FieldText:
		field.textInput.Blur()
	case FieldGroup:
		field.fieldGroup.Blur()
	case FieldList:
		field.listField.Blur()
	case FieldMap:
		field.mapField.Blur()
	case FieldOneof:
		field.oneofField.Blur()
	}
}

func (g *fieldGroup) FocusFirst() {
	if len(g.fields) == 0 {
		return
	}
	g.focused = true
	g.focusIndex = 0
	g.focusChild(0)
}

func (g *fieldGroup) FocusLast() {
	if len(g.fields) == 0 {
		return
	}
	g.focused = true
	g.focusIndex = len(g.fields) - 1

	field := &g.fields[g.focusIndex]
	switch field.kind {
	case FieldGroup:
		field.fieldGroup.FocusLast()
	case FieldList:
		field.listField.FocusLast()
	case FieldMap:
		field.mapField.FocusLast()
	case FieldOneof:
		field.oneofField.FocusLast()
	default:
		g.focusChild(g.focusIndex)
	}
}

func (g *fieldGroup) Blur() {
	g.blurChild(g.focusIndex)
	g.focused = false
}

func (g *fieldGroup) NextField() bool {
	if len(g.fields) == 0 {
		return false
	}

	currentField := g.focusedField()
	if currentField == nil {
		return false
	}

	switch currentField.kind {
	case FieldGroup:
		if currentField.fieldGroup.NextField() {
			return true
		}
		currentField.fieldGroup.Blur()
	case FieldList:
		if currentField.listField.NextField() {
			return true
		}
		currentField.listField.Blur()
	case FieldMap:
		if currentField.mapField.NextField() {
			return true
		}
		currentField.mapField.Blur()
	case FieldOneof:
		if currentField.oneofField.NextField() {
			return true
		}
		currentField.oneofField.Blur()
	default:
		g.blurChild(g.focusIndex)
	}

	if g.focusIndex >= len(g.fields)-1 {
		return false
	}

	g.focusIndex++
	nextField := &g.fields[g.focusIndex]
	switch nextField.kind {
	case FieldGroup:
		nextField.fieldGroup.FocusFirst()
	case FieldList:
		nextField.listField.FocusFirst()
	case FieldMap:
		nextField.mapField.FocusFirst()
	case FieldOneof:
		nextField.oneofField.FocusFirst()
	default:
		g.focusChild(g.focusIndex)
	}
	return true
}

func (g *fieldGroup) PrevField() bool {
	if len(g.fields) == 0 {
		return false
	}

	currentField := g.focusedField()
	if currentField == nil {
		return false
	}

	switch currentField.kind {
	case FieldGroup:
		if currentField.fieldGroup.PrevField() {
			return true
		}
		currentField.fieldGroup.Blur()
	case FieldList:
		if currentField.listField.PrevField() {
			return true
		}
		currentField.listField.Blur()
	case FieldMap:
		if currentField.mapField.PrevField() {
			return true
		}
		currentField.mapField.Blur()
	case FieldOneof:
		if currentField.oneofField.PrevField() {
			return true
		}
		currentField.oneofField.Blur()
	default:
		g.blurChild(g.focusIndex)
	}

	if g.focusIndex <= 0 {
		return false
	}

	g.focusIndex--
	prevField := &g.fields[g.focusIndex]
	switch prevField.kind {
	case FieldGroup:
		prevField.fieldGroup.FocusLast()
	case FieldList:
		prevField.listField.FocusLast()
	case FieldMap:
		prevField.mapField.FocusLast()
	case FieldOneof:
		prevField.oneofField.FocusLast()
	default:
		g.focusChild(g.focusIndex)
	}
	return true
}

func (g *fieldGroup) AcceptsTextInput() bool {
	if !g.focused {
		return false
	}
	field := g.focusedField()
	if field == nil {
		return false
	}
	switch field.kind {
	case FieldText:
		return true
	case FieldList:
		if field.listField != nil {
			return field.listField.AcceptsTextInput()
		}
	case FieldMap:
		if field.mapField != nil {
			return field.mapField.AcceptsTextInput()
		}
	case FieldOneof:
		if field.oneofField != nil {
			return field.oneofField.AcceptsTextInput()
		}
	}
	return false
}

func (g *fieldGroup) HandleKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	if !g.focused || len(g.fields) == 0 {
		return nil, false
	}

	field := g.focusedField()
	if field == nil {
		return nil, false
	}

	switch field.kind {
	case FieldEnum, FieldBool:
		switch msg.String() {
		case "left", "right":
			field.enumPicker.Update(msg)
			return nil, true
		}
	case FieldGroup:
		return field.fieldGroup.HandleKey(msg)
	case FieldList:
		return field.listField.HandleKey(msg)
	case FieldMap:
		return field.mapField.HandleKey(msg)
	case FieldOneof:
		return field.oneofField.HandleKey(msg)
	}

	return nil, false
}

func (g *fieldGroup) Update(msg tea.Msg) tea.Cmd {
	if !g.focused || len(g.fields) == 0 {
		return nil
	}

	field := g.focusedField()
	if field == nil {
		return nil
	}

	switch field.kind {
	case FieldText:
		var cmd tea.Cmd
		field.textInput, cmd = field.textInput.Update(msg)
		return cmd
	case FieldGroup:
		return field.fieldGroup.Update(msg)
	case FieldList:
		return field.listField.Update(msg)
	case FieldMap:
		return field.mapField.Update(msg)
	case FieldOneof:
		return field.oneofField.Update(msg)
	}

	return nil
}

func (g *fieldGroup) SetWidth(width int) {
	for i := range g.fields {
		g.fields[i].SetWidth(width)
	}
}

func (g *fieldGroup) View() string {
	return g.ViewWithDepth(1)
}

func (g *fieldGroup) ViewWithDepth(depth int) string {
	var b strings.Builder
	indent := strings.Repeat("  ", depth)

	for i, field := range g.fields {
		isFocused := g.focused && i == g.focusIndex

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
		case FieldList:
			if isFocused {
				b.WriteString(focusedLabelStyle.Render(prefix + field.name + ":"))
			} else {
				b.WriteString(labelStyle.Render(prefix + field.name + ":"))
			}
			b.WriteString("\n")
			b.WriteString(field.listField.ViewWithDepth(depth + 1))
		case FieldMap:
			if isFocused {
				b.WriteString(focusedLabelStyle.Render(prefix + field.name + ":"))
			} else {
				b.WriteString(labelStyle.Render(prefix + field.name + ":"))
			}
			b.WriteString("\n")
			b.WriteString(field.mapField.ViewWithDepth(depth + 1))
		case FieldOneof:
			pickerFocused := isFocused && field.oneofField.focusState == oneofFocusPicker
			if pickerFocused {
				b.WriteString(focusedLabelStyle.Render(prefix + field.name + ": "))
			} else {
				b.WriteString(labelStyle.Render(prefix + field.name + ": "))
			}
			b.WriteString(field.oneofField.picker.View())
			b.WriteString("\n")
			b.WriteString(field.oneofField.ViewWithDepth(depth + 1))
		default:
			if isFocused {
				b.WriteString(focusedLabelStyle.Render(prefix + field.name + ": "))
			} else {
				b.WriteString(labelStyle.Render(prefix + field.name + ": "))
			}
			b.WriteString(field.View())
			b.WriteString("\n")
		}
	}

	return b.String()
}
