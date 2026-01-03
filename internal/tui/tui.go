package tui

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/prnvbn/grpcexp/internal/grpc"
	"github.com/prnvbn/grpcexp/internal/tui/form"
)

var _ tea.Model = &Model{}

type screenState int

const (
	screenServices screenState = iota
	screenMethods
	screenCallMethod
)

type Model struct {
	state screenState

	servicesList   ServicesList
	methodsList    *MethodsList
	callMethodForm *form.Form

	grpcClient *grpc.Client
	width      int
	height     int
}

func NewModel(grpcClient *grpc.Client) (Model, error) {
	services, err := grpcClient.ListServices()
	if err != nil {
		return Model{}, err
	}

	return Model{
		state:        screenServices,
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
		model, cmd, done := m.handleKey(msg)
		if done {
			return model, cmd
		}
	case tea.WindowSizeMsg:
		m.resize(msg)
	}

	return m, m.forwardToScreen(msg)
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	switch msg.String() {
	case "q", "ctrl+c":
		return *m, tea.Quit, true
	case "esc":
		model, cmd := m.goBack()
		return model, cmd, true
	case "enter":
		return m.drillDown()
	default:
		return *m, nil, false
	}
}

func (m *Model) goBack() (tea.Model, tea.Cmd) {
	switch m.state {
	case screenServices:
		return *m, tea.Quit
	case screenMethods:
		m.state = screenServices
		m.methodsList = nil
		return *m, nil
	case screenCallMethod:
		m.state = screenMethods
		m.callMethodForm = nil
		return *m, nil
	default:
		panic(fmt.Sprintf("unknown state - non exhaustive switch for go back: %d", m.state))
	}
}

func (m *Model) drillDown() (tea.Model, tea.Cmd, bool) {
	switch m.state {
	case screenServices:
		svc, ok := m.servicesList.SelectedItem()
		if !ok {
			fmt.Fprintf(os.Stderr, "no service selected\n")
			return *m, tea.Quit, true
		}

		methods, err := m.grpcClient.ListMethods(svc.name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error listing methods: %v\n", err)
			return *m, tea.Quit, true
		}

		methodsList := NewMethodsList(svc.name, methods)
		methodsList.SetSize(m.width, m.height)
		m.methodsList = &methodsList
		m.state = screenMethods
		return *m, nil, true
	case screenMethods:
		md, ok := m.methodsList.SelectedItem()
		if !ok {
			fmt.Fprintf(os.Stderr, "no method selected\n")
			return *m, tea.Quit, true
		}

		methodDetails := form.NewForm(md.method, m.grpcClient)
		methodDetails.SetSize(m.width, m.height)
		m.callMethodForm = &methodDetails
		m.state = screenCallMethod
		return *m, m.callMethodForm.Init(), true
	case screenCallMethod:
		return *m, nil, false
	default:
		panic(fmt.Sprintf("unknown state - non exhaustive switch for drill down: %d", m.state))
	}
}

func (m *Model) resize(msg tea.WindowSizeMsg) {
	m.width = msg.Width
	m.height = msg.Height
	m.servicesList.SetSize(msg.Width, msg.Height)
	if m.methodsList != nil {
		m.methodsList.SetSize(msg.Width, msg.Height)
	}
	if m.callMethodForm != nil {
		m.callMethodForm.SetSize(msg.Width, msg.Height)
	}
}

func (m *Model) forwardToScreen(msg tea.Msg) tea.Cmd {
	switch m.state {
	case screenServices:
		return m.servicesList.Update(msg)
	case screenMethods:
		if m.methodsList != nil {
			return m.methodsList.Update(msg)
		}
	case screenCallMethod:
		if m.callMethodForm != nil {
			_, cmd := m.callMethodForm.Update(msg)
			return cmd
		}
	default:
		panic("unknown state - non exhaustive switch for update")
	}
	return nil
}

func (m Model) View() string {
	switch m.state {
	case screenServices:
		return m.servicesList.View()
	case screenMethods:
		if m.methodsList != nil {
			return m.methodsList.View()
		}
		return "No methods found for selected service"
	case screenCallMethod:
		if m.callMethodForm != nil {
			return m.callMethodForm.View()
		}
		return "No method details found"
	}
	panic(fmt.Sprintf("unknown state - non exhaustive switch for screen state: %d", m.state))
}
