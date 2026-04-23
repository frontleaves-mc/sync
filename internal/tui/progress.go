package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/frontleaves-mc/sync/internal/model"
)

// SyncRunner 定义执行同步的接口。
type SyncRunner interface {
	ExecuteSync(ctx context.Context, diff *model.DiffResult) *model.SyncResult
}

// SyncCompleteMsg 同步完成消息。
type SyncCompleteMsg struct {
	Result *model.SyncResult
}

type ProgressModel struct {
	runner  SyncRunner
	diff    *model.DiffResult
	current int
	total   int
	result  *model.SyncResult
	width   int
}

func NewProgressModel(runner SyncRunner, diff *model.DiffResult) ProgressModel {
	total := 0
	if diff != nil {
		total = len(diff.ToAdd) + len(diff.ToUpdate) + len(diff.ToRename)
	}
	return ProgressModel{
		runner: runner,
		diff:   diff,
		total:  total,
	}
}

func (m ProgressModel) Init() tea.Cmd {
	return m.startSync()
}

func (m ProgressModel) startSync() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		result := m.runner.ExecuteSync(ctx, m.diff)
		return SyncCompleteMsg{Result: result}
	}
}

func (m ProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
	case SyncCompleteMsg:
		m.result = msg.Result
		return m, func() tea.Msg { return NextStepMsg{} }
	}
	return m, nil
}

func (m ProgressModel) View() string {
	if m.result != nil {
		return ""
	}
	return m.renderProgress()
}

func (m ProgressModel) renderProgress() string {
	s := "\n" + titleStyle.Render("  正在同步...") + "\n\n"

	barWidth := max(10, ClampWidth(m.width)-6)
	bar := strings.Repeat("█", barWidth)
	s += fmt.Sprintf("  %s\n", progressBarStyle.Render(bar))
	s += mutedStyle.Render(fmt.Sprintf("  共 %d 个文件待处理...", m.total)) + "\n"

	return lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).
		Render(lipgloss.NewStyle().MarginTop(2).Render(s))
}

// GetResult 返回同步结果。
func (m ProgressModel) GetResult() *model.SyncResult {
	return m.result
}
