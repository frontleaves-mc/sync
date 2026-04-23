package tui

import (
	"github.com/charmbracelet/bubbletea"
	"github.com/frontleaves-mc/sync/internal/model"
)

type CheckModel struct {
	result  model.CheckResult
	checked bool
	passed  bool
	width   int
}

func NewCheckModel() CheckModel {
	return CheckModel{}
}

func (m CheckModel) Init() tea.Cmd {
	return DoCheckCmd
}

func (m CheckModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
	case CheckDoneMsg:
		m.result = msg.Result
		m.checked = true
		m.passed = msg.Result.McDirFound
		return m, nil
	case tea.KeyMsg:
		if m.checked {
			if m.passed {
				return m, NextStep()
			}
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m CheckModel) View() string {
	if !m.checked {
		return "\n  检查运行环境...\n"
	}

	s := "\n  检查运行环境...\n\n"

	if m.result.McDirFound {
		s += successStyle.Render("  ✅ .minecraft/ 目录已找到") + "\n"
	} else {
		s += errorStyle.Render("  ❌ 未找到 .minecraft/ 目录") + "\n\n"
		s += errorStyle.Render("  请将本程序放置在与 .minecraft/ 同级的目录下运行。") + "\n\n"
		s += mutedStyle.Render("  按任意键退出...") + "\n"
		return s
	}

	if m.result.ModsDirOK {
		s += successStyle.Render("  ✅ mods/ 目录可用") + "\n"
	} else {
		s += warningStyle.Render("  ⚠️  mods/ 目录不存在（将在同步时创建）") + "\n"
	}

	if m.result.ConfigDirOK {
		s += successStyle.Render("  ✅ config/ 目录可用") + "\n"
	} else {
		s += warningStyle.Render("  ⚠️  config/ 目录不存在（将在同步时创建）") + "\n"
	}

	s += "\n" + mutedStyle.Render("  按任意键继续...") + "\n"
	return s
}

// CheckDoneMsg 环境检查完成消息。
type CheckDoneMsg struct {
	Result model.CheckResult
}

// DoCheckCmd 执行环境检查。
func DoCheckCmd() tea.Msg {
	return CheckDoneMsg{Result: model.CheckEnvironment()}
}
