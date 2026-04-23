package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/frontleaves-mc/sync/internal/model"
)

type WelcomeModel struct {
	width int
}

func NewWelcomeModel() WelcomeModel {
	return WelcomeModel{}
}

func (m WelcomeModel) Init() tea.Cmd {
	return nil
}

func (m WelcomeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
	case tea.KeyMsg:
		return m, NextStep()
	}
	return m, nil
}

func (m WelcomeModel) View() string {
	logo1 := `
███████╗██████╗  ██████╗ ███╗   ██╗████████╗██╗     ███████╗ █████╗ ██╗   ██╗███████╗███████╗
██╔════╝██╔══██╗██╔═══██╗████╗  ██║╚══██╔══╝██║     ██╔════╝██╔══██╗██║   ██║██╔════╝██╔════╝
█████╗  ██████╔╝██║   ██║██╔██╗ ██║   ██║   ██║     █████╗  ███████║██║   ██║█████╗  ███████╗
██╔══╝  ██╔══██╗██║   ██║██║╚██╗██║   ██║   ██║     ██╔══╝  ██╔══██║╚██╗ ██╔╝██╔══╝  ╚════██║
██║     ██║  ██║╚██████╔╝██║ ╚████║   ██║   ███████╗███████╗██║  ██║ ╚████╔╝ ███████╗███████║
╚═╝     ╚═╝  ╚═╝ ╚═════╝ ╚═╝  ╚═══╝   ╚═╝   ╚══════╝╚══════╝╚═╝  ╚═╝  ╚═══╝  ╚══════╝╚══════╝`

	logo2 := `
   _____                 
  / ___/__  ______  _____
  \__ \/ / / / __ \/ ___/
 ___/ / /_/ / / / / /__  
/____/\__, /_/ /_/\___/  
     /____/              `

	content := lipgloss.NewStyle().
		Width(m.width).
		Align(lipgloss.Center).
		Render(
			titleStyle.Render(logo1) + "\n" +
				subTitleStyle.Render(logo2) + "\n" +
				subtitleStyle.Render(model.AppName) + "\n" +
				mutedStyle.Render(model.AppVersion) + "\n\n" +
				mutedStyle.Render("按任意键开始..."),
		)

	return lipgloss.NewStyle().
		MarginTop(2).
		Align(lipgloss.Center).
		Render(content)
}
