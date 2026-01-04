package tui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var _ list.Item = &methodItem{}

type MethodsList struct {
	list        list.Model
	serviceName string
}

func NewMethodsList(serviceName string, methods []protoreflect.MethodDescriptor) MethodsList {
	items := make([]list.Item, len(methods))
	for i, method := range methods {
		items[i] = methodItem{method}
	}

	l := list.New(items, minimalDelegate{}, 0, 0)
	l.Title = serviceName
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowFilter(true)
	l.SetShowHelp(true)
	l.SetShowPagination(false)

	l.KeyMap.CursorUp.SetKeys("up")
	l.KeyMap.CursorUp.SetHelp("↑", "up")
	l.KeyMap.CursorDown.SetKeys("down")
	l.KeyMap.CursorDown.SetHelp("↓", "down")

	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "navigate")),
		}
	}

	return MethodsList{
		list:        l,
		serviceName: serviceName,
	}
}

func (m *MethodsList) SetSize(width, height int) {
	m.list.SetSize(width, height)
}

func (m *MethodsList) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			if m.list.Index() >= len(m.list.Items())-1 {
				m.list.Select(0)
			} else {
				m.list.CursorDown()
			}
			return nil
		case "shift+tab":
			if m.list.Index() == 0 {
				m.list.Select(len(m.list.Items()) - 1)
			} else {
				m.list.CursorUp()
			}
			return nil
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return cmd
}

func (m *MethodsList) View() string {
	return m.list.View()
}

func (m *MethodsList) SelectedItem() (methodItem, bool) {
	item, ok := m.list.SelectedItem().(methodItem)
	return item, ok
}

type methodItem struct {
	method protoreflect.MethodDescriptor
}

func (i methodItem) Title() string       { return string(i.method.FullName()) }
func (i methodItem) Description() string { return "" }
func (i methodItem) FilterValue() string { return string(i.method.FullName()) }
