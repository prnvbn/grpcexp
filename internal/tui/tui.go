package tui

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/prnvbn/grpcexp/internal/grpc"
)

var _ tea.Model = &Model{}

type viewState int

const (
	viewServices viewState = iota
	viewMethods
)

type Model struct {
	state        viewState
	servicesList ServicesList
	methodsList  *MethodsList
	grpcClient   *grpc.Client
	width        int
	height       int
}

func NewModel(grpcClient *grpc.Client) (Model, error) {
	services, err := grpcClient.ListServices()
	if err != nil {
		return Model{}, err
	}

	return Model{
		state:        viewServices,
		servicesList: NewServicesList(services),
		grpcClient:   grpcClient,
	}, nil
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "esc", "backspace":
			if m.state == viewMethods {
				m.state = viewServices
				m.methodsList = nil
				return m, nil
			}
		case "enter":
			switch m.state {
			case viewServices:
				svc, ok := m.servicesList.SelectedItem()
				if !ok {
					fmt.Fprintf(os.Stderr, "no service selected\n")
					return m, tea.Quit
				}
				m.servicesList.SetSelected(svc.name)
				methods, err := m.grpcClient.ListMethods(svc.name)
				if err != nil {
					fmt.Fprintf(os.Stderr, "error listing methods: %v\n", err)
					return m, tea.Quit
				}
				methodsList := NewMethodsList(svc.name, methods)
				methodsList.SetSize(m.width, m.height)
				m.methodsList = &methodsList
				m.state = viewMethods
			case viewMethods:

			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.servicesList.SetSize(msg.Width, msg.Height)
		if m.methodsList != nil {
			m.methodsList.SetSize(msg.Width, msg.Height)
		}
	}

	var cmd tea.Cmd
	switch m.state {
	case viewMethods:
		if m.methodsList != nil {
			cmd = m.methodsList.Update(msg)
		}
	default:
		cmd = m.servicesList.Update(msg)
	}
	return m, cmd
}

func (m Model) View() string {
	switch m.state {
	case viewMethods:
		if m.methodsList != nil {
			return m.methodsList.View()
		} else {
			return "No methods found for selected service"
		}
	}
	return m.servicesList.View()
}
