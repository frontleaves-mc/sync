package tui

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/frontleaves-mc/sync/internal/model"
)

// MetadataFetcher 定义获取元数据和计算差异的接口。
type MetadataFetcher interface {
	GetModsMetadata(ctx context.Context) (*model.SyncMetadataResponse, error)
	GetConfigMetadata(ctx context.Context) (*model.SyncMetadataResponse, error)
	ComputeDiff(remote []model.FileMetadata, syncType model.SyncType) *model.DiffResult
}

// diffPhase 表示 diff 界面的内部阶段。
type diffPhase int

const (
	diffPhaseLoading diffPhase = iota
	diffPhasePreview
	diffPhaseError
)

const (
	maxDiffVPHeight = 15
	minDiffVPHeight = 3
	diffVPOverhead  = 9
)

// DiffDoneMsg 差异计算完成消息。
type DiffDoneMsg struct {
	DiffMods *model.DiffResult
	DiffCfg  *model.DiffResult
	Err      error
}

// DiffConfirmMsg 用户确认同步消息。
type DiffConfirmMsg struct{}

type DiffModel struct {
	phase        diffPhase
	spinner      spinner.Model
	syncTypes    []model.SyncType

	fetcher MetadataFetcher

	diffMods *model.DiffResult
	diffCfg  *model.DiffResult
	err      error

	focusConfirm bool
	width        int
	height       int
	viewport     viewport.Model
	vpReady      bool
}

func NewDiffModel(fetcher MetadataFetcher) DiffModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = progressBarStyle

	return DiffModel{
		phase:        diffPhaseLoading,
		spinner:      s,
		fetcher:      fetcher,
		focusConfirm: true,
		viewport:     viewport.New(60, 15),
		vpReady:      true,
	}
}

func (m DiffModel) SetSyncTypes(types []model.SyncType) DiffModel {
	m.syncTypes = types
	return m
}

func (m DiffModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.startFetch())
}

func (m DiffModel) startFetch() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		var diffMods, diffCfg *model.DiffResult

		for _, st := range m.syncTypes {
			switch st {
			case model.SyncTypeMods:
				resp, fetchErr := m.fetcher.GetModsMetadata(ctx)
				if fetchErr != nil {
					return DiffDoneMsg{Err: fmt.Errorf("获取 mods 元数据失败: %w", fetchErr)}
				}
				diffMods = m.fetcher.ComputeDiff(resp.Data.Files, model.SyncTypeMods)
			case model.SyncTypeConfig:
				resp, fetchErr := m.fetcher.GetConfigMetadata(ctx)
				if fetchErr != nil {
					return DiffDoneMsg{Err: fmt.Errorf("获取 config 元数据失败: %w", fetchErr)}
				}
				diffCfg = m.fetcher.ComputeDiff(resp.Data.Files, model.SyncTypeConfig)
			}
		}

		return DiffDoneMsg{DiffMods: diffMods, DiffCfg: diffCfg}
	}
}

func (m DiffModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.initViewport()
		if m.phase == diffPhasePreview {
			m.viewport.SetContent(m.buildContent())
		}

	case spinner.TickMsg:
		if m.phase == diffPhaseLoading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case DiffDoneMsg:
		if msg.Err != nil {
			m.phase = diffPhaseError
			m.err = msg.Err
			return m, nil
		}
		m.diffMods = msg.DiffMods
		m.diffCfg = msg.DiffCfg
		m.phase = diffPhasePreview
		m.viewport.SetContent(m.buildContent())
		m.viewport.GotoTop()
		return m, nil

	case tea.KeyMsg:
		if m.phase == diffPhaseError {
			return m, Cancel()
		}
		if m.phase == diffPhasePreview {
			var vpCmd tea.Cmd
			m.viewport, vpCmd = m.viewport.Update(msg)

			switch msg.String() {
			case "tab", "left", "right":
				m.focusConfirm = !m.focusConfirm
			case "enter":
				if m.focusConfirm {
					return m, tea.Batch(vpCmd, func() tea.Msg { return DiffConfirmMsg{} })
				}
				return m, tea.Batch(vpCmd, Cancel())
			}
			return m, vpCmd
		}
	}

	return m, nil
}

func (m *DiffModel) initViewport() {
	vpWidth := max(20, m.width-8)
	vpHeight := min(maxDiffVPHeight, max(minDiffVPHeight, m.height-diffVPOverhead))

	if !m.vpReady {
		m.viewport = viewport.New(vpWidth, vpHeight)
		m.vpReady = true
	} else {
		m.viewport.Width = vpWidth
		m.viewport.Height = vpHeight
	}
}

func (m DiffModel) View() string {
	switch m.phase {
	case diffPhaseLoading:
		return fmt.Sprintf("\n  正在获取服务端文件列表... %s", m.spinner.View())

	case diffPhaseError:
		return "\n" + errorStyle.Render(fmt.Sprintf("  ❌ %s", m.err)) + "\n\n" + mutedStyle.Render("  按任意键返回...")

	case diffPhasePreview:
		return m.renderPreview()
	}

	return ""
}

func (m DiffModel) renderPreview() string {
	s := titleStyle.Render("  同步预览") + "\n\n"

	_, _, _, totalUnchanged := m.countStats()

	if m.vpReady {
		s += diffBoxStyle.Render(m.viewport.View()) + "\n"

		if m.viewport.TotalLineCount() > m.viewport.Height {
			s += mutedStyle.Render("  ↑/↓ 滚动查看更多") + "\n"
		}
	} else {
		s += diffBoxStyle.Render(m.buildContent()) + "\n"
	}

	s += "\n" + mutedStyle.Render(fmt.Sprintf("  ⏭️  未变更: %d 个文件", totalUnchanged)) + "\n"

	s += "\n  "
	if m.focusConfirm {
		s += buttonActiveStyle.Render("确认同步") + "  " + buttonInactiveStyle.Render("取消")
	} else {
		s += buttonInactiveStyle.Render("确认同步") + "  " + buttonActiveStyle.Render("取消")
	}
	s += "\n"

	return s
}

func (m DiffModel) countStats() (add, update, rename, unchanged int) {
	if m.diffMods != nil {
		add += len(m.diffMods.ToAdd)
		update += len(m.diffMods.ToUpdate)
		rename += len(m.diffMods.ToRename)
		unchanged += m.diffMods.Unchanged
	}
	if m.diffCfg != nil {
		add += len(m.diffCfg.ToAdd)
		update += len(m.diffCfg.ToUpdate)
		rename += len(m.diffCfg.ToRename)
		unchanged += m.diffCfg.Unchanged
	}
	return
}

// availableContentWidth 返回 viewport 内可用于文件条目显示的宽度。
func (m DiffModel) availableContentWidth() int {
	vpWidth := max(20, m.width-8)
	// 减去 viewport 的 border + padding（diffBoxStyle: Padding(0,1) + RoundedBorder = 4）
	return vpWidth - 4
}

func (m DiffModel) buildContent() string {
	var sb strings.Builder

	totalAdd, totalUpdate, totalRename, _ := m.countStats()
	fileWidth := m.availableContentWidth()

	if totalRename > 0 {
		sb.WriteString(highlightStyle.Render(fmt.Sprintf("📝 重命名文件 (%d)", totalRename)) + "\n")
		for _, entry := range m.collectRenames() {
			sb.WriteString(mutedStyle.Render(fmt.Sprintf("   %s → %s", filepath.Base(entry.OldPath), filepath.Base(entry.NewPath))) + "\n")
		}
	}

	if totalAdd > 0 {
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(successStyle.Render(fmt.Sprintf("✅ 新增文件 (%d)", totalAdd)) + "\n")
		for _, f := range m.collectFiles(func(d *model.DiffResult) []model.FileMetadata { return d.ToAdd }) {
			sb.WriteString(formatFileEntry(f, fileWidth))
		}
	}

	if totalUpdate > 0 {
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(warningStyle.Render(fmt.Sprintf("🔄 更新文件 (%d)", totalUpdate)) + "\n")
		for _, f := range m.collectFiles(func(d *model.DiffResult) []model.FileMetadata { return d.ToUpdate }) {
			sb.WriteString(formatFileEntry(f, fileWidth))
		}
	}

	if sb.Len() == 0 {
		sb.WriteString(mutedStyle.Render("所有文件已是最新，无需同步。"))
	}

	return sb.String()
}

func (m DiffModel) collectRenames() []model.RenameEntry {
	var entries []model.RenameEntry
	if m.diffMods != nil {
		entries = append(entries, m.diffMods.ToRename...)
	}
	if m.diffCfg != nil {
		entries = append(entries, m.diffCfg.ToRename...)
	}
	return entries
}

func (m DiffModel) collectFiles(extract func(*model.DiffResult) []model.FileMetadata) []model.FileMetadata {
	var files []model.FileMetadata
	if m.diffMods != nil {
		files = append(files, extract(m.diffMods)...)
	}
	if m.diffCfg != nil {
		files = append(files, extract(m.diffCfg)...)
	}
	return files
}

func formatFileEntry(f model.FileMetadata, maxPathWidth int) string {
	// 预留缩进 "   "（3字符）和大小信息的空间
	availableForPath := max(10, maxPathWidth-10)
	pathStr := f.Path
	if len(pathStr) > availableForPath {
		pathStr = "…" + pathStr[len(pathStr)-availableForPath+1:]
	}
	s := mutedStyle.Render(fmt.Sprintf("   %s", pathStr))
	if f.Size > 0 {
		s += "  " + mutedStyle.Render(model.FormatSize(f.Size))
	}
	return s + "\n"
}

// GetDiffResult 返回合并后的差异结果。
func (m DiffModel) GetDiffResult() *model.DiffResult {
	result := &model.DiffResult{}
	if m.diffMods != nil {
		result.ToAdd = append(result.ToAdd, m.diffMods.ToAdd...)
		result.ToUpdate = append(result.ToUpdate, m.diffMods.ToUpdate...)
		result.ToRename = append(result.ToRename, m.diffMods.ToRename...)
		result.Unchanged += m.diffMods.Unchanged
	}
	if m.diffCfg != nil {
		result.ToAdd = append(result.ToAdd, m.diffCfg.ToAdd...)
		result.ToUpdate = append(result.ToUpdate, m.diffCfg.ToUpdate...)
		result.ToRename = append(result.ToRename, m.diffCfg.ToRename...)
		result.Unchanged += m.diffCfg.Unchanged
	}
	return result
}
