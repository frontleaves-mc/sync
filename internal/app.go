package internal

import (
	"context"

	"github.com/charmbracelet/bubbletea"
	"github.com/frontleaves-mc/sync/internal/model"
	"github.com/frontleaves-mc/sync/internal/tui"
)

// step 主状态机的步骤枚举。
type step int

const (
	stepWelcome step = iota
	stepCheck
	stepSelect
	stepClientModsDetail
	stepDiff
	stepSync
	stepDone
)

// AppModel 主状态机，管理所有步骤的切换。
type AppModel struct {
	currentStep step

	welcome          tui.WelcomeModel
	check            tui.CheckModel
	select_          tui.SelectModel
	clientModsDetail tui.ClientModsDetailModel
	diff             tui.DiffModel
	progress         tui.ProgressModel
	done             tui.DoneModel

	client  *SyncClient
	engine  *SyncEngine
	fetcher *metadataFetcherImpl
	width   int
	height  int
}

// metadataFetcherImpl 组合 SyncClient 和 SyncEngine 实现 MetadataFetcher 接口。
type metadataFetcherImpl struct {
	client *SyncClient
	engine *SyncEngine
}

func (f *metadataFetcherImpl) GetModsMetadataWithMode(ctx context.Context, mode string) (*model.SyncMetadataResponse, error) {
	return f.client.GetModsMetadataWithMode(ctx, mode)
}

func (f *metadataFetcherImpl) GetConfigMetadata(ctx context.Context) (*model.SyncMetadataResponse, error) {
	return f.client.GetConfigMetadata(ctx)
}

func (f *metadataFetcherImpl) ComputeDiff(remote []model.FileMetadata, syncType model.SyncType) *model.DiffResult {
	return f.engine.ComputeDiff(remote, syncType)
}

// NewAppModel 创建应用主模型。
func NewAppModel() AppModel {
	client := NewSyncClient()
	engine := NewSyncEngine(client)
	fetcher := &metadataFetcherImpl{client: client, engine: engine}

	return AppModel{
		currentStep: stepWelcome,
		welcome:     tui.NewWelcomeModel(),
		check:       tui.NewCheckModel(),
		select_:     tui.NewSelectModel(),
		diff:        tui.NewDiffModel(fetcher),
		client:      client,
		engine:      engine,
		fetcher:     fetcher,
	}
}

func (m AppModel) Init() tea.Cmd {
	return m.welcome.Init()
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}

	switch m.currentStep {
	case stepWelcome:
		return m.updateWelcome(msg)
	case stepCheck:
		return m.updateCheck(msg)
	case stepSelect:
		return m.updateSelect(msg)
	case stepClientModsDetail:
		return m.updateClientModsDetail(msg)
	case stepDiff:
		return m.updateDiff(msg)
	case stepSync:
		return m.updateProgress(msg)
	case stepDone:
		return m.updateDone(msg)
	}

	return m, nil
}

func (m AppModel) updateWelcome(msg tea.Msg) (tea.Model, tea.Cmd) {
	model, cmd := m.welcome.Update(msg)
	m.welcome = model.(tui.WelcomeModel)

	if _, ok := msg.(tui.NextStepMsg); ok {
		m.currentStep = stepCheck
		return m, m.check.Init()
	}

	return m, cmd
}

func (m AppModel) updateCheck(msg tea.Msg) (tea.Model, tea.Cmd) {
	model, cmd := m.check.Update(msg)
	m.check = model.(tui.CheckModel)

	if _, ok := msg.(tui.NextStepMsg); ok {
		m.currentStep = stepSelect
		return m, m.select_.Init()
	}

	return m, cmd
}

func (m AppModel) updateSelect(msg tea.Msg) (tea.Model, tea.Cmd) {
	model, cmd := m.select_.Update(msg)
	m.select_ = model.(tui.SelectModel)

	if _, ok := msg.(tui.ClientModsDetailMsg); ok {
		m.currentStep = stepClientModsDetail
		m.clientModsDetail = tui.NewClientModsDetailModel(m.fetcher)
		return m, m.clientModsDetail.Init()
	}

	if sel, ok := msg.(tui.SelectMsg); ok {
		m.currentStep = stepDiff
		m.diff = tui.NewDiffModel(m.fetcher).SetSyncTypes(sel.SyncTypes)
		return m, m.diff.Init()
	}

	return m, cmd
}

func (m AppModel) updateClientModsDetail(msg tea.Msg) (tea.Model, tea.Cmd) {
	model, cmd := m.clientModsDetail.Update(msg)
	m.clientModsDetail = model.(tui.ClientModsDetailModel)

	switch msg.(type) {
	case tui.ClientModsDetailBackMsg:
		m.currentStep = stepSelect
		return m, nil

	case tui.ClientModsDetailConfirmMsg:
		confirmMsg := msg.(tui.ClientModsDetailConfirmMsg)
		m.currentStep = stepDiff
		selectedTypes := m.select_.GetSelectedTypes()
		m.diff = tui.NewDiffModel(m.fetcher).
			SetSyncTypes(selectedTypes).
			SetPrecomputedClientDiff(confirmMsg.SelectedDiff)
		return m, m.diff.Init()
	}

	return m, cmd
}

func (m AppModel) updateDiff(msg tea.Msg) (tea.Model, tea.Cmd) {
	model, cmd := m.diff.Update(msg)
	m.diff = model.(tui.DiffModel)

	if _, ok := msg.(tui.DiffConfirmMsg); ok {
		m.currentStep = stepSync
		diffResult := m.diff.GetDiffResult()
		m.progress = tui.NewProgressModel(m.engine, diffResult)
		return m, m.progress.Init()
	}

	if _, ok := msg.(tui.CancelMsg); ok {
		return m, tea.Quit
	}

	return m, cmd
}

func (m AppModel) updateProgress(msg tea.Msg) (tea.Model, tea.Cmd) {
	model, cmd := m.progress.Update(msg)
	m.progress = model.(tui.ProgressModel)

	if _, ok := msg.(tui.NextStepMsg); ok {
		m.currentStep = stepDone
		m.done = tui.NewDoneModel(m.progress.GetResult())
		return m, m.done.Init()
	}

	return m, cmd
}

func (m AppModel) updateDone(msg tea.Msg) (tea.Model, tea.Cmd) {
	model, cmd := m.done.Update(msg)
	m.done = model.(tui.DoneModel)
	return m, cmd
}

func (m AppModel) View() string {
	switch m.currentStep {
	case stepWelcome:
		return m.welcome.View()
	case stepCheck:
		return m.check.View()
	case stepSelect:
		return m.select_.View()
	case stepClientModsDetail:
		return m.clientModsDetail.View()
	case stepDiff:
		return m.diff.View()
	case stepSync:
		return m.progress.View()
	case stepDone:
		return m.done.View()
	}
	return ""
}
