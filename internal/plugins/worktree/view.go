package worktree

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Sidebar styles
	sidebarStyle = lipgloss.NewStyle().
			BorderRight(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240"))

	selectedStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("255"))

	normalStyle = lipgloss.NewStyle()

	// Status colors
	statusActiveColor  = lipgloss.Color("42")  // Green
	statusWaitingColor = lipgloss.Color("214") // Yellow/orange
	statusDoneColor    = lipgloss.Color("42")  // Green
	statusErrorColor   = lipgloss.Color("196") // Red
	statusPausedColor  = lipgloss.Color("240") // Gray

	// Preview pane styles
	previewHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("255")).
				Background(lipgloss.Color("236"))

	tabActiveStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("255")).
			Background(lipgloss.Color("62"))

	tabInactiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240"))

	// Modal styles
	modalStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2)

	inputStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1)

	inputFocusedStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("62")).
				Padding(0, 1)

	buttonStyle = lipgloss.NewStyle().
			Padding(0, 2).
			Background(lipgloss.Color("240")).
			Foreground(lipgloss.Color("255"))

	buttonFocusedStyle = lipgloss.NewStyle().
				Padding(0, 2).
				Background(lipgloss.Color("62")).
				Foreground(lipgloss.Color("255"))
)

// View renders the plugin UI.
func (p *Plugin) View(width, height int) string {
	p.width = width
	p.height = height

	switch p.viewMode {
	case ViewModeCreate:
		return p.renderCreateModal(width, height)
	default:
		return p.renderListView(width, height)
	}
}

// renderListView renders the main split-pane list view.
func (p *Plugin) renderListView(width, height int) string {
	// Calculate pane widths (40% sidebar, 60% preview)
	sidebarW := (width * p.sidebarWidth) / 100
	if sidebarW < 25 {
		sidebarW = 25
	}
	previewW := width - sidebarW - 1 // -1 for border

	// Render each pane
	sidebar := p.renderSidebar(sidebarW, height)
	preview := p.renderPreview(previewW, height)

	// Join horizontally
	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, preview)
}

// renderSidebar renders the worktree list sidebar.
func (p *Plugin) renderSidebar(width, height int) string {
	var lines []string

	// Header
	header := "Worktrees"
	if p.activePane == PaneSidebar {
		header = selectedStyle.Render(header)
	}
	lines = append(lines, header)
	lines = append(lines, strings.Repeat("─", width-2))

	// Calculate visible items
	contentHeight := height - 2 // header + separator
	p.visibleCount = contentHeight

	// Render worktree items
	if len(p.worktrees) == 0 {
		lines = append(lines, normalStyle.Foreground(lipgloss.Color("240")).Render("  No worktrees"))
		lines = append(lines, normalStyle.Foreground(lipgloss.Color("240")).Render("  Press 'n' to create one"))
	} else {
		for i := p.scrollOffset; i < len(p.worktrees) && i < p.scrollOffset+p.visibleCount; i++ {
			wt := p.worktrees[i]
			line := p.renderWorktreeItem(wt, i == p.selectedIdx, width-2)
			lines = append(lines, line)
		}
	}

	// Pad to fill height
	for len(lines) < height {
		lines = append(lines, "")
	}

	content := strings.Join(lines[:height], "\n")
	return sidebarStyle.Width(width).Height(height).Render(content)
}

// renderWorktreeItem renders a single worktree list item.
func (p *Plugin) renderWorktreeItem(wt *Worktree, selected bool, width int) string {
	// Status indicator
	statusIcon := wt.Status.Icon()
	statusColor := statusPausedColor
	switch wt.Status {
	case StatusActive:
		statusColor = statusActiveColor
	case StatusWaiting:
		statusColor = statusWaitingColor
	case StatusDone:
		statusColor = statusDoneColor
	case StatusError:
		statusColor = statusErrorColor
	}

	icon := lipgloss.NewStyle().Foreground(statusColor).Render(statusIcon)

	// Name and time
	name := wt.Name
	timeStr := formatRelativeTime(wt.UpdatedAt)

	// Stats if available
	statsStr := ""
	if wt.Stats != nil && (wt.Stats.Additions > 0 || wt.Stats.Deletions > 0) {
		statsStr = fmt.Sprintf("+%d -%d", wt.Stats.Additions, wt.Stats.Deletions)
	}

	// First line: icon, name, time
	line1 := fmt.Sprintf(" %s %s", icon, name)
	if len(line1) < width-len(timeStr)-2 {
		line1 = line1 + strings.Repeat(" ", width-len(line1)-len(timeStr)-1) + timeStr
	}

	// Second line: agent type, task ID, stats
	var parts []string
	if wt.Agent != nil {
		parts = append(parts, string(wt.Agent.Type))
	} else {
		parts = append(parts, "—")
	}
	if wt.TaskID != "" {
		parts = append(parts, wt.TaskID)
	}
	if statsStr != "" {
		parts = append(parts, statsStr)
	}
	line2 := "   " + strings.Join(parts, "  ")

	// Combine lines
	content := line1 + "\n" + line2

	if selected && p.activePane == PaneSidebar {
		return selectedStyle.Width(width).Render(content)
	}
	return normalStyle.Width(width).Render(content)
}

// renderPreview renders the preview pane.
func (p *Plugin) renderPreview(width, height int) string {
	var lines []string

	// Tab header
	tabs := p.renderTabs(width)
	lines = append(lines, tabs)
	lines = append(lines, strings.Repeat("─", width))

	contentHeight := height - 2

	// Render content based on active tab
	var content string
	switch p.previewTab {
	case PreviewTabOutput:
		content = p.renderOutputContent(width, contentHeight)
	case PreviewTabDiff:
		content = p.renderDiffContent(width, contentHeight)
	case PreviewTabTask:
		content = p.renderTaskContent(width, contentHeight)
	}

	lines = append(lines, content)

	// Pad to fill height
	result := strings.Join(lines, "\n")
	return lipgloss.NewStyle().Width(width).Height(height).Render(result)
}

// renderTabs renders the preview pane tab header.
func (p *Plugin) renderTabs(width int) string {
	tabs := []string{"Output", "Diff", "Task"}
	var rendered []string

	for i, tab := range tabs {
		style := tabInactiveStyle
		if PreviewTab(i) == p.previewTab {
			style = tabActiveStyle
		}
		rendered = append(rendered, style.Render(" "+tab+" "))
	}

	return strings.Join(rendered, " ")
}

// renderOutputContent renders agent output.
func (p *Plugin) renderOutputContent(width, height int) string {
	wt := p.selectedWorktree()
	if wt == nil {
		return dimText("No worktree selected")
	}

	if wt.Agent == nil {
		return dimText("No agent running\nPress 'a' to start an agent")
	}

	if wt.Agent.OutputBuf == nil {
		return dimText("No output yet")
	}

	lines := wt.Agent.OutputBuf.Lines()
	if len(lines) == 0 {
		return dimText("No output yet")
	}

	// Apply scroll offset and limit to height
	start := p.previewOffset
	if start >= len(lines) {
		start = len(lines) - 1
	}
	if start < 0 {
		start = 0
	}
	end := start + height
	if end > len(lines) {
		end = len(lines)
	}

	return strings.Join(lines[start:end], "\n")
}

// renderDiffContent renders git diff.
func (p *Plugin) renderDiffContent(width, height int) string {
	if p.diffContent == "" {
		wt := p.selectedWorktree()
		if wt == nil {
			return dimText("No worktree selected")
		}
		return dimText("No changes")
	}

	lines := splitLines(p.diffContent)

	// Apply scroll offset
	start := p.previewOffset
	if start >= len(lines) {
		start = len(lines) - 1
	}
	if start < 0 {
		start = 0
	}
	end := start + height
	if end > len(lines) {
		end = len(lines)
	}

	// Basic diff highlighting
	var rendered []string
	for _, line := range lines[start:end] {
		rendered = append(rendered, colorDiffLine(line, width))
	}

	return strings.Join(rendered, "\n")
}

// colorDiffLine applies basic diff coloring.
func colorDiffLine(line string, width int) string {
	if len(line) == 0 {
		return line
	}

	// Truncate if needed
	if len(line) > width {
		line = line[:width]
	}

	switch {
	case strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---"):
		return lipgloss.NewStyle().Bold(true).Render(line)
	case strings.HasPrefix(line, "@@"):
		return lipgloss.NewStyle().Foreground(lipgloss.Color("cyan")).Render(line)
	case strings.HasPrefix(line, "+"):
		return lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render(line)
	case strings.HasPrefix(line, "-"):
		return lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(line)
	default:
		return line
	}
}

// renderTaskContent renders linked task info.
func (p *Plugin) renderTaskContent(width, height int) string {
	wt := p.selectedWorktree()
	if wt == nil {
		return dimText("No worktree selected")
	}

	if wt.TaskID == "" {
		return dimText("No linked task\nPress 't' to link a task")
	}

	// TODO: Load and display task details from TD
	return fmt.Sprintf("Task: %s\n\n(Task details will be shown here)", wt.TaskID)
}

// renderCreateModal renders the new worktree modal.
func (p *Plugin) renderCreateModal(width, height int) string {
	// Modal dimensions
	modalW := 50
	if modalW > width-4 {
		modalW = width - 4
	}

	var lines []string
	lines = append(lines, lipgloss.NewStyle().Bold(true).Render("Create New Worktree"))
	lines = append(lines, "")

	// Name field
	nameLabel := "Name:"
	nameStyle := inputStyle
	if p.createFocus == 0 {
		nameStyle = inputFocusedStyle
	}
	nameValue := p.createName
	if nameValue == "" {
		nameValue = " "
	}
	lines = append(lines, nameLabel)
	lines = append(lines, nameStyle.Width(modalW-4).Render(nameValue))
	lines = append(lines, "")

	// Base branch field
	baseLabel := "Base Branch (default: current):"
	baseStyle := inputStyle
	if p.createFocus == 1 {
		baseStyle = inputFocusedStyle
	}
	baseValue := p.createBaseBranch
	if baseValue == "" {
		baseValue = " "
	}
	lines = append(lines, baseLabel)
	lines = append(lines, baseStyle.Width(modalW-4).Render(baseValue))
	lines = append(lines, "")

	// Task ID field
	taskLabel := "Link Task (optional):"
	taskStyle := inputStyle
	if p.createFocus == 2 {
		taskStyle = inputFocusedStyle
	}
	taskValue := p.createTaskID
	if taskValue == "" {
		taskValue = " "
	}
	lines = append(lines, taskLabel)
	lines = append(lines, taskStyle.Width(modalW-4).Render(taskValue))
	lines = append(lines, "")

	// Buttons
	createBtnStyle := buttonStyle
	if p.createFocus == 3 {
		createBtnStyle = buttonFocusedStyle
	}
	lines = append(lines, createBtnStyle.Render("Create"))

	content := strings.Join(lines, "\n")
	modal := modalStyle.Width(modalW).Render(content)

	// Center the modal
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, modal)
}

// dimText renders dim placeholder text.
func dimText(s string) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(s)
}

// formatRelativeTime formats a time as relative (e.g., "3m", "2h").
func formatRelativeTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}

	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "now"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}
