package tui

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	selectedStyle                   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	_             list.Item         = &svcItem{}
	_             list.ItemDelegate = &minimalDelegate{}
)

type ServicesList struct {
	list     list.Model
	selected string
}

func NewServicesList(services []string) ServicesList {
	items := make([]list.Item, len(services))
	for i, svc := range services {
		items[i] = svcItem{name: svc}
	}

	l := list.New(items, minimalDelegate{}, 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowFilter(true)
	l.SetShowHelp(true)
	l.SetShowPagination(false)

	return ServicesList{
		list: l,
	}
}

func (s *ServicesList) SetSize(width, height int) {
	s.list.SetSize(width, height)
}

func (s *ServicesList) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	s.list, cmd = s.list.Update(msg)
	return cmd
}

func (s *ServicesList) View() string {
	return s.list.View()
}

func (s *ServicesList) SelectedItem() (svcItem, bool) {
	item, ok := s.list.SelectedItem().(svcItem)
	return item, ok
}

func (s *ServicesList) Selected() string {
	return s.selected
}

func (s *ServicesList) SetSelected(name string) {
	s.selected = name
}

type svcItem struct {
	name string
}

func (i svcItem) Title() string       { return i.name }
func (i svcItem) Description() string { return "" }
func (i svcItem) FilterValue() string { return i.name }

// docs: https://github.com/charmbracelet/bubbles/tree/master/list#customizing-styles
type minimalDelegate struct{}

func (d minimalDelegate) Height() int                             { return 1 }
func (d minimalDelegate) Spacing() int                            { return 0 }
func (d minimalDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d minimalDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	svc, ok := item.(list.DefaultItem)
	if !ok {
		return
	}
	if index == m.Index() {
		fmt.Fprintf(w, "> %s", selectedStyle.Render(svc.Title()))
	} else {
		fmt.Fprintf(w, "  %s", svc.Title())
	}
}
