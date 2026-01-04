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
	default:
		g.focusChild(g.focusIndex)
	}
	return true
}

func (g *fieldGroup) AcceptsTextInput() bool {
	field := g.focusedField()
	if field == nil {
		return false
	}
	if field.kind == FieldText {
		return true
	}
	if field.kind == FieldList && field.listField != nil {
		return field.listField.AcceptsTextInput()
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

	if field.kind == FieldEnum || field.kind == FieldBool {
		switch msg.String() {
		case "left", "h", "right", "l":
			field.enumPicker.Update(msg)
			return nil, true
		}
	}

	if field.kind == FieldGroup {
		return field.fieldGroup.HandleKey(msg)
	}

	if field.kind == FieldList {
		return field.listField.HandleKey(msg)
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

	if field.kind == FieldText {
		var cmd tea.Cmd
		field.textInput, cmd = field.textInput.Update(msg)
		return cmd
	}

	if field.kind == FieldGroup {
		return field.fieldGroup.Update(msg)
	}

	if field.kind == FieldList {
		return field.listField.Update(msg)
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
