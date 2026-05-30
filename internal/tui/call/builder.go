package call

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type Builder struct {
	root              *fieldGroup
	submitFocused     bool
	unsupportedFields []string
}

func NewBuilder(msgDesc protoreflect.MessageDescriptor) *Builder {
	b := &Builder{
		root: buildFieldGroup(msgDesc),
	}

	if b.root.Empty() {
		b.submitFocused = true
	} else {
		b.root.FocusFirst()
	}

	return b
}

func (b *Builder) HandleKey(msg tea.KeyMsg, onSubmit func() tea.Cmd) (tea.Cmd, bool) {
	switch msg.String() {
	case "tab", "down":
		b.nextField()
		return nil, true
	case "enter", " ":
		if b.submitFocused {
			return onSubmit(), true
		}

		cmd, handled := b.root.HandleKey(msg)
		if handled {
			return cmd, true
		}
		b.nextField()
		return nil, true
	case "shift+tab", "up":
		b.prevField()
		return nil, true
	case "left", "right":
		cmd, handled := b.root.HandleKey(msg)
		if handled {
			return cmd, true
		}
	}
	return nil, false
}

func (b *Builder) Update(msg tea.Msg) tea.Cmd {
	return b.root.Update(msg)
}

func (b *Builder) View(submitLabel string, active bool, disabled bool) string {
	var out strings.Builder

	if b.root.Empty() {
		out.WriteString(labelStyle.Render("No input fields."))
		out.WriteString("\n")
	} else {
		out.WriteString(b.root.ViewWithDepth(0))
	}

	if len(b.unsupportedFields) > 0 {
		out.WriteString(labelStyle.Render(fmt.Sprintf("(unsupported: %s)",
			strings.Join(b.unsupportedFields, ", "))))
		out.WriteString("\n")
	}

	out.WriteString("\n")
	label := fmt.Sprintf("  [%s]", submitLabel)
	if b.submitFocused && active && !disabled {
		label = fmt.Sprintf("> [%s]", submitLabel)
		out.WriteString(focusedLabelStyle.Render(label))
	} else {
		out.WriteString(labelStyle.Render(label))
	}

	return out.String()
}

func (b *Builder) SetWidth(width int) {
	b.root.SetWidth(width)
}

func (b *Builder) AcceptsTextInput() bool {
	return b.root.AcceptsTextInput()
}

func (b *Builder) Value() map[string]any {
	return b.root.Value()
}

func (b *Builder) ResetToSubmit() {
	b.root.Blur()
	b.submitFocused = true
}

func (b *Builder) Deactivate() {
	b.root.Blur()
}

func (b *Builder) Activate() {
	if b.submitFocused {
		b.root.Blur()
		return
	}
	b.root.FocusFirst()
}

func (b *Builder) nextField() {
	if b.submitFocused {
		return
	}
	if !b.root.NextField() {
		b.root.Blur()
		b.submitFocused = true
	}
}

func (b *Builder) prevField() {
	if b.submitFocused {
		b.submitFocused = false
		b.root.FocusLast()
		return
	}
	b.root.PrevField()
}

func buildFieldGroup(msgDesc protoreflect.MessageDescriptor) *fieldGroup {
	g := &fieldGroup{
		name:       "",
		fields:     make([]Field, 0),
		focusIndex: 0,
		focused:    false,
	}

	fields := msgDesc.Fields()

	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		fieldName := string(field.Name())

		if field.IsMap() {
			mapField := NewMapField(fieldName, field)
			if mapField != nil {
				g.fields = append(g.fields, *mapField)
			}
			continue
		}

		if field.ContainingOneof() != nil {
			continue
		}

		if field.IsList() {
			listField := NewListField(fieldName, field)
			if listField != nil {
				g.fields = append(g.fields, *listField)
			}
			continue
		}

		formField := NewFieldFromProto(field)
		if formField != nil {
			g.fields = append(g.fields, *formField)
		}
	}

	oneofs := msgDesc.Oneofs()
	for i := 0; i < oneofs.Len(); i++ {
		oneof := oneofs.Get(i)
		oneofName := string(oneof.Name())
		oneofField := NewOneofField(oneofName, oneof)
		if oneofField != nil {
			g.fields = append(g.fields, *oneofField)
		}
	}

	return g
}
