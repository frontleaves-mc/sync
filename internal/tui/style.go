package tui

import "github.com/charmbracelet/lipgloss"

const (
	// MinContentWidth 内容区域最小宽度，确保 logo 能完整显示。
	MinContentWidth = 93
)

// ClampWidth 确保内容宽度不低于最小值。
func ClampWidth(terminalWidth int) int {
	return max(MinContentWidth, terminalWidth)
}

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7DC4E0")).
			MarginBottom(1)

	subTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#25d8ab")).
			MarginBottom(1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A6DA95"))

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A6DA95"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ED8796"))

	warningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#EED49F"))

	mutedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6E738D"))

	highlightStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#C6A0F6"))

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#5B6078")).
			Padding(1, 2)

	diffBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#5B6078")).
			Padding(0, 1)

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#C6A0F6")).
				Bold(true)

	normalItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#CAD3F5"))

	checkedPrefix = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A6DA95")).Render("☑")

	uncheckedPrefix = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6E738D")).Render("☐")

	buttonActiveStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#7DC4E0")).
				Foreground(lipgloss.Color("#1E2030")).
				Padding(0, 3).
				Bold(true)

	buttonInactiveStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#363A4F")).
				Foreground(lipgloss.Color("#CAD3F5")).
				Padding(0, 3)

	progressBarStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7DC4E0"))

	progressBgStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#363A4F"))
)
