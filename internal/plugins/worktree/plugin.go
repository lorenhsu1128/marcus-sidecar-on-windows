package worktree

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/marcus/sidecar/internal/mouse"
	"github.com/marcus/sidecar/internal/plugin"
)

const (
	pluginID   = "worktree-manager"
	pluginName = "worktrees"
	pluginIcon = "W"

	// Refresh interval for worktree list
	refreshInterval = 5 * time.Second

	// Output buffer capacity (lines)
	outputBufferCap = 500
)

// Plugin implements the worktree manager plugin.
type Plugin struct {
	// Required by plugin.Plugin interface
	ctx     *plugin.Context
	focused bool
	width   int
	height  int

	// Worktree state
	worktrees []*Worktree
	agents    map[string]*Agent

	// Session tracking for safe cleanup
	managedSessions map[string]bool

	// View state
	viewMode      ViewMode
	activePane    FocusPane
	previewTab    PreviewTab
	selectedIdx   int
	scrollOffset  int // Sidebar list scroll offset
	visibleCount  int // Number of visible list items
	previewOffset int
	sidebarWidth  int // Persisted sidebar width

	// Mouse support
	mouseHandler *mouse.Handler

	// Async state
	refreshing  bool
	lastRefresh time.Time

	// Diff state
	diffContent string
	diffRaw     string

	// Create modal state
	createName       string
	createBaseBranch string
	createTaskID     string
	createFocus      int // 0=name, 1=base, 2=task, 3=confirm
}

// New creates a new worktree manager plugin.
func New() *Plugin {
	return &Plugin{
		worktrees:       make([]*Worktree, 0),
		agents:          make(map[string]*Agent),
		managedSessions: make(map[string]bool),
		viewMode:        ViewModeList,
		activePane:      PaneSidebar,
		previewTab:      PreviewTabOutput,
		mouseHandler:    mouse.NewHandler(),
		sidebarWidth:    40, // Default 40% sidebar
	}
}

// ID returns the plugin identifier.
func (p *Plugin) ID() string { return pluginID }

// Name returns the plugin display name.
func (p *Plugin) Name() string { return pluginName }

// Icon returns the plugin icon.
func (p *Plugin) Icon() string { return pluginIcon }

// IsFocused returns whether the plugin is focused.
func (p *Plugin) IsFocused() bool { return p.focused }

// SetFocused sets the focus state.
func (p *Plugin) SetFocused(f bool) { p.focused = f }

// Init initializes the plugin with context.
func (p *Plugin) Init(ctx *plugin.Context) error {
	p.ctx = ctx

	// Register dynamic keybindings
	if ctx.Keymap != nil {
		// Sidebar list context
		ctx.Keymap.RegisterPluginBinding("n", "new-worktree", "worktree-list")
		ctx.Keymap.RegisterPluginBinding("enter", "select", "worktree-list")
		ctx.Keymap.RegisterPluginBinding("D", "delete-worktree", "worktree-list")
		ctx.Keymap.RegisterPluginBinding("p", "push", "worktree-list")
		ctx.Keymap.RegisterPluginBinding("d", "show-diff", "worktree-list")
		ctx.Keymap.RegisterPluginBinding("l", "focus-right", "worktree-list")
		ctx.Keymap.RegisterPluginBinding("right", "focus-right", "worktree-list")
		ctx.Keymap.RegisterPluginBinding("\\", "toggle-sidebar", "worktree-list")

		// Preview pane context
		ctx.Keymap.RegisterPluginBinding("h", "focus-left", "worktree-preview")
		ctx.Keymap.RegisterPluginBinding("left", "focus-left", "worktree-preview")
		ctx.Keymap.RegisterPluginBinding("esc", "focus-left", "worktree-preview")
		ctx.Keymap.RegisterPluginBinding("tab", "next-tab", "worktree-preview")
		ctx.Keymap.RegisterPluginBinding("shift+tab", "prev-tab", "worktree-preview")
		ctx.Keymap.RegisterPluginBinding("\\", "toggle-sidebar", "worktree-preview")

		// Create modal context
		ctx.Keymap.RegisterPluginBinding("esc", "cancel", "worktree-create")
		ctx.Keymap.RegisterPluginBinding("enter", "confirm", "worktree-create")
		ctx.Keymap.RegisterPluginBinding("tab", "next-field", "worktree-create")
		ctx.Keymap.RegisterPluginBinding("shift+tab", "prev-field", "worktree-create")
	}

	return nil
}

// Start begins async operations.
func (p *Plugin) Start() tea.Cmd {
	return p.refreshWorktrees()
}

// Stop cleans up plugin resources.
func (p *Plugin) Stop() {
	// Cleanup managed tmux sessions if needed
}

// Update handles messages.
func (p *Plugin) Update(msg tea.Msg) (plugin.Plugin, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case RefreshMsg:
		if !p.refreshing {
			p.refreshing = true
			cmds = append(cmds, p.refreshWorktrees())
		}

	case RefreshDoneMsg:
		p.refreshing = false
		p.lastRefresh = time.Now()
		if msg.Err == nil {
			p.worktrees = msg.Worktrees
			// Load stats for each worktree
			for _, wt := range p.worktrees {
				cmds = append(cmds, p.loadStats(wt.Path))
			}
		}

	case StatsLoadedMsg:
		for _, wt := range p.worktrees {
			if wt.Name == msg.WorktreeName {
				wt.Stats = msg.Stats
				break
			}
		}

	case DiffLoadedMsg:
		if p.selectedWorktree() != nil && p.selectedWorktree().Name == msg.WorktreeName {
			p.diffContent = msg.Content
			p.diffRaw = msg.Raw
		}

	case CreateDoneMsg:
		p.viewMode = ViewModeList
		if msg.Err == nil {
			p.worktrees = append(p.worktrees, msg.Worktree)
			p.selectedIdx = len(p.worktrees) - 1
		}
		p.clearCreateModal()

	case DeleteDoneMsg:
		if msg.Err == nil {
			p.removeWorktreeByName(msg.Name)
			if p.selectedIdx >= len(p.worktrees) && p.selectedIdx > 0 {
				p.selectedIdx--
			}
		}

	case PushDoneMsg:
		// Handle push result notification
		if msg.Err == nil {
			cmds = append(cmds, p.refreshWorktrees())
		}

	case tea.KeyMsg:
		cmd := p.handleKeyPress(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case tea.MouseMsg:
		cmd := p.handleMouse(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return p, tea.Batch(cmds...)
}

// selectedWorktree returns the currently selected worktree.
func (p *Plugin) selectedWorktree() *Worktree {
	if p.selectedIdx < 0 || p.selectedIdx >= len(p.worktrees) {
		return nil
	}
	return p.worktrees[p.selectedIdx]
}

// removeWorktreeByName removes a worktree from the list by name.
func (p *Plugin) removeWorktreeByName(name string) {
	for i, wt := range p.worktrees {
		if wt.Name == name {
			p.worktrees = append(p.worktrees[:i], p.worktrees[i+1:]...)
			return
		}
	}
}

// clearCreateModal resets create modal state.
func (p *Plugin) clearCreateModal() {
	p.createName = ""
	p.createBaseBranch = ""
	p.createTaskID = ""
	p.createFocus = 0
}

// handleKeyPress processes key input based on current view mode.
func (p *Plugin) handleKeyPress(msg tea.KeyMsg) tea.Cmd {
	switch p.viewMode {
	case ViewModeList:
		return p.handleListKeys(msg)
	case ViewModeCreate:
		return p.handleCreateKeys(msg)
	}
	return nil
}

// handleListKeys handles keys in list view.
func (p *Plugin) handleListKeys(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "j", "down":
		if p.activePane == PaneSidebar {
			p.moveCursor(1)
			return p.loadSelectedDiff()
		}
		p.previewOffset++
	case "k", "up":
		if p.activePane == PaneSidebar {
			p.moveCursor(-1)
			return p.loadSelectedDiff()
		}
		if p.previewOffset > 0 {
			p.previewOffset--
		}
	case "g":
		if p.activePane == PaneSidebar {
			p.selectedIdx = 0
			p.scrollOffset = 0
			return p.loadSelectedDiff()
		}
		p.previewOffset = 0
	case "G":
		if p.activePane == PaneSidebar {
			p.selectedIdx = len(p.worktrees) - 1
			p.ensureVisible()
			return p.loadSelectedDiff()
		}
	case "n":
		p.viewMode = ViewModeCreate
	case "D":
		return p.deleteSelected()
	case "p":
		return p.pushSelected()
	case "l", "right", "enter":
		if p.activePane == PaneSidebar {
			p.activePane = PanePreview
		}
	case "h", "left", "esc":
		if p.activePane == PanePreview {
			p.activePane = PaneSidebar
		}
	case "tab":
		p.cyclePreviewTab(1)
	case "shift+tab":
		p.cyclePreviewTab(-1)
	case "r":
		return func() tea.Msg { return RefreshMsg{} }
	}
	return nil
}

// handleCreateKeys handles keys in create modal.
func (p *Plugin) handleCreateKeys(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		p.viewMode = ViewModeList
		p.clearCreateModal()
	case "tab":
		p.createFocus = (p.createFocus + 1) % 4
	case "shift+tab":
		p.createFocus = (p.createFocus + 3) % 4
	case "enter":
		if p.createFocus == 3 {
			return p.createWorktree()
		}
		p.createFocus = (p.createFocus + 1) % 4
	case "backspace":
		switch p.createFocus {
		case 0:
			if len(p.createName) > 0 {
				p.createName = p.createName[:len(p.createName)-1]
			}
		case 1:
			if len(p.createBaseBranch) > 0 {
				p.createBaseBranch = p.createBaseBranch[:len(p.createBaseBranch)-1]
			}
		case 2:
			if len(p.createTaskID) > 0 {
				p.createTaskID = p.createTaskID[:len(p.createTaskID)-1]
			}
		}
	default:
		if len(msg.String()) == 1 {
			switch p.createFocus {
			case 0:
				p.createName += msg.String()
			case 1:
				p.createBaseBranch += msg.String()
			case 2:
				p.createTaskID += msg.String()
			}
		}
	}
	return nil
}

// moveCursor moves the selection cursor.
func (p *Plugin) moveCursor(delta int) {
	p.selectedIdx += delta
	if p.selectedIdx < 0 {
		p.selectedIdx = 0
	}
	if p.selectedIdx >= len(p.worktrees) {
		p.selectedIdx = len(p.worktrees) - 1
	}
	p.ensureVisible()
}

// ensureVisible adjusts scroll to keep selected item visible.
func (p *Plugin) ensureVisible() {
	if p.selectedIdx < p.scrollOffset {
		p.scrollOffset = p.selectedIdx
	}
	if p.visibleCount > 0 && p.selectedIdx >= p.scrollOffset+p.visibleCount {
		p.scrollOffset = p.selectedIdx - p.visibleCount + 1
	}
}

// cyclePreviewTab cycles through preview tabs.
func (p *Plugin) cyclePreviewTab(delta int) {
	p.previewTab = PreviewTab((int(p.previewTab) + delta + 3) % 3)
	p.previewOffset = 0
}

// handleMouse processes mouse input.
func (p *Plugin) handleMouse(msg tea.MouseMsg) tea.Cmd {
	// TODO: implement mouse handling
	return nil
}

// Commands returns the available commands.
func (p *Plugin) Commands() []plugin.Command {
	switch p.viewMode {
	case ViewModeCreate:
		return []plugin.Command{
			{ID: "cancel", Name: "Cancel", Description: "Cancel worktree creation", Context: "worktree-create", Priority: 1},
			{ID: "confirm", Name: "Create", Description: "Create the worktree", Context: "worktree-create", Priority: 2},
		}
	default:
		cmds := []plugin.Command{
			{ID: "new-worktree", Name: "New", Description: "Create new worktree", Context: "worktree-list", Priority: 1},
			{ID: "refresh", Name: "Refresh", Description: "Refresh worktree list", Context: "worktree-list", Priority: 2},
		}
		if p.selectedWorktree() != nil {
			cmds = append(cmds,
				plugin.Command{ID: "delete-worktree", Name: "Delete", Description: "Delete selected worktree", Context: "worktree-list", Priority: 3},
				plugin.Command{ID: "push", Name: "Push", Description: "Push branch to remote", Context: "worktree-list", Priority: 4},
			)
		}
		return cmds
	}
}

// FocusContext returns the current focus context for keybinding dispatch.
func (p *Plugin) FocusContext() string {
	switch p.viewMode {
	case ViewModeCreate:
		return "worktree-create"
	default:
		if p.activePane == PanePreview {
			return "worktree-preview"
		}
		return "worktree-list"
	}
}
