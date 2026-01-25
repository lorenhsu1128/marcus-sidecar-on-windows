package workspace

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/marcus/sidecar/internal/modal"
	"github.com/marcus/sidecar/internal/styles"
	"github.com/marcus/sidecar/internal/ui"
)

// renderCreateModal renders the new worktree modal with dimmed background.
func (p *Plugin) renderCreateModal(width, height int) string {
	background := p.renderListView(width, height)

	p.ensureCreateModal()
	if p.createModal == nil {
		return background
	}
	p.syncCreateModalFocus()

	modalContent := p.createModal.Render(width, height, p.mouseHandler)
	return ui.OverlayModal(background, modalContent, width, height)
}

// renderTaskLinkModal renders the task link modal for existing worktrees with dimmed background.
func (p *Plugin) renderTaskLinkModal(width, height int) string {
	// Render the background (list view)
	background := p.renderListView(width, height)

	// Modal dimensions - increased for better task display
	modalW := 70
	if modalW > width-4 {
		modalW = width - 4
	}

	// Calculate input field width
	// - modalStyle has border (2) + padding (4) = 6 chars
	// - inputStyle has border (2) + padding (2) = 4 chars
	inputW := modalW - 10
	if inputW < 20 {
		inputW = 20
	}

	// Set textinput width and remove default prompt
	p.taskSearchInput.Width = inputW
	p.taskSearchInput.Prompt = ""

	var sb strings.Builder
	title := "Link Task"
	if p.linkingWorktree != nil {
		title = fmt.Sprintf("Link Task to %s", p.linkingWorktree.Name)
	}
	sb.WriteString(lipgloss.NewStyle().Bold(true).Render(title))
	sb.WriteString("\n\n")

	// Search field - use textinput.View() for proper cursor rendering
	searchLabel := "Search tasks:"
	searchStyle := inputFocusedStyle()
	sb.WriteString(searchLabel)
	sb.WriteString("\n")
	sb.WriteString(searchStyle.Render(p.taskSearchInput.View()))

	// Task dropdown
	if p.taskSearchLoading {
		sb.WriteString("\n")
		sb.WriteString(dimText("  Loading tasks..."))
	} else if len(p.taskSearchFiltered) > 0 {
		maxDropdown := 8
		dropdownCount := min(maxDropdown, len(p.taskSearchFiltered))
		for i := 0; i < dropdownCount; i++ {
			task := p.taskSearchFiltered[i]
			prefix := "  "
			if i == p.taskSearchIdx {
				prefix = "> "
			}
			// Truncate title based on available width
			taskTitle := task.Title
			idWidth := len(task.ID)
			maxTitle := modalW - idWidth - 10
			if maxTitle < 10 {
				maxTitle = 10
			}
			if len(taskTitle) > maxTitle {
				taskTitle = taskTitle[:maxTitle-3] + "..."
			}
			line := fmt.Sprintf("%s%s  %s", prefix, task.ID, taskTitle)
			sb.WriteString("\n")
			if i == p.taskSearchIdx {
				sb.WriteString(lipgloss.NewStyle().Foreground(styles.Primary).Render(line))
			} else {
				sb.WriteString(dimText(line))
			}
		}
		if len(p.taskSearchFiltered) > maxDropdown {
			sb.WriteString("\n")
			sb.WriteString(dimText(fmt.Sprintf("  ... and %d more", len(p.taskSearchFiltered)-maxDropdown)))
		}
	} else if p.taskSearchInput.Value() != "" {
		sb.WriteString("\n")
		sb.WriteString(dimText("  No matching tasks"))
	} else if len(p.taskSearchAll) == 0 {
		sb.WriteString("\n")
		sb.WriteString(dimText("  No open tasks found"))
	} else {
		// Show all tasks when no query
		maxDropdown := 8
		dropdownCount := min(maxDropdown, len(p.taskSearchAll))
		for i := 0; i < dropdownCount; i++ {
			task := p.taskSearchAll[i]
			prefix := "  "
			if i == p.taskSearchIdx {
				prefix = "> "
			}
			taskTitle := task.Title
			idWidth := len(task.ID)
			maxTitle := modalW - idWidth - 10
			if maxTitle < 10 {
				maxTitle = 10
			}
			if len(taskTitle) > maxTitle {
				taskTitle = taskTitle[:maxTitle-3] + "..."
			}
			line := fmt.Sprintf("%s%s  %s", prefix, task.ID, taskTitle)
			sb.WriteString("\n")
			if i == p.taskSearchIdx {
				sb.WriteString(lipgloss.NewStyle().Foreground(styles.Primary).Render(line))
			} else {
				sb.WriteString(dimText(line))
			}
		}
		if len(p.taskSearchAll) > maxDropdown {
			sb.WriteString("\n")
			sb.WriteString(dimText(fmt.Sprintf("  ... and %d more", len(p.taskSearchAll)-maxDropdown)))
		}
	}

	sb.WriteString("\n\n")
	sb.WriteString(dimText("↑/↓ navigate  Enter select  Esc cancel"))

	content := sb.String()
	modal := modalStyle().Width(modalW).Render(content)

	// Calculate modal position for hit regions
	modalH := lipgloss.Height(modal)
	modalX := (width - modalW) / 2
	modalY := (height - modalH) / 2

	// Register hit regions for task dropdown items
	// Content: title(1) + blank(1) + label(1) + bordered-input(3) = 6 lines before dropdown
	// Border offset is 2: border(1) + padding(1)
	dropdownStartY := modalY + 2 + 6 // border(1) + padding(1) + content lines to dropdown

	// Determine which list to use for hit regions
	tasks := p.taskSearchFiltered
	if len(tasks) == 0 && p.taskSearchInput.Value() == "" && len(p.taskSearchAll) > 0 {
		tasks = p.taskSearchAll
	}

	if len(tasks) > 0 {
		maxDropdown := 8
		dropdownCount := min(maxDropdown, len(tasks))
		for i := 0; i < dropdownCount; i++ {
			p.mouseHandler.HitMap.AddRect(regionTaskLinkDropdown, modalX+2, dropdownStartY+i, modalW-6, 1, i)
		}
	}

	// Use OverlayModal for dimmed background effect
	return ui.OverlayModal(background, modal, width, height)
}

// renderConfirmDeleteModal renders the delete confirmation modal.
func (p *Plugin) renderConfirmDeleteModal(width, height int) string {
	// Render the background (list view)
	background := p.renderListView(width, height)

	p.ensureConfirmDeleteModal()
	if p.deleteConfirmModal == nil {
		return background
	}

	modalContent := p.deleteConfirmModal.Render(width, height, p.mouseHandler)
	return ui.OverlayModal(background, modalContent, width, height)
}

const (
	deleteConfirmLocalID  = "delete-confirm-local-branch"
	deleteConfirmRemoteID = "delete-confirm-remote-branch"
	deleteConfirmDeleteID = "delete-confirm-delete"
	deleteConfirmCancelID = "delete-confirm-cancel"
)

// ensureConfirmDeleteModal builds/rebuilds the delete confirmation modal.
func (p *Plugin) ensureConfirmDeleteModal() {
	if p.deleteConfirmWorktree == nil {
		return
	}

	modalW := 58
	if modalW > p.width-4 {
		modalW = p.width - 4
	}
	if modalW < 20 {
		modalW = 20
	}

	if p.deleteConfirmModal != nil && p.deleteConfirmModalWidth == modalW {
		return
	}
	p.deleteConfirmModalWidth = modalW

	p.deleteConfirmModal = modal.New("Delete Worktree?",
		modal.WithWidth(modalW),
		modal.WithVariant(modal.VariantDanger),
		modal.WithHints(false),
	).
		AddSection(p.deleteConfirmInfoSection()).
		AddSection(modal.Spacer()).
		AddSection(p.deleteConfirmWarningSection()).
		AddSection(modal.Spacer()).
		AddSection(modal.When(func() bool { return !p.deleteIsMainBranch }, p.deleteConfirmBranchHeaderSection())).
		AddSection(modal.When(func() bool { return !p.deleteIsMainBranch }, modal.Checkbox(deleteConfirmLocalID, "Delete local branch", &p.deleteLocalBranchOpt))).
		AddSection(modal.When(func() bool { return !p.deleteIsMainBranch }, p.deleteConfirmLocalHintSection())).
		AddSection(modal.When(func() bool { return !p.deleteIsMainBranch && p.deleteHasRemote }, modal.Checkbox(deleteConfirmRemoteID, "Delete remote branch", &p.deleteRemoteBranchOpt))).
		AddSection(modal.When(func() bool { return !p.deleteIsMainBranch && p.deleteHasRemote }, p.deleteConfirmRemoteHintSection())).
		AddSection(modal.When(func() bool { return !p.deleteIsMainBranch }, modal.Spacer())).
		AddSection(modal.Buttons(
			modal.Btn(" Delete ", deleteConfirmDeleteID, modal.BtnDanger()),
			modal.Btn(" Cancel ", deleteConfirmCancelID),
		))
}

func (p *Plugin) deleteConfirmInfoSection() modal.Section {
	return modal.Custom(func(contentWidth int, focusID, hoverID string) modal.RenderedSection {
		if p.deleteConfirmWorktree == nil {
			return modal.RenderedSection{}
		}

		wt := p.deleteConfirmWorktree
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Name:   %s\n", lipgloss.NewStyle().Bold(true).Render(wt.Name)))
		sb.WriteString(fmt.Sprintf("Branch: %s\n", wt.Branch))
		sb.WriteString(fmt.Sprintf("Path:   %s", dimText(wt.Path)))

		return modal.RenderedSection{Content: sb.String()}
	}, nil)
}

func (p *Plugin) deleteConfirmWarningSection() modal.Section {
	return modal.Custom(func(contentWidth int, focusID, hoverID string) modal.RenderedSection {
		warningStyle := lipgloss.NewStyle().Foreground(styles.Warning)

		var sb strings.Builder
		sb.WriteString(warningStyle.Render("This will:"))
		sb.WriteString("\n")
		sb.WriteString(dimText("  • Remove the working directory"))
		sb.WriteString("\n")
		sb.WriteString(dimText("  • Uncommitted changes will be lost"))

		return modal.RenderedSection{Content: sb.String()}
	}, nil)
}

func (p *Plugin) deleteConfirmBranchHeaderSection() modal.Section {
	return modal.Custom(func(contentWidth int, focusID, hoverID string) modal.RenderedSection {
		content := lipgloss.NewStyle().Bold(true).Render("Branch Cleanup (Optional)")
		return modal.RenderedSection{Content: content}
	}, nil)
}

func (p *Plugin) deleteConfirmLocalHintSection() modal.Section {
	return modal.Custom(func(contentWidth int, focusID, hoverID string) modal.RenderedSection {
		if p.deleteConfirmWorktree == nil {
			return modal.RenderedSection{}
		}
		wt := p.deleteConfirmWorktree
		return modal.RenderedSection{Content: dimText("  Removes '" + wt.Branch + "' locally")}
	}, nil)
}

func (p *Plugin) deleteConfirmRemoteHintSection() modal.Section {
	return modal.Custom(func(contentWidth int, focusID, hoverID string) modal.RenderedSection {
		if p.deleteConfirmWorktree == nil {
			return modal.RenderedSection{}
		}
		wt := p.deleteConfirmWorktree
		return modal.RenderedSection{Content: dimText("  Removes 'origin/" + wt.Branch + "'")}
	}, nil)
}

const (
	deleteShellConfirmDeleteID = "delete-shell-confirm-delete"
	deleteShellConfirmCancelID = "delete-shell-confirm-cancel"
)

// renderConfirmDeleteShellModal renders the shell delete confirmation modal.
func (p *Plugin) renderConfirmDeleteShellModal(width, height int) string {
	// Render the background (list view)
	background := p.renderListView(width, height)

	p.ensureConfirmDeleteShellModal()
	if p.deleteShellModal == nil {
		return background
	}

	modalContent := p.deleteShellModal.Render(width, height, p.mouseHandler)
	return ui.OverlayModal(background, modalContent, width, height)
}

// ensureConfirmDeleteShellModal builds/rebuilds the shell delete confirmation modal.
func (p *Plugin) ensureConfirmDeleteShellModal() {
	if p.deleteConfirmShell == nil {
		return
	}

	modalW := 50
	if modalW > p.width-4 {
		modalW = p.width - 4
	}
	if modalW < 20 {
		modalW = 20
	}

	if p.deleteShellModal != nil && p.deleteShellModalWidth == modalW {
		return
	}
	p.deleteShellModalWidth = modalW

	p.deleteShellModal = modal.New("Delete Shell?",
		modal.WithWidth(modalW),
		modal.WithVariant(modal.VariantDanger),
		modal.WithHints(false),
	).
		AddSection(p.deleteShellInfoSection()).
		AddSection(modal.Spacer()).
		AddSection(p.deleteShellWarningSection()).
		AddSection(modal.Spacer()).
		AddSection(modal.Buttons(
			modal.Btn(" Delete ", deleteShellConfirmDeleteID, modal.BtnDanger()),
			modal.Btn(" Cancel ", deleteShellConfirmCancelID),
		))
}

func (p *Plugin) deleteShellInfoSection() modal.Section {
	return modal.Custom(func(contentWidth int, focusID, hoverID string) modal.RenderedSection {
		if p.deleteConfirmShell == nil {
			return modal.RenderedSection{}
		}
		shell := p.deleteConfirmShell

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Name:    %s\n", lipgloss.NewStyle().Bold(true).Render(shell.Name)))
		sb.WriteString(fmt.Sprintf("Session: %s", dimText(shell.TmuxName)))

		return modal.RenderedSection{Content: sb.String()}
	}, nil)
}

func (p *Plugin) deleteShellWarningSection() modal.Section {
	return modal.Custom(func(contentWidth int, focusID, hoverID string) modal.RenderedSection {
		warningStyle := lipgloss.NewStyle().Foreground(styles.Warning)

		var sb strings.Builder
		sb.WriteString(warningStyle.Render("This will:"))
		sb.WriteString("\n")
		sb.WriteString(dimText("  • Terminate the tmux session"))
		sb.WriteString("\n")
		sb.WriteString(dimText("  • Any running processes will be killed"))

		return modal.RenderedSection{Content: sb.String()}
	}, nil)
}

const (
	renameShellInputID  = "rename-shell-input"
	renameShellRenameID = "rename-shell-rename"
	renameShellCancelID = "rename-shell-cancel"
	renameShellActionID = "rename-shell-action"
)

// ensureRenameShellModal builds/rebuilds the rename shell modal.
func (p *Plugin) ensureRenameShellModal() {
	if p.renameShellSession == nil {
		return
	}

	modalW := 50
	if modalW > p.width-4 {
		modalW = p.width - 4
	}
	if modalW < 20 {
		modalW = 20
	}

	// Only rebuild if modal doesn't exist or width changed
	if p.renameShellModal != nil && p.renameShellModalWidth == modalW {
		return
	}
	p.renameShellModalWidth = modalW

	p.renameShellModal = modal.New("Rename Shell",
		modal.WithWidth(modalW),
		modal.WithPrimaryAction(renameShellActionID),
		modal.WithHints(false),
	).
		AddSection(p.renameShellInfoSection()).
		AddSection(modal.Spacer()).
		AddSection(modal.InputWithLabel(renameShellInputID, "New Name:", &p.renameShellInput)).
		AddSection(modal.When(func() bool { return p.renameShellError != "" }, p.renameShellErrorSection())).
		AddSection(modal.Spacer()).
		AddSection(modal.Buttons(
			modal.Btn(" Rename ", renameShellRenameID),
			modal.Btn(" Cancel ", renameShellCancelID),
		))
}

// renameShellInfoSection renders the shell info section.
func (p *Plugin) renameShellInfoSection() modal.Section {
	return modal.Custom(func(contentWidth int, focusID, hoverID string) modal.RenderedSection {
		if p.renameShellSession == nil {
			return modal.RenderedSection{}
		}

		shell := p.renameShellSession
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Session: %s\n", dimText(shell.TmuxName)))
		sb.WriteString(fmt.Sprintf("Current: %s", lipgloss.NewStyle().Bold(true).Render(shell.Name)))

		return modal.RenderedSection{Content: sb.String()}
	}, nil)
}

// renameShellErrorSection renders the error message section.
func (p *Plugin) renameShellErrorSection() modal.Section {
	return modal.Custom(func(contentWidth int, focusID, hoverID string) modal.RenderedSection {
		if p.renameShellError == "" {
			return modal.RenderedSection{}
		}

		errStyle := lipgloss.NewStyle().Foreground(styles.Error)
		content := errStyle.Render("Error: " + p.renameShellError)

		return modal.RenderedSection{Content: content}
	}, nil)
}

// renderRenameShellModal renders the rename shell modal.
func (p *Plugin) renderRenameShellModal(width, height int) string {
	background := p.renderListView(width, height)

	p.ensureRenameShellModal()
	if p.renameShellModal == nil {
		return background
	}

	modalContent := p.renameShellModal.Render(width, height, p.mouseHandler)
	return ui.OverlayModal(background, modalContent, width, height)
}

// renderPromptPickerModal renders the prompt picker modal.
func (p *Plugin) renderPromptPickerModal(width, height int) string {
	// Render the background (create modal behind it)
	background := p.renderCreateModal(width, height)

	p.ensurePromptPickerModal()
	if p.promptPickerModal == nil {
		return background
	}

	modalContent := p.promptPickerModal.Render(width, height, p.mouseHandler)
	return ui.OverlayModal(background, modalContent, width, height)
}

// ensureAgentChoiceModal builds/rebuilds the agent choice modal.
func (p *Plugin) ensureAgentChoiceModal() {
	if p.agentChoiceWorktree == nil {
		return
	}

	modalW := 50
	if p.width > 0 && modalW > p.width-4 {
		modalW = p.width - 4
	}
	if modalW < 20 {
		modalW = 20
	}

	// Only rebuild if modal doesn't exist or width changed
	if p.agentChoiceModal != nil && p.agentChoiceModalWidth == modalW {
		return
	}
	p.agentChoiceModalWidth = modalW

	// Build list items for the options
	items := []modal.ListItem{
		{ID: "agent-choice-attach", Label: "Attach to session"},
		{ID: "agent-choice-restart", Label: "Restart agent"},
	}

	title := fmt.Sprintf("Agent Running: %s", p.agentChoiceWorktree.Name)

	p.agentChoiceModal = modal.New(title,
		modal.WithWidth(modalW),
		modal.WithPrimaryAction(agentChoiceActionID),
		modal.WithHints(false),
	).
		AddSection(modal.Text("An agent is already running on this worktree.\nWhat would you like to do?")).
		AddSection(modal.Spacer()).
		AddSection(modal.List(agentChoiceListID, items, &p.agentChoiceIdx, modal.WithMaxVisible(2))).
		AddSection(modal.Spacer()).
		AddSection(modal.Buttons(
			modal.Btn(" Confirm ", agentChoiceConfirmID),
			modal.Btn(" Cancel ", agentChoiceCancelID),
		))
}

// renderAgentChoiceModal renders the agent action choice modal.
func (p *Plugin) renderAgentChoiceModal(width, height int) string {
	background := p.renderListView(width, height)

	p.ensureAgentChoiceModal()
	if p.agentChoiceModal == nil {
		return background
	}

	modalContent := p.agentChoiceModal.Render(width, height, p.mouseHandler)
	return ui.OverlayModal(background, modalContent, width, height)
}

// clearAgentChoiceModal clears agent choice modal state.
func (p *Plugin) clearAgentChoiceModal() {
	p.agentChoiceWorktree = nil
	p.agentChoiceIdx = 0
	p.agentChoiceModal = nil
	p.agentChoiceModalWidth = 0
}

// renderMergeModal renders the merge workflow modal with dimmed background.
func (p *Plugin) renderMergeModal(width, height int) string {
	// Render the background (list view)
	background := p.renderListView(width, height)

	if p.mergeState == nil {
		return background
	}

	// Modal dimensions
	modalW := 70
	if modalW > width-4 {
		modalW = width - 4
	}
	modalH := height - 6
	if modalH < 20 {
		modalH = 20
	}

	var sb strings.Builder

	// Title
	title := fmt.Sprintf("Merge Workflow: %s", p.mergeState.Worktree.Name)
	sb.WriteString(lipgloss.NewStyle().Bold(true).Render(title))
	sb.WriteString("\n\n")

	// Progress indicators - show different steps based on merge method
	var steps []MergeWorkflowStep
	if p.mergeState.UseDirectMerge {
		// Direct merge path
		steps = []MergeWorkflowStep{
			MergeStepReviewDiff,
			MergeStepMergeMethod,
			MergeStepDirectMerge,
			MergeStepPostMergeConfirmation,
			MergeStepCleanup,
		}
	} else {
		// PR workflow path
		steps = []MergeWorkflowStep{
			MergeStepReviewDiff,
			MergeStepMergeMethod,
			MergeStepPush,
			MergeStepCreatePR,
			MergeStepWaitingMerge,
			MergeStepPostMergeConfirmation,
			MergeStepCleanup,
		}
	}

	for _, step := range steps {
		status := p.mergeState.StepStatus[step]
		icon := "○" // pending
		color := styles.TextMuted

		switch status {
		case "running":
			icon = "●"
			color = styles.Warning // yellow
		case "done":
			icon = "✓"
			color = styles.Success // green
		case "error":
			icon = "✗"
			color = styles.Error // red
		case "skipped":
			icon = "○"
			color = styles.TextMuted // gray, same as pending
		}

		// Highlight current step
		stepName := step.String()
		if step == p.mergeState.Step {
			stepName = lipgloss.NewStyle().Bold(true).Render(stepName)
		}

		stepLine := fmt.Sprintf("  %s %s",
			lipgloss.NewStyle().Foreground(color).Render(icon),
			stepName,
		)
		sb.WriteString(stepLine)
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("─", min(modalW-4, 60)))
	sb.WriteString("\n\n")

	// Step-specific content
	switch p.mergeState.Step {
	case MergeStepReviewDiff:
		sb.WriteString(lipgloss.NewStyle().Bold(true).Render("Files Changed:"))
		sb.WriteString("\n\n")
		if p.mergeState.StepStatus[MergeStepReviewDiff] == "running" {
			sb.WriteString(dimText("Loading..."))
			sb.WriteString("\n")
		} else if p.mergeState.DiffSummary != "" {
			// Truncate to fit modal
			summaryLines := strings.Split(p.mergeState.DiffSummary, "\n")
			maxLines := modalH - 15 // Account for header, progress, footer
			if maxLines < 5 {
				maxLines = 5
			}
			if len(summaryLines) > maxLines {
				summaryLines = summaryLines[:maxLines]
				summaryLines = append(summaryLines, fmt.Sprintf("... (%d more files)", len(strings.Split(p.mergeState.DiffSummary, "\n"))-maxLines))
			}
			for _, line := range summaryLines {
				sb.WriteString(p.colorStatLine(line, modalW-4))
				sb.WriteString("\n")
			}
		} else {
			sb.WriteString(dimText("No files changed"))
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
		sb.WriteString(dimText("Press Enter to continue, Esc to cancel"))

	case MergeStepMergeMethod:
		sb.WriteString(lipgloss.NewStyle().Bold(true).Render("Choose Merge Method:"))
		sb.WriteString("\n\n")

		baseBranch := resolveBaseBranch(p.mergeState.Worktree)

		// Style helpers for selection and hover states
		selectedStyle := func(s string) string {
			return lipgloss.NewStyle().Foreground(styles.Primary).Render(s)
		}
		hoverStyle := func(s string) string {
			return lipgloss.NewStyle().Foreground(styles.TextSecondary).Render(s)
		}

		// Option 1: Create PR (default)
		prIcon := "○"
		prStyle := dimText
		if p.mergeState.MergeMethodOption == 0 {
			prIcon = "●"
			prStyle = selectedStyle
		} else if p.mergeMethodHover == 1 {
			prStyle = hoverStyle
		}
		sb.WriteString(prStyle(fmt.Sprintf(" %s Create Pull Request (Recommended)", prIcon)))
		sb.WriteString("\n")
		sb.WriteString(dimText("      Push to origin and create a GitHub PR for review"))
		sb.WriteString("\n\n")

		// Option 2: Direct merge
		directIcon := "○"
		directStyle := dimText
		if p.mergeState.MergeMethodOption == 1 {
			directIcon = "●"
			directStyle = selectedStyle
		} else if p.mergeMethodHover == 2 {
			directStyle = hoverStyle
		}
		sb.WriteString(directStyle(fmt.Sprintf(" %s Direct Merge", directIcon)))
		sb.WriteString("\n")
		sb.WriteString(dimText(fmt.Sprintf("      Merge directly to '%s' without PR", baseBranch)))
		sb.WriteString("\n")
		sb.WriteString(lipgloss.NewStyle().Foreground(styles.Warning).Render("      Warning: Bypasses code review"))
		sb.WriteString("\n\n")

		sb.WriteString(dimText("↑/↓: select   Enter: continue   Esc: cancel"))

	case MergeStepDirectMerge:
		sb.WriteString("Merging directly to base branch...")
		sb.WriteString("\n\n")
		baseBranch := resolveBaseBranch(p.mergeState.Worktree)
		sb.WriteString(dimText(fmt.Sprintf("Merging '%s' into '%s'...", p.mergeState.Worktree.Branch, baseBranch)))

	case MergeStepPush:
		sb.WriteString("Pushing branch to remote...")

	case MergeStepCreatePR:
		sb.WriteString("Creating pull request...")

	case MergeStepWaitingMerge:
		if p.mergeState.ExistingPR {
			sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(styles.Warning).Render("Using Existing Pull Request"))
		} else {
			sb.WriteString(lipgloss.NewStyle().Bold(true).Render("Pull Request Created"))
		}
		sb.WriteString("\n\n")
		if p.mergeState.PRURL != "" {
			sb.WriteString(fmt.Sprintf("URL: %s", p.mergeState.PRURL))
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
		sb.WriteString("Checking merge status every 30 seconds...")
		sb.WriteString("\n\n")
		sb.WriteString(strings.Repeat("─", min(modalW-4, 60)))
		sb.WriteString("\n\n")

		// Radio button selection
		sb.WriteString(lipgloss.NewStyle().Bold(true).Render("After merge:"))
		sb.WriteString("\n\n")

		// Option 1: Delete worktree (default)
		if p.mergeState.DeleteAfterMerge {
			sb.WriteString(lipgloss.NewStyle().Foreground(styles.Primary).Render(" ● Delete worktree after merge"))
		} else {
			sb.WriteString(dimText(" ○ Delete worktree after merge"))
		}
		sb.WriteString("\n")

		// Option 2: Keep worktree
		if !p.mergeState.DeleteAfterMerge {
			sb.WriteString(lipgloss.NewStyle().Foreground(styles.Primary).Render(" ● Keep worktree"))
		} else {
			sb.WriteString(dimText(" ○ Keep worktree"))
		}
		sb.WriteString("\n\n")

		sb.WriteString(dimText(" (This takes effect only once the PR is merged)"))
		sb.WriteString("\n\n")
		sb.WriteString(dimText("Enter: check now   o: open PR   Esc: exit   ↑/↓: change option"))

	case MergeStepPostMergeConfirmation:
		// Success header
		mergeMethod := "PR Merged"
		if p.mergeState.UseDirectMerge {
			mergeMethod = "Direct Merge Complete"
		}
		sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(styles.Success).Render(mergeMethod + "!"))
		sb.WriteString("\n\n")

		sb.WriteString(strings.Repeat("─", min(modalW-4, 60)))
		sb.WriteString("\n\n")

		// Cleanup options header
		sb.WriteString(lipgloss.NewStyle().Bold(true).Render("Cleanup Options"))
		sb.WriteString("\n")
		sb.WriteString(dimText("Select what to clean up:"))
		sb.WriteString("\n\n")

		baseBranch := resolveBaseBranch(p.mergeState.Worktree)

		// Checkbox options
		type checkboxOpt struct {
			label   string
			checked bool
			hint    string
		}
		opts := []checkboxOpt{
			{"Delete local worktree", p.mergeState.DeleteLocalWorktree,
				"Removes " + p.mergeState.Worktree.Path},
			{"Delete local branch", p.mergeState.DeleteLocalBranch,
				"Removes '" + p.mergeState.Worktree.Branch + "' locally"},
			{"Delete remote branch", p.mergeState.DeleteRemoteBranch,
				"Removes from GitHub (often auto-deleted)"},
		}

		for i, opt := range opts {
			checkbox := "[ ]"
			if opt.checked {
				checkbox = "[x]"
			}

			line := fmt.Sprintf("  %s %s", checkbox, opt.label)

			if p.mergeState.ConfirmationFocus == i {
				sb.WriteString(lipgloss.NewStyle().Foreground(styles.Primary).Bold(true).Render("> " + line[2:]))
			} else if p.mergeConfirmCheckboxHover == i+1 {
				// Hover state (subtle highlight)
				sb.WriteString(lipgloss.NewStyle().Foreground(styles.Secondary).Render(line))
			} else {
				sb.WriteString(line)
			}
			sb.WriteString("\n")
			sb.WriteString(dimText("      " + opt.hint))
			sb.WriteString("\n")
		}

		sb.WriteString("\n")
		sb.WriteString(strings.Repeat("─", min(modalW-4, 60)))
		sb.WriteString("\n\n")

		// Pull section
		sb.WriteString(lipgloss.NewStyle().Bold(true).Render("Sync Local Branch"))
		sb.WriteString("\n")

		// Pull checkbox
		pullCheckbox := "[ ]"
		if p.mergeState.PullAfterMerge {
			pullCheckbox = "[x]"
		}
		pullLabel := fmt.Sprintf("Update local '%s' from remote", baseBranch)
		pullLine := fmt.Sprintf("  %s %s", pullCheckbox, pullLabel)
		if p.mergeState.ConfirmationFocus == 3 {
			sb.WriteString(lipgloss.NewStyle().Foreground(styles.Primary).Bold(true).Render("> " + pullLine[2:]))
		} else if p.mergeConfirmCheckboxHover == 4 {
			// Hover state (subtle highlight)
			sb.WriteString(lipgloss.NewStyle().Foreground(styles.Secondary).Render(pullLine))
		} else {
			sb.WriteString(pullLine)
		}
		sb.WriteString("\n")
		if p.mergeState.CurrentBranch != "" {
			sb.WriteString(dimText(fmt.Sprintf("      Current branch: %s", p.mergeState.CurrentBranch)))
		} else {
			sb.WriteString(dimText(fmt.Sprintf("      Updates local %s to include merged PR", baseBranch)))
		}
		sb.WriteString("\n\n")

		// Buttons (now at focus indices 4 and 5)
		confirmLabel := " Clean Up "
		skipLabel := " Skip All "

		confirmStyle := lipgloss.NewStyle().Background(styles.Primary).Foreground(styles.TextInverse)
		skipStyle := lipgloss.NewStyle().Background(styles.BgTertiary).Foreground(styles.TextPrimary)

		if p.mergeState.ConfirmationFocus == 4 {
			confirmStyle = confirmStyle.Bold(true).Background(styles.Success)
		} else if p.mergeConfirmButtonHover == 1 {
			// Hover state for Clean Up button
			confirmStyle = confirmStyle.Background(styles.Secondary)
		}
		if p.mergeState.ConfirmationFocus == 5 {
			skipStyle = skipStyle.Bold(true).Background(styles.Warning)
		} else if p.mergeConfirmButtonHover == 2 {
			// Hover state for Skip All button
			skipStyle = skipStyle.Background(styles.TextMuted)
		}

		sb.WriteString("  ")
		sb.WriteString(confirmStyle.Render(confirmLabel))
		sb.WriteString("  ")
		sb.WriteString(skipStyle.Render(skipLabel))
		sb.WriteString("\n\n")

		sb.WriteString(dimText("↑/↓: navigate  space: toggle  enter: confirm  esc: cancel"))

	case MergeStepCleanup:
		sb.WriteString("Cleaning up worktree and branch...")

	case MergeStepDone:
		sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(styles.Success).Render("Merge workflow complete!"))
		sb.WriteString("\n\n")

		// Show cleanup summary
		if p.mergeState.CleanupResults != nil {
			results := p.mergeState.CleanupResults
			sb.WriteString("Summary:\n")

			successStyle := lipgloss.NewStyle().Foreground(styles.Success)
			if results.LocalWorktreeDeleted {
				sb.WriteString(successStyle.Render("  ✓ Local worktree deleted"))
				sb.WriteString("\n")
			}
			if results.LocalBranchDeleted {
				sb.WriteString(successStyle.Render("  ✓ Local branch deleted"))
				sb.WriteString("\n")
			}
			if results.RemoteBranchDeleted {
				sb.WriteString(successStyle.Render("  ✓ Remote branch deleted"))
				sb.WriteString("\n")
			}
			if results.PullAttempted {
				if results.PullSuccess {
					sb.WriteString(successStyle.Render("  ✓ Pulled latest changes"))
					sb.WriteString("\n")
				} else if results.PullError != nil {
					// Truncated error with toggle
					warnStyle := lipgloss.NewStyle().Foreground(styles.Warning)
					errorStyle := lipgloss.NewStyle().Foreground(styles.Error)

					sb.WriteString(warnStyle.Render("  ⚠ Pull failed: "))
					sb.WriteString(errorStyle.Render(results.PullErrorSummary))
					sb.WriteString("\n")

					// Show full details if expanded
					if results.ShowErrorDetails {
						sb.WriteString("\n")
						sb.WriteString(dimText("  Details:"))
						sb.WriteString("\n")
						// Wrap long lines and indent (cap at 10 lines)
						allDetailLines := strings.Split(results.PullErrorFull, "\n")
						maxDetailLines := 10
						detailLines := allDetailLines
						if len(allDetailLines) > maxDetailLines {
							detailLines = allDetailLines[:maxDetailLines]
						}
						for _, line := range detailLines {
							if line = strings.TrimSpace(line); line != "" {
								sb.WriteString(dimText("    " + line))
								sb.WriteString("\n")
							}
						}
						if len(allDetailLines) > maxDetailLines {
							sb.WriteString(dimText(fmt.Sprintf("    ... (%d more lines)",
								len(allDetailLines)-maxDetailLines)))
							sb.WriteString("\n")
						}
						sb.WriteString("\n")
						sb.WriteString(dimText("  Press 'd' to hide details"))
					} else {
						sb.WriteString(dimText("  Press 'd' for full error details"))
					}
					sb.WriteString("\n")

					// Resolution actions for diverged branches
					if results.BranchDiverged {
						sb.WriteString("\n")
						sepLen := modalW - 4
						if sepLen > 60 {
							sepLen = 60
						}
						sb.WriteString(strings.Repeat("─", sepLen))
						sb.WriteString("\n\n")
						sb.WriteString(lipgloss.NewStyle().Bold(true).Render("Resolution Options"))
						sb.WriteString("\n")
						sb.WriteString(dimText(fmt.Sprintf("  Your local '%s' has diverged from remote.", results.BaseBranch)))
						sb.WriteString("\n\n")

						// Rebase option
						sb.WriteString(dimText("    [r] Rebase local onto remote"))
						sb.WriteString("\n")
						sb.WriteString(dimText("        Replay your local commits on top of remote changes"))
						sb.WriteString("\n\n")

						// Merge option
						sb.WriteString(dimText("    [m] Merge remote into local"))
						sb.WriteString("\n")
						sb.WriteString(dimText("        Creates a merge commit combining both histories"))
						sb.WriteString("\n")
					}
				}
			}

			// Show any errors/warnings
			if len(results.Errors) > 0 {
				sb.WriteString("\n")
				sb.WriteString(lipgloss.NewStyle().Foreground(styles.Warning).Render("Warnings:"))
				sb.WriteString("\n")
				for _, err := range results.Errors {
					sb.WriteString(dimText("  • " + err))
					sb.WriteString("\n")
				}
			}
		} else {
			// No cleanup was performed
			sb.WriteString("No cleanup performed. Worktree and branches remain.")
		}

		sb.WriteString("\n\n")
		// Context-aware footer hints
		if p.mergeState.CleanupResults != nil && p.mergeState.CleanupResults.BranchDiverged {
			sb.WriteString(dimText("r: rebase  m: merge  d: details  Enter: close"))
		} else if p.mergeState.CleanupResults != nil && p.mergeState.CleanupResults.PullError != nil {
			sb.WriteString(dimText("d: details  Enter: close"))
		} else {
			sb.WriteString(dimText("Press Enter to close"))
		}
	}

	// Show error if any
	if p.mergeState.Error != nil {
		sb.WriteString("\n\n")
		sb.WriteString(lipgloss.NewStyle().Foreground(styles.Error).Render(
			fmt.Sprintf("Error: %s", p.mergeState.Error.Error())))
	}

	content := sb.String()
	modal := modalStyle().Width(modalW).Render(content)

	// Register hit regions for merge method options during MergeStepMergeMethod
	if p.mergeState.Step == MergeStepMergeMethod {
		modalH := lipgloss.Height(modal)
		modalX := (width - modalW) / 2
		modalY := (height - modalH) / 2

		// Calculate position of merge method radio options
		// Border offset is 2: border(1) + padding(1)
		// Content structure before step content:
		// - Title (1 line)
		// - blank (1 line)
		// - Progress steps: 5 if UseDirectMerge, 7 otherwise
		// - blank (1 line)
		// - separator (1 line)
		// - blank (2 lines from \n\n)
		progressSteps := 7
		if p.mergeState.UseDirectMerge {
			progressSteps = 5
		}
		// Lines before step content: 1 + 1 + progressSteps + 1 + 1 + 2 = progressSteps + 6
		// Then "Choose Merge Method:" (1) + blank (1) = 2 more lines before options
		option1Y := modalY + 2 + progressSteps + 6 + 2 // border(1) + padding(1) + header content + step header + blank
		option2Y := option1Y + 3                       // Option 1 text + description + blank

		p.mouseHandler.HitMap.AddRect(regionMergeMethodOption, modalX+2, option1Y, modalW-6, 1, 0) // Create PR
		p.mouseHandler.HitMap.AddRect(regionMergeMethodOption, modalX+2, option2Y, modalW-6, 1, 1) // Direct Merge
	}

	// Register hit regions for radio buttons during MergeStepWaitingMerge
	if p.mergeState.Step == MergeStepWaitingMerge {
		modalH := lipgloss.Height(modal)
		modalX := (width - modalW) / 2
		modalY := (height - modalH) / 2

		// Radio buttons position calculated from bottom of content
		// Border offset is 2: border(1) + padding(1)
		contentLines := strings.Count(content, "\n")
		// Radio buttons are approximately at contentLines - 7 (from bottom)
		// "Delete worktree" and "Keep worktree" options
		radio1Y := modalY + 2 + contentLines - 7
		radio2Y := radio1Y + 1
		p.mouseHandler.HitMap.AddRect(regionMergeRadio, modalX+2, radio1Y, modalW-6, 1, 0) // Delete
		p.mouseHandler.HitMap.AddRect(regionMergeRadio, modalX+2, radio2Y, modalW-6, 1, 1) // Keep
	}

	// Register hit regions for post-merge confirmation step
	if p.mergeState.Step == MergeStepPostMergeConfirmation {
		modalH := lipgloss.Height(modal)
		modalX := (width - modalW) / 2
		modalY := (height - modalH) / 2

		// Calculate positions based on content structure
		// Title(1) + blank(1) + separator(1) + blank(1) + header(1) + subtext(1) + blank(1) = 7 lines before checkboxes
		// Border offset is 2: border(1) + padding(1)
		checkboxBaseY := modalY + 2 + 7 // border(1) + padding(1) + content lines to checkboxes

		// Three cleanup checkbox options (2 lines each: checkbox + hint)
		for i := 0; i < 3; i++ {
			p.mouseHandler.HitMap.AddRect(
				regionMergeConfirmCheckbox,
				modalX+2,
				checkboxBaseY+(i*2),
				modalW-6,
				1,
				i,
			)
		}

		// Pull checkbox (focus index 3)
		// After 3 cleanup checkboxes (6 lines) + blank(1) + separator(1) + blank(1) + header(1) = 10 lines
		pullCheckboxY := checkboxBaseY + 10
		p.mouseHandler.HitMap.AddRect(regionMergeConfirmCheckbox, modalX+2, pullCheckboxY, modalW-6, 1, 3)

		// Button hit regions (after pull checkbox + hint + blank)
		buttonY := pullCheckboxY + 3
		p.mouseHandler.HitMap.AddRect(regionMergeConfirmButton, modalX+4, buttonY, 12, 1, nil)
		p.mouseHandler.HitMap.AddRect(regionMergeSkipButton, modalX+18, buttonY, 12, 1, nil)
	}

	// Use OverlayModal for dimmed background effect
	return ui.OverlayModal(background, modal, width, height)
}

// renderCommitForMergeModal renders the commit-before-merge modal.
func (p *Plugin) renderCommitForMergeModal(width, height int) string {
	// Render the background (list view)
	background := p.renderListView(width, height)

	if p.mergeCommitState == nil {
		return background
	}

	// Modal dimensions
	modalW := 60
	if modalW > width-4 {
		modalW = width - 4
	}

	// Calculate input field width
	inputW := modalW - 10
	if inputW < 20 {
		inputW = 20
	}

	// Set textinput width and remove default prompt
	p.mergeCommitMessageInput.Width = inputW
	p.mergeCommitMessageInput.Prompt = ""

	wt := p.mergeCommitState.Worktree

	var sb strings.Builder
	title := "Uncommitted Changes"
	sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(styles.Warning).Render(title))
	sb.WriteString("\n\n")

	// Workspace info
	sb.WriteString(fmt.Sprintf("Workspace: %s\n", lipgloss.NewStyle().Bold(true).Render(wt.Name)))
	sb.WriteString(fmt.Sprintf("Branch:   %s\n", wt.Branch))
	sb.WriteString("\n")

	// Change counts
	sb.WriteString("Changes to commit:\n")
	if p.mergeCommitState.StagedCount > 0 {
		sb.WriteString(fmt.Sprintf("  • %d staged file(s)\n", p.mergeCommitState.StagedCount))
	}
	if p.mergeCommitState.ModifiedCount > 0 {
		sb.WriteString(fmt.Sprintf("  • %d modified file(s)\n", p.mergeCommitState.ModifiedCount))
	}
	if p.mergeCommitState.UntrackedCount > 0 {
		sb.WriteString(fmt.Sprintf("  • %d untracked file(s)\n", p.mergeCommitState.UntrackedCount))
	}
	sb.WriteString("\n")

	// Info message
	infoStyle := lipgloss.NewStyle().Foreground(styles.TextMuted)
	sb.WriteString(infoStyle.Render("You must commit these changes before creating a PR."))
	sb.WriteString("\n")
	sb.WriteString(infoStyle.Render("All changes will be staged and committed."))
	sb.WriteString("\n\n")

	// Commit message field
	sb.WriteString("Commit message:\n")
	sb.WriteString(inputFocusedStyle().Render(p.mergeCommitMessageInput.View()))
	sb.WriteString("\n")

	// Display error if present
	if p.mergeCommitState.Error != "" {
		sb.WriteString("\n")
		errStyle := lipgloss.NewStyle().Foreground(styles.Error)
		sb.WriteString(errStyle.Render("Error: " + p.mergeCommitState.Error))
	}

	sb.WriteString("\n\n")
	sb.WriteString(dimText("Enter to commit and continue • Esc to cancel"))

	content := sb.String()
	modal := modalStyle().Width(modalW).Render(content)

	// Use OverlayModal for dimmed background effect
	return ui.OverlayModal(background, modal, width, height)
}

// renderTypeSelectorModal renders the type selector modal (Shell vs Worktree).
func (p *Plugin) renderTypeSelectorModal(width, height int) string {
	background := p.renderListView(width, height)

	// Wider when Shell selected to fit name input
	modalW := 32
	if p.typeSelectorIdx == 0 {
		modalW = 44
	}
	if modalW > width-4 {
		modalW = width - 4
	}

	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Bold(true).Render("Create New"))
	sb.WriteString("\n\n")

	// Options
	options := []string{"Shell", "Workspace"}
	for i, opt := range options {
		prefix := "  "
		style := lipgloss.NewStyle().Foreground(styles.TextMuted)

		if i == p.typeSelectorIdx {
			prefix = "> "
			style = lipgloss.NewStyle().Foreground(styles.Primary).Bold(true)
		} else if i == p.typeSelectorHover {
			style = lipgloss.NewStyle().Foreground(styles.TextSecondary)
		}

		sb.WriteString(style.Render(prefix + opt))
		sb.WriteString("\n")
	}

	// Show name input when Shell is selected
	nameInputLines := 0
	if p.typeSelectorIdx == 0 {
		sb.WriteString("\n")
		nameInputLines++

		nameLabel := dimText("Name (optional):")
		sb.WriteString(nameLabel)
		sb.WriteString("\n")
		nameInputLines++

		inputW := modalW - 10
		if inputW < 20 {
			inputW = 20
		}
		p.typeSelectorNameInput.Width = inputW
		p.typeSelectorNameInput.Placeholder = p.nextShellDisplayName()

		nameStyle := inputStyle()
		if p.typeSelectorFocus == 1 {
			nameStyle = inputFocusedStyle()
		}
		sb.WriteString(nameStyle.Render(p.typeSelectorNameInput.View()))
		sb.WriteString("\n")
		nameInputLines += 3 // bordered input is 3 lines
	}

	sb.WriteString("\n")

	// Buttons
	confirmFocus := p.typeSelectorConfirmFocus()
	cancelFocus := p.typeSelectorCancelFocus()
	confirmStyle := styles.Button
	cancelStyle := styles.Button
	if p.typeSelectorFocus == confirmFocus {
		confirmStyle = styles.ButtonFocused
	} else if p.typeSelectorButtonHover == 1 {
		confirmStyle = styles.ButtonHover
	}
	if p.typeSelectorFocus == cancelFocus {
		cancelStyle = styles.ButtonFocused
	} else if p.typeSelectorButtonHover == 2 {
		cancelStyle = styles.ButtonHover
	}
	sb.WriteString(confirmStyle.Render(" Confirm "))
	sb.WriteString("  ")
	sb.WriteString(cancelStyle.Render(" Cancel "))

	content := sb.String()
	modal := modalStyle().Width(modalW).Render(content)

	// Hit regions
	modalHeight := lipgloss.Height(modal)
	modalX := (width - modalW) / 2
	modalY := (height - modalHeight) / 2

	optionStartY := modalY + 4 // border(1) + padding(1) + title(1) + blank(1)
	optionW := modalW - 6

	for i := range options {
		p.mouseHandler.HitMap.AddRect(regionTypeSelectorOption, modalX+3, optionStartY+i, optionW, 1, i)
	}

	hitX := modalX + 3
	// Name input hit region (when Shell selected)
	if p.typeSelectorIdx == 0 {
		// Name input starts after options(2) + blank(1) + label(1) = 4 lines from optionStartY
		nameInputY := optionStartY + 4
		p.mouseHandler.HitMap.AddRect(regionTypeSelectorNameInput, hitX, nameInputY, modalW-6, 3, nil)
	}

	// Buttons after options(2) + blank(1) + nameInputLines + blank(1)
	buttonY := optionStartY + 2 + nameInputLines + 1
	p.mouseHandler.HitMap.AddRect(regionTypeSelectorConfirm, hitX, buttonY, 13, 1, nil)
	cancelX := hitX + 13 + 2
	p.mouseHandler.HitMap.AddRect(regionTypeSelectorCancel, cancelX, buttonY, 12, 1, nil)

	return ui.OverlayModal(background, modal, width, height)
}
