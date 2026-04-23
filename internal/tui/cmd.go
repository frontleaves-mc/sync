package tui

import "github.com/charmbracelet/bubbletea"

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
