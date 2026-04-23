package tui

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/frontleaves-mc/sync/internal/model"
)

// DetailConfig 配置 SyncDetailModel 的行为。
type DetailConfig struct {
	Title       string
	SyncType    model.SyncType
	FetchFn     func(context.Context, MetadataFetcher) (*model.SyncMetadataResponse, error)
	NormalizeFn func([]model.FileMetadata) []model.FileMetadata
}

var (
	// ClientModsDetailConfig Client Mods 详情配置。
	ClientModsDetailConfig = DetailConfig{
		Title:    "Client Mods",
		SyncType: model.SyncTypeModsClient,
		FetchFn: func(ctx context.Context, f MetadataFetcher) (*model.SyncMetadataResponse, error) {
			return f.GetModsMetadataWithMode(ctx, "client")
		},
		NormalizeFn: model.NormalizeModPaths,
	}

	// ResourcepacksDetailConfig Resourcepacks 详情配置。
	ResourcepacksDetailConfig = DetailConfig{
		Title:    "Resourcepacks",
		SyncType: model.SyncTypeResourcepacks,
		FetchFn: func(ctx context.Context, f MetadataFetcher) (*model.SyncMetadataResponse, error) {
			return f.GetResourcepacksMetadata(ctx)
		},
	}
)

// clientDetailPhase 表示详情界面的内部阶段。
type clientDetailPhase int

const (
	clientPhaseLoading clientDetailPhase = iota
	clientPhaseList
	clientPhaseEmpty
	clientPhaseError
)

// clientModItem 扁平化的可操作项。
type clientModItem struct {
	kind   string // "add" | "update" | "rename"
	meta   model.FileMetadata
	rename model.RenameEntry
}

// SyncDetailModel 通用逐文件选择界面，支持 Client Mods 和 Resourcepacks。
type SyncDetailModel struct {
	config  DetailConfig
	phase   clientDetailPhase
	spinner spinner.Model
	fetcher MetadataFetcher

	diffResult *model.DiffResult
	items      []clientModItem
	cursor     int
	checked    map[int]bool

	width        int
	height       int
	scrollOffset int
	err          error
}

// NewSyncDetailModel 创建通用详情界面。
func NewSyncDetailModel(config DetailConfig, fetcher MetadataFetcher) SyncDetailModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = progressBarStyle

	return SyncDetailModel{
		config:  config,
		phase:   clientPhaseLoading,
		spinner: s,
		fetcher: fetcher,
		checked: make(map[int]bool),
	}
}

func (m SyncDetailModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.startFetch())
}

func (m SyncDetailModel) startFetch() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		resp, err := m.config.FetchFn(ctx, m.fetcher)
		if err != nil {
			return SyncDetailDoneMsg{Kind: m.config.SyncType, Err: fmt.Errorf("获取元数据失败: %w", err)}
		}
		files := resp.Data.Files
		if m.config.NormalizeFn != nil {
			files = m.config.NormalizeFn(files)
		}
		diff := m.fetcher.ComputeDiff(files, m.config.SyncType)
		return SyncDetailDoneMsg{Kind: m.config.SyncType, DiffResult: diff}
	}
}

func (m SyncDetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case spinner.TickMsg:
		if m.phase == clientPhaseLoading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case SyncDetailDoneMsg:
		if msg.Kind != m.config.SyncType {
			return m, nil
		}
		if msg.Err != nil {
			m.phase = clientPhaseError
			m.err = msg.Err
			return m, nil
		}
		m.diffResult = msg.DiffResult
		m.buildItems()
		if len(m.items) == 0 {
			m.phase = clientPhaseEmpty
			return m, nil
		}
		m.phase = clientPhaseList

	case tea.KeyMsg:
		switch m.phase {
		case clientPhaseEmpty, clientPhaseError:
			return m, func() tea.Msg { return SyncDetailBackMsg{Kind: m.config.SyncType} }

		case clientPhaseList:
			switch msg.String() {
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
					m.adjustScroll()
				}
			case "down", "j":
				if m.cursor < len(m.items)-1 {
					m.cursor++
					m.adjustScroll()
				}
			case " ":
				if m.cursor >= 0 && m.cursor < len(m.items) {
					m.checked[m.cursor] = !m.checked[m.cursor]
				}
			case "enter":
				selected := m.collectSelected()
				return m, func() tea.Msg {
					return SyncDetailConfirmMsg{Kind: m.config.SyncType, SelectedDiff: selected}
				}
			case "esc", "backspace":
				return m, func() tea.Msg { return SyncDetailBackMsg{Kind: m.config.SyncType} }
			}
		}
	}

	return m, nil
}

func (m *SyncDetailModel) buildItems() {
	m.items = nil
	if m.diffResult == nil {
		return
	}
	for _, f := range m.diffResult.ToAdd {
		m.items = append(m.items, clientModItem{kind: "add", meta: f})
	}
	for _, f := range m.diffResult.ToUpdate {
		m.items = append(m.items, clientModItem{kind: "update", meta: f})
	}
	for _, r := range m.diffResult.ToRename {
		m.items = append(m.items, clientModItem{kind: "rename", rename: r})
	}
	for i := range m.items {
		m.checked[i] = true
	}
	m.scrollOffset = 0
}

func (m SyncDetailModel) visibleLines() int {
	overhead := 6
	return max(1, m.height-overhead)
}

func (m *SyncDetailModel) adjustScroll() {
	visible := m.visibleLines()
	if m.cursor < m.scrollOffset {
		m.scrollOffset = m.cursor
	} else if m.cursor >= m.scrollOffset+visible {
		m.scrollOffset = m.cursor - visible + 1
	}
}

func (m SyncDetailModel) View() string {
	switch m.phase {
	case clientPhaseLoading:
		s := lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(
			fmt.Sprintf("\n  正在获取 %s 文件列表... %s", m.config.Title, m.spinner.View()))
		return lipgloss.NewStyle().MarginTop(2).Render(s)

	case clientPhaseError:
		s := lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(
			"\n" + errorStyle.Render(fmt.Sprintf("❌ %s", m.err)) + "\n" + mutedStyle.Render("按任意键返回..."))
		return lipgloss.NewStyle().MarginTop(2).Render(s)

	case clientPhaseEmpty:
		s := lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(
			"\n" + successStyle.Render(fmt.Sprintf("✅ 所有 %s 已是最新，无需同步。", m.config.Title)) + "\n" + mutedStyle.Render("按任意键返回..."))
		return lipgloss.NewStyle().MarginTop(2).Render(s)

	case clientPhaseList:
		return m.renderList()
	}
	return ""
}

func (m SyncDetailModel) renderList() string {
	var sb strings.Builder

	sb.WriteString(titleStyle.Render(fmt.Sprintf("  %s - 选择同步内容", m.config.Title)) + "\n")
	sb.WriteString(mutedStyle.Render("  ↑/↓ 移动，空格勾选，回车确认，Esc 返回") + "\n\n")

	content := m.buildFullContent()
	lines := strings.Split(content, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	visible := m.visibleLines()
	start := m.scrollOffset
	end := min(len(lines), start+visible)

	for i := start; i < end; i++ {
		sb.WriteString(lines[i] + "\n")
	}

	if end < len(lines) {
		sb.WriteString(mutedStyle.Render(fmt.Sprintf("  ... 还有 %d 项未显示", len(lines)-end)) + "\n")
	}

	checkedCount := 0
	for i := range m.items {
		if m.checked[i] {
			checkedCount++
		}
	}
	sb.WriteString("\n" + mutedStyle.Render(fmt.Sprintf("  已选择: %d/%d 项", checkedCount, len(m.items))) + "\n")

	content2 := sb.String()
	return lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).
		Render(lipgloss.NewStyle().MarginTop(2).Render(content2))
}

func (m SyncDetailModel) buildFullContent() string {
	var sb strings.Builder
	maxWidth := max(40, m.width-8)

	if m.diffResult == nil {
		return ""
	}

	idx := 0
	if len(m.diffResult.ToAdd) > 0 {
		sb.WriteString(successStyle.Render(fmt.Sprintf("  ✅ 新增文件 (%d)", len(m.diffResult.ToAdd))) + "\n")
		for _, f := range m.diffResult.ToAdd {
			sb.WriteString(m.renderItem(idx, f.Name, f.Size, maxWidth))
			idx++
		}
	}

	if len(m.diffResult.ToUpdate) > 0 {
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(warningStyle.Render(fmt.Sprintf("  🔄 更新文件 (%d)", len(m.diffResult.ToUpdate))) + "\n")
		for _, f := range m.diffResult.ToUpdate {
			sb.WriteString(m.renderItem(idx, f.Name, f.Size, maxWidth))
			idx++
		}
	}

	if len(m.diffResult.ToRename) > 0 {
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(highlightStyle.Render(fmt.Sprintf("  📝 重命名 (%d)", len(m.diffResult.ToRename))) + "\n")
		for _, r := range m.diffResult.ToRename {
			label := filepath.Base(r.OldPath) + " → " + filepath.Base(r.NewPath)
			sb.WriteString(m.renderItem(idx, label, 0, maxWidth))
			idx++
		}
	}

	return sb.String()
}

func (m SyncDetailModel) renderItem(idx int, name string, size int64, maxW int) string {
	prefix := uncheckedPrefix + " "
	if m.checked[idx] {
		prefix = checkedPrefix + " "
	}

	cursorMark := " "
	if m.cursor == idx {
		cursorMark = selectedItemStyle.Render("›")
	}

	availableForName := max(10, maxW-20)
	nameStr := name
	if len(nameStr) > availableForName {
		nameStr = "…" + nameStr[len(nameStr)-availableForName+1:]
	}

	line := fmt.Sprintf("  %s %s %s", cursorMark, prefix, nameStr)
	if size > 0 {
		line += "  " + mutedStyle.Render(model.FormatSize(size))
	}
	return line + "\n"
}

func (m SyncDetailModel) collectSelected() *model.DiffResult {
	result := &model.DiffResult{}
	if m.diffResult != nil {
		result.Unchanged = m.diffResult.Unchanged
	}
	for i, item := range m.items {
		if !m.checked[i] {
			continue
		}
		switch item.kind {
		case "add":
			result.ToAdd = append(result.ToAdd, item.meta)
		case "update":
			result.ToUpdate = append(result.ToUpdate, item.meta)
		case "rename":
			result.ToRename = append(result.ToRename, item.rename)
		}
	}
	return result
}
