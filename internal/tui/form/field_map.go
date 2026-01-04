package form

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type mapFocusTarget int

const (
	mapFocusAddButton mapFocusTarget = iota
	mapFocusKey
	mapFocusValue
	mapFocusRemoveButton
)

type mapEntry struct {
	key   Field
	value Field
}

type fieldMap struct {
	name        string
	keyDesc     protoreflect.FieldDescriptor
	valueDesc   protoreflect.FieldDescriptor
	entries     []mapEntry
	focusIndex  int
	focusTarget mapFocusTarget
	focused     bool
	width       int
}

func newMapField(name string, field protoreflect.FieldDescriptor) *fieldMap {
	return &fieldMap{
		name:        name,
		keyDesc:     field.MapKey(),
		valueDesc:   field.MapValue(),
		entries:     make([]mapEntry, 0),
		focusIndex:  0,
		focusTarget: mapFocusAddButton,
		focused:     false,
	}
}

func (m *fieldMap) createEntryFields(index int) *mapEntry {
	keyField := NewFieldFromProto(m.keyDesc)
	valueField := NewFieldFromProto(m.valueDesc)

	if keyField == nil || valueField == nil {
		return nil
	}

	keyField.name = "key"
	valueField.name = "value"

	if m.width > 0 {
		keyField.SetWidth(m.width - 20)
		valueField.SetWidth(m.width - 20)
	}

	return &mapEntry{
		key:   *keyField,
		value: *valueField,
	}
}

func (m *fieldMap) AddEntry() {
	entry := m.createEntryFields(len(m.entries))
	if entry != nil {
		m.entries = append(m.entries, *entry)
	}
}

func (m *fieldMap) RemoveEntry(idx int) {
	if idx < 0 || idx >= len(m.entries) {
		return
	}

	m.entries = append(m.entries[:idx], m.entries[idx+1:]...)

	if len(m.entries) == 0 {
		m.focusIndex = 0
		m.focusTarget = mapFocusAddButton
	} else if m.focusIndex >= len(m.entries) {
		m.focusIndex = len(m.entries) - 1
	}
}

func (m *fieldMap) Value() map[string]any {
	result := make(map[string]any)
	for _, entry := range m.entries {
		keyStr := fmt.Sprintf("%v", entry.key.Value())
		result[keyStr] = entry.value.Value()
	}
	return result
}

func (m *fieldMap) Empty() bool {
	return len(m.entries) == 0
}

func (m *fieldMap) FocusFirst() tea.Cmd {
	m.focused = true
	m.focusIndex = 0
	m.focusTarget = mapFocusAddButton
	return nil
}

func (m *fieldMap) FocusLast() tea.Cmd {
	m.focused = true
	if len(m.entries) == 0 {
		m.focusTarget = mapFocusAddButton
		return nil
	}
	m.focusIndex = len(m.entries) - 1
	m.focusTarget = mapFocusRemoveButton
	return nil
}

func (m *fieldMap) Blur() {
	m.focused = false
	if m.focusIndex < len(m.entries) {
		switch m.focusTarget {
		case mapFocusKey:
			m.entries[m.focusIndex].key.Blur()
		case mapFocusValue:
			m.entries[m.focusIndex].value.Blur()
		}
	}
}

func (m *fieldMap) focusedEntry() *mapEntry {
	if m.focusIndex < 0 || m.focusIndex >= len(m.entries) {
		return nil
	}
	return &m.entries[m.focusIndex]
}

func (m *fieldMap) NextField() bool {
	if !m.focused {
		return false
	}

	switch m.focusTarget {
	case mapFocusAddButton:
		if len(m.entries) > 0 {
			m.focusIndex = 0
			m.focusTarget = mapFocusKey
			m.entries[0].key.Focus()
			return true
		}
		return false

	case mapFocusKey:
		entry := m.focusedEntry()
		if entry == nil {
			return false
		}

		if entry.key.kind == FieldGroup && entry.key.fieldGroup != nil {
			if entry.key.fieldGroup.NextField() {
				return true
			}
			entry.key.fieldGroup.Blur()
		} else {
			entry.key.Blur()
		}

		m.focusTarget = mapFocusValue
		entry.value.Focus()
		return true

	case mapFocusValue:
		entry := m.focusedEntry()
		if entry == nil {
			return false
		}

		if entry.value.kind == FieldGroup && entry.value.fieldGroup != nil {
			if entry.value.fieldGroup.NextField() {
				return true
			}
			entry.value.fieldGroup.Blur()
		} else {
			entry.value.Blur()
		}

		m.focusTarget = mapFocusRemoveButton
		return true

	case mapFocusRemoveButton:
		if m.focusIndex < len(m.entries)-1 {
			m.focusIndex++
			m.focusTarget = mapFocusKey
			m.entries[m.focusIndex].key.Focus()
			return true
		}
		return false
	}

	return false
}

func (m *fieldMap) PrevField() bool {
	if !m.focused {
		return false
	}

	switch m.focusTarget {
	case mapFocusAddButton:
		return false

	case mapFocusKey:
		entry := m.focusedEntry()
		if entry == nil {
			m.focusTarget = mapFocusAddButton
			return true
		}

		if entry.key.kind == FieldGroup && entry.key.fieldGroup != nil {
			if entry.key.fieldGroup.PrevField() {
				return true
			}
			entry.key.fieldGroup.Blur()
		} else {
			entry.key.Blur()
		}

		if m.focusIndex == 0 {
			m.focusTarget = mapFocusAddButton
			return true
		}

		m.focusIndex--
		m.focusTarget = mapFocusRemoveButton
		return true

	case mapFocusValue:
		entry := m.focusedEntry()
		if entry == nil {
			return false
		}

		if entry.value.kind == FieldGroup && entry.value.fieldGroup != nil {
			if entry.value.fieldGroup.PrevField() {
				return true
			}
			entry.value.fieldGroup.Blur()
		} else {
			entry.value.Blur()
		}

		m.focusTarget = mapFocusKey
		entry.key.Focus()
		return true

	case mapFocusRemoveButton:
		m.focusTarget = mapFocusValue
		if m.focusIndex < len(m.entries) {
			m.entries[m.focusIndex].value.Focus()
		}
		return true
	}

	return false
}

func (m *fieldMap) AcceptsTextInput() bool {
	if m.focusTarget != mapFocusKey && m.focusTarget != mapFocusValue {
		return false
	}
	entry := m.focusedEntry()
	if entry == nil {
		return false
	}
	if m.focusTarget == mapFocusKey {
		return entry.key.AcceptsTextInput()
	}
	return entry.value.AcceptsTextInput()
}

func (m *fieldMap) HandleKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	if !m.focused {
		return nil, false
	}

	key := msg.String()

	if key == "enter" || key == " " {
		switch m.focusTarget {
		case mapFocusAddButton:
			m.AddEntry()
			return nil, true
		case mapFocusRemoveButton:
			m.RemoveEntry(m.focusIndex)
			return nil, true
		}
	}

	isArrowKey := key == "left" || key == "right"
	isVimKey := key == "h" || key == "l"
	entryAcceptsText := (m.focusTarget == mapFocusKey || m.focusTarget == mapFocusValue) && m.AcceptsTextInput()

	if isArrowKey || (isVimKey && !entryAcceptsText) {
		entry := m.focusedEntry()
		switch m.focusTarget {
		case mapFocusKey:
			if entry != nil {
				if entry.key.kind == FieldGroup {
					cmd, handled := entry.key.HandleKey(msg)
					if handled {
						return cmd, true
					}
				}
				if entry.key.kind == FieldEnum || entry.key.kind == FieldBool {
					cmd, handled := entry.key.HandleKey(msg)
					if handled {
						return cmd, true
					}
				}
			}
			if key == "right" || key == "l" {
				if entry != nil {
					entry.key.Blur()
					m.focusTarget = mapFocusValue
					entry.value.Focus()
					return nil, true
				}
			}
		case mapFocusValue:
			if entry != nil {
				if entry.value.kind == FieldGroup {
					cmd, handled := entry.value.HandleKey(msg)
					if handled {
						return cmd, true
					}
				}
				if entry.value.kind == FieldEnum || entry.value.kind == FieldBool {
					cmd, handled := entry.value.HandleKey(msg)
					if handled {
						return cmd, true
					}
				}
			}
			if key == "left" || key == "h" {
				if entry != nil {
					entry.value.Blur()
					m.focusTarget = mapFocusKey
					entry.key.Focus()
					return nil, true
				}
			}
			if key == "right" || key == "l" {
				if entry != nil {
					entry.value.Blur()
					m.focusTarget = mapFocusRemoveButton
					return nil, true
				}
			}
		case mapFocusRemoveButton:
			if key == "left" || key == "h" {
				m.focusTarget = mapFocusValue
				if entry != nil {
					entry.value.Focus()
				}
				return nil, true
			}
		}
	}

	entry := m.focusedEntry()
	if entry != nil {
		switch m.focusTarget {
		case mapFocusKey:
			return entry.key.HandleKey(msg)
		case mapFocusValue:
			return entry.value.HandleKey(msg)
		}
	}

	return nil, false
}

func (m *fieldMap) Update(msg tea.Msg) tea.Cmd {
	if !m.focused {
		return nil
	}

	entry := m.focusedEntry()
	if entry == nil {
		return nil
	}

	switch m.focusTarget {
	case mapFocusKey:
		return entry.key.Update(msg)
	case mapFocusValue:
		return entry.value.Update(msg)
	}

	return nil
}

func (m *fieldMap) SetWidth(width int) {
	m.width = width
	for i := range m.entries {
		m.entries[i].key.SetWidth(width - 20)
		m.entries[i].value.SetWidth(width - 20)
	}
}

func (m *fieldMap) View() string {
	return m.ViewWithDepth(0)
}

func (m *fieldMap) ViewWithDepth(depth int) string {
	var b strings.Builder
	indent := strings.Repeat("  ", depth)

	addButtonFocused := m.focused && m.focusTarget == mapFocusAddButton
	if addButtonFocused {
		b.WriteString(focusedLabelStyle.Render(indent + "> [+] Add"))
	} else {
		b.WriteString(labelStyle.Render(indent + "  [+] Add"))
	}
	b.WriteString("\n")

	for i, entry := range m.entries {
		keyFocused := m.focused && m.focusTarget == mapFocusKey && m.focusIndex == i
		valueFocused := m.focused && m.focusTarget == mapFocusValue && m.focusIndex == i
		removeFocused := m.focused && m.focusTarget == mapFocusRemoveButton && m.focusIndex == i

		entryFocused := keyFocused || valueFocused || removeFocused

		var entryPrefix string
		if entryFocused {
			entryPrefix = indent + "  > "
		} else {
			entryPrefix = indent + "    "
		}

		if entryFocused {
			b.WriteString(focusedLabelStyle.Render(fmt.Sprintf("%s[%d]:", entryPrefix, i)))
		} else {
			b.WriteString(labelStyle.Render(fmt.Sprintf("%s[%d]:", entryPrefix, i)))
		}
		b.WriteString("\n")

		fieldIndent := strings.Repeat("  ", depth+2)
		var keyPrefix string
		if keyFocused {
			keyPrefix = fieldIndent + "> "
		} else {
			keyPrefix = fieldIndent + "  "
		}

		if entry.key.kind == FieldGroup {
			if keyFocused {
				b.WriteString(focusedLabelStyle.Render(keyPrefix + "key:"))
			} else {
				b.WriteString(labelStyle.Render(keyPrefix + "key:"))
			}
			b.WriteString("\n")
			b.WriteString(entry.key.fieldGroup.ViewWithDepth(depth + 3))
		} else {
			if keyFocused {
				b.WriteString(focusedLabelStyle.Render(keyPrefix + "key: "))
			} else {
				b.WriteString(labelStyle.Render(keyPrefix + "key: "))
			}
			b.WriteString(entry.key.View())
			b.WriteString("\n")
		}

		var valuePrefix string
		if valueFocused {
			valuePrefix = fieldIndent + "> "
		} else {
			valuePrefix = fieldIndent + "  "
		}

		if entry.value.kind == FieldGroup {
			if valueFocused {
				b.WriteString(focusedLabelStyle.Render(valuePrefix + "value:"))
			} else {
				b.WriteString(labelStyle.Render(valuePrefix + "value:"))
			}
			b.WriteString("\n")
			b.WriteString(entry.value.fieldGroup.ViewWithDepth(depth + 3))
		} else {
			if valueFocused {
				b.WriteString(focusedLabelStyle.Render(valuePrefix + "value: "))
			} else {
				b.WriteString(labelStyle.Render(valuePrefix + "value: "))
			}
			b.WriteString(entry.value.View())
			b.WriteString("\n")
		}

		removeIndent := strings.Repeat("  ", depth+2)
		if removeFocused {
			b.WriteString(focusedLabelStyle.Render(removeIndent + "> [-] Remove"))
		} else {
			b.WriteString(labelStyle.Render(removeIndent + "  [-] Remove"))
		}
		b.WriteString("\n")
	}

	return b.String()
}
