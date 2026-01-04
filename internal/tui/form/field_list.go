package form

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type focusTarget int

const (
	focusAddButton focusTarget = iota
	focusItem
	focusRemoveButton
)

type fieldList struct {
	name        string
	elementDesc protoreflect.FieldDescriptor
	items       []Field
	focusIndex  int
	focusTarget focusTarget
	focused     bool
	width       int
}

func newListField(name string, field protoreflect.FieldDescriptor) *fieldList {
	return &fieldList{
		name:        name,
		elementDesc: field,
		items:       make([]Field, 0),
		focusIndex:  0,
		focusTarget: focusAddButton,
		focused:     false,
	}
}

func (l *fieldList) createItemField(index int) *Field {
	field := NewFieldFromProto(l.elementDesc)
	if field != nil {
		field.name = fmt.Sprintf("[%d]", index)
		if l.width > 0 {
			field.SetWidth(l.width - 18)
		}
	}
	return field
}

func (l *fieldList) AddItem() {
	field := l.createItemField(len(l.items))
	if field != nil {
		l.items = append(l.items, *field)
	}
}

func (l *fieldList) RemoveItem(idx int) {
	if idx < 0 || idx >= len(l.items) {
		return
	}

	l.items = append(l.items[:idx], l.items[idx+1:]...)

	for i := idx; i < len(l.items); i++ {
		l.items[i].name = fmt.Sprintf("[%d]", i)
	}

	if len(l.items) == 0 {
		l.focusIndex = 0
		l.focusTarget = focusAddButton
	} else if l.focusIndex >= len(l.items) {
		l.focusIndex = len(l.items) - 1
	}
}

func (l *fieldList) Value() []any {
	values := make([]any, len(l.items))
	for i, item := range l.items {
		values[i] = item.Value()
	}
	return values
}

func (l *fieldList) Empty() bool {
	return len(l.items) == 0
}

func (l *fieldList) FocusFirst() tea.Cmd {
	l.focused = true
	l.focusIndex = 0
	l.focusTarget = focusAddButton
	return nil
}

func (l *fieldList) FocusLast() tea.Cmd {
	l.focused = true
	if len(l.items) == 0 {
		l.focusTarget = focusAddButton
		return nil
	}
	l.focusIndex = len(l.items) - 1
	l.focusTarget = focusRemoveButton
	return nil
}

func (l *fieldList) Blur() {
	l.focused = false
	if l.focusTarget == focusItem && l.focusIndex < len(l.items) {
		l.items[l.focusIndex].Blur()
	}
}

func (l *fieldList) focusedItem() *Field {
	if l.focusIndex < 0 || l.focusIndex >= len(l.items) {
		return nil
	}
	return &l.items[l.focusIndex]
}

func (l *fieldList) NextField() bool {
	if !l.focused {
		return false
	}

	switch l.focusTarget {
	case focusAddButton:
		if len(l.items) > 0 {
			l.focusIndex = 0
			l.focusTarget = focusItem
			l.items[0].Focus()
			return true
		}
		return false

	case focusItem:
		item := l.focusedItem()
		if item == nil {
			return false
		}

		if item.kind == FieldGroup && item.fieldGroup != nil {
			if item.fieldGroup.NextField() {
				return true
			}
			item.fieldGroup.Blur()
		} else {
			item.Blur()
		}

		l.focusTarget = focusRemoveButton
		return true

	case focusRemoveButton:
		if l.focusIndex < len(l.items)-1 {
			l.focusIndex++
			l.focusTarget = focusItem
			l.items[l.focusIndex].Focus()
			return true
		}
		return false
	}

	return false
}

func (l *fieldList) PrevField() bool {
	if !l.focused {
		return false
	}

	switch l.focusTarget {
	case focusAddButton:
		return false

	case focusItem:
		item := l.focusedItem()
		if item == nil {
			l.focusTarget = focusAddButton
			return true
		}

		if item.kind == FieldGroup && item.fieldGroup != nil {
			if item.fieldGroup.PrevField() {
				return true
			}
			item.fieldGroup.Blur()
		} else {
			item.Blur()
		}

		if l.focusIndex == 0 {
			l.focusTarget = focusAddButton
			return true
		}

		l.focusIndex--
		l.focusTarget = focusRemoveButton
		return true

	case focusRemoveButton:
		l.focusTarget = focusItem
		if l.focusIndex < len(l.items) {
			l.items[l.focusIndex].Focus()
		}
		return true
	}

	return false
}

func (l *fieldList) AcceptsTextInput() bool {
	if l.focusTarget != focusItem {
		return false
	}
	item := l.focusedItem()
	if item == nil {
		return false
	}
	return item.AcceptsTextInput()
}

func (l *fieldList) HandleKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	if !l.focused {
		return nil, false
	}

	key := msg.String()

	if key == "enter" || key == " " {
		switch l.focusTarget {
		case focusAddButton:
			l.AddItem()
			return nil, true
		case focusRemoveButton:
			l.RemoveItem(l.focusIndex)
			return nil, true
		}
	}

	isArrowKey := key == "left" || key == "right"
	itemAcceptsText := l.focusTarget == focusItem && l.AcceptsTextInput()

	if isArrowKey || (!itemAcceptsText) {
		switch l.focusTarget {
		case focusItem:
			item := l.focusedItem()
			if item != nil {
				if item.kind == FieldGroup {
					cmd, handled := item.HandleKey(msg)
					if handled {
						return cmd, true
					}
				}
				if item.kind == FieldEnum || item.kind == FieldBool {
					cmd, handled := item.HandleKey(msg)
					if handled {
						return cmd, true
					}
				}
			}
			if key == "right" {
				if len(l.items) > 0 && l.focusIndex < len(l.items) {
					l.items[l.focusIndex].Blur()
					l.focusTarget = focusRemoveButton
					return nil, true
				}
			}
		case focusRemoveButton:
			if key == "left" {
				l.focusTarget = focusItem
				if l.focusIndex < len(l.items) {
					return l.items[l.focusIndex].Focus(), true
				}
				return nil, true
			}
		}
	}

	if l.focusTarget == focusItem {
		item := l.focusedItem()
		if item != nil {
			return item.HandleKey(msg)
		}
	}

	return nil, false
}

func (l *fieldList) Update(msg tea.Msg) tea.Cmd {
	if !l.focused || l.focusTarget != focusItem {
		return nil
	}

	item := l.focusedItem()
	if item == nil {
		return nil
	}

	return item.Update(msg)
}

func (l *fieldList) SetWidth(width int) {
	l.width = width
	for i := range l.items {
		l.items[i].SetWidth(width - 18)
	}
}

func (l *fieldList) View() string {
	return l.ViewWithDepth(0)
}

func (l *fieldList) ViewWithDepth(depth int) string {
	var b strings.Builder
	indent := strings.Repeat("  ", depth)

	addButtonFocused := l.focused && l.focusTarget == focusAddButton
	if addButtonFocused {
		b.WriteString(focusedLabelStyle.Render(indent + "> [+] Add"))
	} else {
		b.WriteString(labelStyle.Render(indent + "  [+] Add"))
	}
	b.WriteString("\n")

	for i, item := range l.items {
		itemFocused := l.focused && l.focusTarget == focusItem && l.focusIndex == i
		removeFocused := l.focused && l.focusTarget == focusRemoveButton && l.focusIndex == i

		var prefix string
		if itemFocused {
			prefix = indent + "  > "
		} else {
			prefix = indent + "    "
		}

		if item.kind == FieldGroup {
			if itemFocused {
				b.WriteString(focusedLabelStyle.Render(fmt.Sprintf("%s%s:", prefix, item.name)))
			} else {
				b.WriteString(labelStyle.Render(fmt.Sprintf("%s%s:", prefix, item.name)))
			}
			b.WriteString("\n")
			b.WriteString(item.fieldGroup.ViewWithDepth(depth + 2))

			removePrefix := strings.Repeat("  ", depth+2)
			if removeFocused {
				b.WriteString(focusedLabelStyle.Render(removePrefix + "> [-] Remove"))
			} else {
				b.WriteString(labelStyle.Render(removePrefix + "  [-] Remove"))
			}
			b.WriteString("\n")
		} else {
			if itemFocused {
				b.WriteString(focusedLabelStyle.Render(fmt.Sprintf("%s%s: ", prefix, item.name)))
			} else {
				b.WriteString(labelStyle.Render(fmt.Sprintf("%s%s: ", prefix, item.name)))
			}
			b.WriteString(item.View())

			if removeFocused {
				b.WriteString(focusedLabelStyle.Render("  > [-]"))
			} else {
				b.WriteString(labelStyle.Render("    [-]"))
			}
			b.WriteString("\n")
		}
	}

	return b.String()
}
