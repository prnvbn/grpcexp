package call

import "charm.land/lipgloss/v2"

var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			MarginBottom(1)

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243"))

	focusedLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("205")).
				Bold(true)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true)

	unselectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243"))

	bracketStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243"))
)
