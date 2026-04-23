package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/frontleaves-mc/sync/internal/model"
)

type syncOption struct {
	name     string
	desc     string
	syncType model.SyncType
	icon     string
	required bool
}

var syncOptions = []syncOption{
	{name: "Server Mods", desc: "服务端模组（必选）", syncType: model.SyncTypeModsServer, icon: "🖥️", required: true},
	{name: "Client Mods", desc: "客户端模组", syncType: model.SyncTypeModsClient, icon: "🎮"},
	{name: "Config 同步", desc: "配置文件", syncType: model.SyncTypeConfig, icon: "📄"},
}

// SelectMsg 用户完成选择后发送的消息，携带选中的同步类型。
type SelectMsg struct {
	SyncTypes []model.SyncType
}

type SelectModel struct {
	cursor  int
	checked map[int]bool
	width   int
}

func NewSelectModel() SelectModel {
	return SelectModel{
		cursor:  0,
		checked: map[int]bool{0: true},
	}
}

func (m SelectModel) Init() tea.Cmd {
	return nil
}

// GetSelectedTypes 返回当前选中的同步类型列表。
func (m SelectModel) GetSelectedTypes() []model.SyncType {
	var selected []model.SyncType
	for i, opt := range syncOptions {
		if m.checked[i] {
			selected = append(selected, opt.syncType)
		}
	}
	return selected
}

func (m SelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(syncOptions)-1 {
				m.cursor++
			}
		case " ":
			if m.cursor == 0 {
				return m, nil
			}
			m.checked[m.cursor] = !m.checked[m.cursor]
		case "right", "l":
			if m.cursor == 1 && m.checked[1] {
				return m, func() tea.Msg { return ClientModsDetailMsg{} }
			}
		case "enter":
			selected := m.GetSelectedTypes()
			if len(selected) == 0 {
				return m, nil
			}
			return m, func() tea.Msg {
				return SelectMsg{SyncTypes: selected}
			}
		}
	}
	return m, nil
}

func (m SelectModel) View() string {
	s := titleStyle.Render("  选择同步内容") + "\n"
	s += mutedStyle.Render("  空格勾选，→ 查看详情，回车确认") + "\n\n"

	checkedCount := 0
	for i, opt := range syncOptions {
		var prefix string
		style := normalItemStyle

		if opt.required {
			prefix = lockedPrefix + " "
			checkedCount++
		} else if m.checked[i] {
			prefix = checkedPrefix + " "
			checkedCount++
		} else {
			prefix = uncheckedPrefix + " "
		}

		if m.cursor == i {
			style = selectedItemStyle
		}

		line := fmt.Sprintf("  %s %s %s", prefix, opt.icon, style.Render(opt.name))
		descText := mutedStyle.Render("        " + opt.desc)

		if i == 1 && m.checked[1] {
			descText += mutedStyle.Render("  → 按右键查看详情")
		}

		s += line + descText + "\n"
	}

	s += "\n" + mutedStyle.Render(fmt.Sprintf("  已选择: %d 项", checkedCount)) + "\n"
	return lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).
		Render(lipgloss.NewStyle().MarginTop(2).Render(s))
}
