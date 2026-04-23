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

// QuitMsg 退出程序消息。
type QuitMsg struct{}

// ClientModsDetailMsg 从选择界面钻入 Client Mods 详情。
type ClientModsDetailMsg struct{}

// ClientModsDetailDoneMsg Client Mods 元数据获取完成。
type ClientModsDetailDoneMsg struct {
	DiffResult *model.DiffResult
	Err        error
}

// ClientModsDetailConfirmMsg 用户确认 Client Mods 选择。
type ClientModsDetailConfirmMsg struct {
	SelectedDiff *model.DiffResult
}

// ClientModsDetailBackMsg 用户从 Client Mods 详情返回。
type ClientModsDetailBackMsg struct{}
