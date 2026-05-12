package tui

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/frontleaves-mc/sync/internal/model"
)

// MetadataFetcher 定义获取元数据和计算差异的接口。
type MetadataFetcher interface {
	GetModsMetadataWithMode(ctx context.Context, mode string) (*model.SyncMetadataResponse, error)
	GetConfigMetadata(ctx context.Context) (*model.SyncMetadataResponse, error)
	GetResourcepacksMetadata(ctx context.Context) (*model.SyncMetadataResponse, error)
	GetShaderpacksMetadata(ctx context.Context) (*model.SyncMetadataResponse, error)
	GetExtendsMetadata(ctx context.Context) (*model.SyncMetadataResponse, error)
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
	DiffServerMods    *model.DiffResult
	DiffClientMods    *model.DiffResult
	DiffCfg           *model.DiffResult
	DiffResourcepacks *model.DiffResult
	DiffShaderpacks   *model.DiffResult
	DiffExtends       *model.DiffResult
	Err               error
}

// DiffConfirmMsg 用户确认同步消息。
type DiffConfirmMsg struct{}

type DiffModel struct {
	phase     diffPhase
	spinner   spinner.Model
	syncTypes []model.SyncType

	fetcher MetadataFetcher

	diffServerMods    *model.DiffResult
	diffClientMods    *model.DiffResult
	diffCfg           *model.DiffResult
	diffResourcepacks *model.DiffResult
	diffShaderpacks   *model.DiffResult
	diffExtends       *model.DiffResult

	precomputedClientDiff    *model.DiffResult
	precomputedResourcepacks *model.DiffResult
	precomputedShaderpacks   *model.DiffResult

	err error

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

// SetPrecomputedClientDiff 设置从 Client Mods 详情界面传入的预筛选结果。
func (m DiffModel) SetPrecomputedClientDiff(diff *model.DiffResult) DiffModel {
	m.precomputedClientDiff = diff
	return m
}

// SetPrecomputedResourcepacks 设置从 Resourcepacks 详情界面传入的预筛选结果。
func (m DiffModel) SetPrecomputedResourcepacks(diff *model.DiffResult) DiffModel {
	m.precomputedResourcepacks = diff
	return m
}

// SetPrecomputedShaderpacks 设置从 Shaderpacks 详情界面传入的预筛选结果。
func (m DiffModel) SetPrecomputedShaderpacks(diff *model.DiffResult) DiffModel {
	m.precomputedShaderpacks = diff
	return m
}

func (m DiffModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.startFetch())
}

func (m DiffModel) startFetch() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		var diffServer, diffClient, diffCfg, diffRp, diffSh, diffExt *model.DiffResult

		for _, st := range m.syncTypes {
			switch st {
			case model.SyncTypeModsServer:
				resp, fetchErr := m.fetcher.GetModsMetadataWithMode(ctx, "server")
				if fetchErr != nil {
					return DiffDoneMsg{Err: fmt.Errorf("获取 server mods 元数据失败: %w", fetchErr)}
				}
				diffServer = m.fetcher.ComputeDiff(model.NormalizeModPaths(resp.Data.Files), model.SyncTypeModsServer)
			case model.SyncTypeModsClient:
				if m.precomputedClientDiff != nil {
					diffClient = m.precomputedClientDiff
				} else {
					resp, fetchErr := m.fetcher.GetModsMetadataWithMode(ctx, "client")
					if fetchErr != nil {
						return DiffDoneMsg{Err: fmt.Errorf("获取 client mods 元数据失败: %w", fetchErr)}
					}
					diffClient = m.fetcher.ComputeDiff(model.NormalizeModPaths(resp.Data.Files), model.SyncTypeModsClient)
				}
			case model.SyncTypeConfig:
				resp, fetchErr := m.fetcher.GetConfigMetadata(ctx)
				if fetchErr != nil {
					return DiffDoneMsg{Err: fmt.Errorf("获取 config 元数据失败: %w", fetchErr)}
				}
				diffCfg = m.fetcher.ComputeDiff(resp.Data.Files, model.SyncTypeConfig)
			case model.SyncTypeResourcepacks:
				if m.precomputedResourcepacks != nil {
					diffRp = m.precomputedResourcepacks
				} else {
					resp, fetchErr := m.fetcher.GetResourcepacksMetadata(ctx)
					if fetchErr != nil {
						return DiffDoneMsg{Err: fmt.Errorf("获取 resourcepacks 元数据失败: %w", fetchErr)}
					}
					diffRp = m.fetcher.ComputeDiff(resp.Data.Files, model.SyncTypeResourcepacks)
				}
			case model.SyncTypeShaderpacks:
				if m.precomputedShaderpacks != nil {
					diffSh = m.precomputedShaderpacks
				} else {
					resp, fetchErr := m.fetcher.GetShaderpacksMetadata(ctx)
					if fetchErr != nil {
						return DiffDoneMsg{Err: fmt.Errorf("获取 shaderpacks 元数据失败: %w", fetchErr)}
					}
					diffSh = m.fetcher.ComputeDiff(resp.Data.Files, model.SyncTypeShaderpacks)
				}
			case model.SyncTypeExtends:
				resp, fetchErr := m.fetcher.GetExtendsMetadata(ctx)
				if fetchErr != nil {
					return DiffDoneMsg{Err: fmt.Errorf("获取 extends 元数据失败: %w", fetchErr)}
				}
				diffExt = m.fetcher.ComputeDiff(model.NormalizeExtendsPaths(resp.Data.Files), model.SyncTypeExtends)
			}
		}

		return DiffDoneMsg{
			DiffServerMods:    diffServer,
			DiffClientMods:    diffClient,
			DiffCfg:           diffCfg,
			DiffResourcepacks: diffRp,
			DiffShaderpacks:   diffSh,
			DiffExtends:       diffExt,
		}
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
		m.diffServerMods = msg.DiffServerMods
		m.diffClientMods = msg.DiffClientMods
		m.diffCfg = msg.DiffCfg
		m.diffResourcepacks = msg.DiffResourcepacks
		m.diffShaderpacks = msg.DiffShaderpacks
		m.diffExtends = msg.DiffExtends
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
	vpWidth := max(20, m.width-4)
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
		s := lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).
			Render(fmt.Sprintf("\n  正在获取服务端文件列表... %s", m.spinner.View()))
		return lipgloss.NewStyle().MarginTop(2).Render(s)

	case diffPhaseError:
		s := lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).
			Render("\n" + errorStyle.Render(fmt.Sprintf("❌ %s", m.err)) + "\n" + mutedStyle.Render("按任意键返回..."))
		return lipgloss.NewStyle().MarginTop(2).Render(s)

	case diffPhasePreview:
		return m.renderPreview()
	}

	return ""
}

func (m DiffModel) renderPreview() string {
	s := titleStyle.Render("  同步预览") + "\n\n"

	_, _, _, _, _, totalUnchanged := m.countStats()

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

	return lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).
		Render(lipgloss.NewStyle().MarginTop(2).Render(s))
}

func (m DiffModel) countStats() (add, update, rename, del, fileCount, unchanged int) {
	for _, d := range []*model.DiffResult{m.diffServerMods, m.diffClientMods, m.diffCfg, m.diffResourcepacks, m.diffShaderpacks, m.diffExtends} {
		if d != nil {
			add += len(d.ToAdd)
			update += len(d.ToUpdate)
			rename += len(d.ToRename)
			del += len(d.ToDelete)
			unchanged += d.Unchanged
		}
	}
	return
}

// availableContentWidth 返回 viewport 内可用于文件条目显示的宽度。
func (m DiffModel) availableContentWidth() int {
	vpWidth := max(20, m.width-4)
	return vpWidth - 4
}

func (m DiffModel) buildContent() string {
	var sb strings.Builder

	totalAdd, totalUpdate, totalRename, totalDelete, _, _ := m.countStats()
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

	if totalDelete > 0 {
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(errorStyle.Render(fmt.Sprintf("🗑️ 删除文件 (%d)", totalDelete)) + "\n")
		for _, path := range m.collectDeletes() {
			truncated := path
			available := max(10, fileWidth-6)
			if len(truncated) > available {
				truncated = "…" + truncated[len(truncated)-available+1:]
			}
			sb.WriteString(mutedStyle.Render(fmt.Sprintf("   %s", truncated)) + "\n")
		}
	}

	if sb.Len() == 0 {
		sb.WriteString(mutedStyle.Render("所有文件已是最新，无需同步。"))
	}

	return sb.String()
}

func (m DiffModel) collectRenames() []model.RenameEntry {
	var entries []model.RenameEntry
	for _, d := range []*model.DiffResult{m.diffServerMods, m.diffClientMods, m.diffCfg, m.diffResourcepacks, m.diffShaderpacks, m.diffExtends} {
		if d != nil {
			entries = append(entries, d.ToRename...)
		}
	}
	return entries
}

func (m DiffModel) collectDeletes() []string {
	var paths []string
	for _, d := range []*model.DiffResult{m.diffServerMods, m.diffClientMods, m.diffCfg, m.diffResourcepacks, m.diffShaderpacks, m.diffExtends} {
		if d != nil {
			paths = append(paths, d.ToDelete...)
		}
	}
	return paths
}

func (m DiffModel) collectFiles(extract func(*model.DiffResult) []model.FileMetadata) []model.FileMetadata {
	var files []model.FileMetadata
	for _, d := range []*model.DiffResult{m.diffServerMods, m.diffClientMods, m.diffCfg, m.diffResourcepacks, m.diffShaderpacks, m.diffExtends} {
		if d != nil {
			files = append(files, extract(d)...)
		}
	}
	return files
}

func formatFileEntry(f model.FileMetadata, maxPathWidth int) string {
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
	for _, d := range []*model.DiffResult{m.diffServerMods, m.diffClientMods, m.diffCfg, m.diffResourcepacks, m.diffShaderpacks, m.diffExtends} {
		if d != nil {
			result.ToAdd = append(result.ToAdd, d.ToAdd...)
			result.ToUpdate = append(result.ToUpdate, d.ToUpdate...)
			result.ToRename = append(result.ToRename, d.ToRename...)
			result.ToDelete = append(result.ToDelete, d.ToDelete...)
			result.Unchanged += d.Unchanged
		}
	}
	return result
}
