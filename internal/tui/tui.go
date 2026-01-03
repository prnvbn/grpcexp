package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/prnvbn/grpcexp/internal/grpc"
)

var _ tea.Model = &Model{}

type Model struct {
	servicesList ServicesList
	grpcClient   *grpc.Client
}

func NewModel(grpcClient *grpc.Client) (Model, error) {
	services, err := grpcClient.ListServices()
	if err != nil {
		return Model{}, err
	}

	return Model{
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
		case "enter":
			if item, ok := m.servicesList.SelectedItem(); ok {
				m.servicesList.SetSelected(item.name)
				// TODO: update tui to show the methods for the selected service
			}
		}
	case tea.WindowSizeMsg:
		m.servicesList.SetSize(msg.Width, msg.Height)
	}

	cmd := m.servicesList.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	return m.servicesList.View()
}
