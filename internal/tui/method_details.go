package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var _ tea.Model = &MethodDetails{}

type MethodDetails struct {
	method protoreflect.MethodDescriptor
}

func NewMethodDetails(method protoreflect.MethodDescriptor) MethodDetails {
	return MethodDetails{method}
}

func (m *MethodDetails) Init() tea.Cmd {
	return nil
}

func (m *MethodDetails) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m *MethodDetails) View() string {
	return fmt.Sprintf("Method Details: %s", m.method.FullName())
}
