package decks

import "charm.land/lipgloss/v2"

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#f8f8f2")).
			Background(lipgloss.Color("#2f5d50")).
			Padding(0, 1)

	sectionLabelStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#f2d16b"))

	mutedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8a9a96"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ff6b6b"))

	inputStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#3f6f66")).
			Padding(0, 1)

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#3f6f66")).
			Padding(0, 1)

	activePanelStyle = panelStyle.
				BorderForeground(lipgloss.Color("#f2d16b"))

	bannerPanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#f2d16b")).
				Background(lipgloss.Color("#1f2a2e")).
				Padding(0, 1)

	bannerMetaStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8dd7c7"))

	bannerTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#f8f8f2")).
				Background(lipgloss.Color("#2f5d50")).
				Padding(0, 1)

	bannerLargeTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#f8f8f2"))

	bannerHeroArtStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#8dd7c7")).
				Background(lipgloss.Color("#182124"))

	bannerStatStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#1f2a2e")).
			Background(lipgloss.Color("#f2d16b"))

	cardTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#f2d16b"))

	statChipStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#1f2a2e")).
			Background(lipgloss.Color("#8dd7c7"))

	listTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#f8f8f2"))

	listMetaStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8a9a96"))

	listSelectedTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#e778ff"))

	listSelectedMetaStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#d783e8"))
)
