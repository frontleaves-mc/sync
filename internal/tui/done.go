package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/frontleaves-mc/sync/internal/model"
)

type DoneModel struct {
	result *model.SyncResult
	width  int
	height int
}

func NewDoneModel(result *model.SyncResult) DoneModel {
	return DoneModel{result: result}
}

func (m DoneModel) Init() tea.Cmd {
	return nil
}

func (m DoneModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		return m, tea.Quit
	}
	return m, nil
}

func (m DoneModel) View() string {
	s := "\n" + titleStyle.Render("  同步完成！") + "\n\n"

	if m.result == nil {
		s += warningStyle.Render("  ⚠️  无同步结果") + "\n"
		s += "\n" + mutedStyle.Render("  按任意键退出...") + "\n"
		return lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).
			Render(lipgloss.NewStyle().MarginTop(2).Render(s))
	}

	if m.result.Downloaded > 0 {
		s += successStyle.Render(fmt.Sprintf("  ✅ 新增/更新: %d 个文件", m.result.Downloaded)) + "\n"
	}
	if m.result.Renamed > 0 {
		s += highlightStyle.Render(fmt.Sprintf("  📝 重命名: %d 个文件", m.result.Renamed)) + "\n"
	}
	if m.result.Deleted > 0 {
		s += errorStyle.Render(fmt.Sprintf("  🗑️ 删除: %d 个文件", m.result.Deleted)) + "\n"
	}
	if len(m.result.Failed) > 0 {
		s += errorStyle.Render(fmt.Sprintf("  ⚠️  失败: %d 个文件", len(m.result.Failed))) + "\n"
		lineWidth := max(20, ClampWidth(m.width)-4)
		for _, f := range m.result.Failed {
			pathStr := f.Path
			reasonLen := len(f.Reason) + 3
			maxPathLen := max(10, lineWidth-reasonLen)
			if len(pathStr) > maxPathLen {
				pathStr = "…" + pathStr[len(pathStr)-maxPathLen+1:]
			}
			s += errorStyle.Render(fmt.Sprintf("     %s — %s", pathStr, f.Reason)) + "\n"
		}
	}

	s += "\n" + mutedStyle.Render("  按任意键退出...") + "\n"
	return lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).
		Render(lipgloss.NewStyle().MarginTop(2).Render(s))
}
