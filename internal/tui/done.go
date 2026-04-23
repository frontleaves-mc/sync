package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbletea"
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
		return s
	}

	if m.result.Downloaded > 0 {
		s += successStyle.Render(fmt.Sprintf("  ✅ 新增/更新: %d 个文件", m.result.Downloaded)) + "\n"
	}
	if m.result.Renamed > 0 {
		s += highlightStyle.Render(fmt.Sprintf("  📝 重命名: %d 个文件", m.result.Renamed)) + "\n"
	}
	if len(m.result.Failed) > 0 {
		s += errorStyle.Render(fmt.Sprintf("  ⚠️  失败: %d 个文件", len(m.result.Failed))) + "\n"
		lineWidth := max(20, ClampWidth(m.width)-4)
		for _, f := range m.result.Failed {
			// 截断过长的路径以适配终端宽度
			pathStr := f.Path
			reasonLen := len(f.Reason) + 3 // " — " separator
			maxPathLen := max(10, lineWidth-reasonLen)
			if len(pathStr) > maxPathLen {
				pathStr = "…" + pathStr[len(pathStr)-maxPathLen+1:]
			}
			s += errorStyle.Render(fmt.Sprintf("     %s — %s", pathStr, f.Reason)) + "\n"
		}
	}

	s += "\n" + mutedStyle.Render("  按任意键退出...") + "\n"
	return s
}
