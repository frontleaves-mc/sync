package tui

import (
	"github.com/charmbracelet/bubbletea"
	"github.com/frontleaves-mc/sync/internal/model"
)

// NextStepMsg 切换到下一步骤的消息。
type NextStepMsg struct{}

// NextStep 返回一个发送 NextStepMsg 的命令。
func NextStep() tea.Cmd {
	return func() tea.Msg { return NextStepMsg{} }
}

// CancelMsg 取消同步，退出程序。
type CancelMsg struct{}

// Cancel 返回一个发送 CancelMsg 的命令。
func Cancel() tea.Cmd {
	return func() tea.Msg { return CancelMsg{} }
}

// SyncDetailEnterMsg 从选择界面钻入详情（Client Mods / Resourcepacks）。
type SyncDetailEnterMsg struct {
	Kind model.SyncType
}

// SyncDetailDoneMsg 详情元数据获取完成。
type SyncDetailDoneMsg struct {
	Kind       model.SyncType
	DiffResult *model.DiffResult
	Err        error
}

// SyncDetailConfirmMsg 用户确认详情选择。
type SyncDetailConfirmMsg struct {
	Kind         model.SyncType
	SelectedDiff *model.DiffResult
}

// SyncDetailBackMsg 用户从详情返回。
type SyncDetailBackMsg struct {
	Kind model.SyncType
}
