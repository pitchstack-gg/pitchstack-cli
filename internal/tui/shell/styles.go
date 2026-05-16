package shell

import "charm.land/lipgloss/v2"

var (
	activeTabStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("0")).
			Background(lipgloss.Color("14"))

	tabStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("7")).
			Background(lipgloss.Color("8"))

	mutedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8"))
)
