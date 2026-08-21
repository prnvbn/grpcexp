package call

import (
	"maps"
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
		if field.optional && !field.present {
			continue
		}
		if field.kind == FieldOneof {
			value := field.oneofField.Value()
			maps.Copy(fields, value)
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
	g.fields[idx].Focus()
}

func (g *fieldGroup) blurChild(idx int) {
	if idx < 0 || idx >= len(g.fields) {
		return
	}
	g.fields[idx].Blur()
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

	g.fields[g.focusIndex].FocusFromEnd()
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
	}

	if currentField.optional && !currentField.optionalControlFocused {
		currentField.Blur()
		currentField.optionalControlFocused = true
		return true
	}
	g.blurChild(g.focusIndex)

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
	if currentField.optionalControlFocused {
		currentField.focusValueFromEnd()
		return true
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
	g.fields[g.focusIndex].FocusFromEnd()
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
		return !field.optionalControlFocused
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

	return field.HandleKey(msg)
}

func (g *fieldGroup) Update(msg tea.Msg) tea.Cmd {
	if !g.focused || len(g.fields) == 0 {
		return nil
	}

	field := g.focusedField()
	if field == nil {
		return nil
	}

	return field.Update(msg)
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
		inputFocused := isFocused && !field.optionalControlFocused

		var prefix string
		if inputFocused {
			prefix = indent + "> "
		} else {
			prefix = indent + "  "
		}

		switch field.kind {
		case FieldGroup:
			if inputFocused {
				b.WriteString(focusedLabelStyle.Render(prefix + field.name + ":"))
			} else {
				b.WriteString(labelStyle.Render(prefix + field.name + ":"))
			}
			b.WriteString("\n")
			b.WriteString(field.fieldGroup.ViewWithDepth(depth + 1))
		case FieldList:
			if inputFocused {
				b.WriteString(focusedLabelStyle.Render(prefix + field.name + ":"))
			} else {
				b.WriteString(labelStyle.Render(prefix + field.name + ":"))
			}
			b.WriteString("\n")
			b.WriteString(field.listField.ViewWithDepth(depth + 1))
		case FieldMap:
			if inputFocused {
				b.WriteString(focusedLabelStyle.Render(prefix + field.name + ":"))
			} else {
				b.WriteString(labelStyle.Render(prefix + field.name + ":"))
			}
			b.WriteString("\n")
			b.WriteString(field.mapField.ViewWithDepth(depth + 1))
		case FieldOneof:
			pickerFocused := inputFocused && field.oneofField.focusState == oneofFocusPicker
			if pickerFocused {
				b.WriteString(focusedLabelStyle.Render(prefix + field.name + ": "))
			} else {
				b.WriteString(labelStyle.Render(prefix + field.name + ": "))
			}
			b.WriteString(field.oneofField.picker.View())
			b.WriteString("\n")
			b.WriteString(field.oneofField.ViewWithDepth(depth + 1))
		default:
			if inputFocused {
				b.WriteString(focusedLabelStyle.Render(prefix + field.name + ": "))
			} else {
				b.WriteString(labelStyle.Render(prefix + field.name + ": "))
			}
			b.WriteString(field.inputView(prefix))
			b.WriteString(field.checkbox(isFocused && field.optionalControlFocused))
			b.WriteString("\n")
		}
	}

	return b.String()
}
