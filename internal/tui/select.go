package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbletea"
	"github.com/frontleaves-mc/sync/internal/model"
)

type syncOption struct {
	name     string
	desc     string
	syncType model.SyncType
	icon     string
}

var syncOptions = []syncOption{
	{name: "Mods 同步", desc: "同步模组文件", syncType: model.SyncTypeMods, icon: "📦"},
	{name: "Config 同步", desc: "同步配置文件", syncType: model.SyncTypeConfig, icon: "📄"},
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
		checked: map[int]bool{0: true, 1: true},
	}
}

func (m SelectModel) Init() tea.Cmd {
	return nil
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
			m.checked[m.cursor] = !m.checked[m.cursor]
		case "enter":
			var selected []model.SyncType
			for i, opt := range syncOptions {
				if m.checked[i] {
					selected = append(selected, opt.syncType)
				}
			}
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
	s += mutedStyle.Render("  空格勾选，回车确认") + "\n\n"

	checkedCount := 0
	for i, opt := range syncOptions {
		prefix := uncheckedPrefix + " "
		style := normalItemStyle
		if m.checked[i] {
			prefix = checkedPrefix + " "
			checkedCount++
		}
		if m.cursor == i {
			style = selectedItemStyle
		}

		line := fmt.Sprintf("  %s %s %s", prefix, opt.icon, style.Render(opt.name))
		s += line + mutedStyle.Render("        "+opt.desc) + "\n"
	}

	s += "\n" + mutedStyle.Render(fmt.Sprintf("  已选择: %d 项", checkedCount)) + "\n"
	return s
}
