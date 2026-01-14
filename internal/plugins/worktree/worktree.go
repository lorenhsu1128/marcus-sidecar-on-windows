package worktree

import (
	"bufio"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// refreshWorktrees returns a command to refresh the worktree list.
func (p *Plugin) refreshWorktrees() tea.Cmd {
	return func() tea.Msg {
		worktrees, err := p.listWorktrees()
		return RefreshDoneMsg{Worktrees: worktrees, Err: err}
	}
}

// listWorktrees parses git worktree list --porcelain output.
func (p *Plugin) listWorktrees() ([]*Worktree, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = p.ctx.WorkDir
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git worktree list: %w", err)
	}

	return parseWorktreeList(string(output), p.ctx.WorkDir)
}

// parseWorktreeList parses porcelain format output.
func parseWorktreeList(output, mainWorkdir string) ([]*Worktree, error) {
	var worktrees []*Worktree
	var current *Worktree

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "worktree ") {
			if current != nil {
				worktrees = append(worktrees, current)
			}
			path := strings.TrimPrefix(line, "worktree ")
			name := filepath.Base(path)
			// Skip main worktree (where git repo lives)
			if path == mainWorkdir {
				current = nil
				continue
			}
			current = &Worktree{
				Name:      name,
				Path:      path,
				Status:    StatusPaused,
				CreatedAt: time.Now(), // Will be updated from file stat
			}
		} else if current != nil {
			if strings.HasPrefix(line, "HEAD ") {
				// HEAD commit hash - not storing currently
			} else if strings.HasPrefix(line, "branch ") {
				branch := strings.TrimPrefix(line, "branch refs/heads/")
				current.Branch = branch
			} else if line == "bare" {
				// Bare worktree
			} else if line == "detached" {
				current.Branch = "(detached)"
			}
		}
	}

	if current != nil {
		worktrees = append(worktrees, current)
	}

	return worktrees, scanner.Err()
}

// createWorktree returns a command to create a new worktree.
func (p *Plugin) createWorktree() tea.Cmd {
	name := p.createName
	baseBranch := p.createBaseBranch
	taskID := p.createTaskID

	if name == "" {
		return func() tea.Msg {
			return CreateDoneMsg{Err: fmt.Errorf("worktree name is required")}
		}
	}

	return func() tea.Msg {
		wt, err := p.doCreateWorktree(name, baseBranch, taskID)
		return CreateDoneMsg{Worktree: wt, Err: err}
	}
}

// doCreateWorktree performs the actual worktree creation.
func (p *Plugin) doCreateWorktree(name, baseBranch, taskID string) (*Worktree, error) {
	// Default base branch to current branch if not specified
	if baseBranch == "" {
		baseBranch = "HEAD"
	}

	// Determine worktree path (sibling to main repo)
	parentDir := filepath.Dir(p.ctx.WorkDir)
	wtPath := filepath.Join(parentDir, name)

	// Create worktree with new branch
	args := []string{"worktree", "add", "-b", name, wtPath, baseBranch}
	cmd := exec.Command("git", args...)
	cmd.Dir = p.ctx.WorkDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("git worktree add: %s: %w", strings.TrimSpace(string(output)), err)
	}

	// Determine actual base branch name
	actualBase := baseBranch
	if baseBranch == "HEAD" {
		if b, err := getCurrentBranch(p.ctx.WorkDir); err == nil {
			actualBase = b
		}
	}

	wt := &Worktree{
		Name:       name,
		Path:       wtPath,
		Branch:     name,
		BaseBranch: actualBase,
		TaskID:     taskID,
		Status:     StatusPaused,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	return wt, nil
}

// deleteSelected returns a command to delete the selected worktree.
func (p *Plugin) deleteSelected() tea.Cmd {
	wt := p.selectedWorktree()
	if wt == nil {
		return nil
	}
	name := wt.Name
	path := wt.Path

	return func() tea.Msg {
		err := doDeleteWorktree(path)
		return DeleteDoneMsg{Name: name, Err: err}
	}
}

// doDeleteWorktree removes a worktree.
func doDeleteWorktree(path string) error {
	// First try without force
	cmd := exec.Command("git", "worktree", "remove", path)
	if err := cmd.Run(); err == nil {
		return nil
	}

	// If that fails, try with force
	cmd = exec.Command("git", "worktree", "remove", "--force", path)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git worktree remove: %s: %w", strings.TrimSpace(string(output)), err)
	}

	return nil
}

// pushSelected returns a command to push the selected worktree's branch.
func (p *Plugin) pushSelected() tea.Cmd {
	wt := p.selectedWorktree()
	if wt == nil {
		return nil
	}
	name := wt.Name
	path := wt.Path
	branch := wt.Branch

	return func() tea.Msg {
		err := doPush(path, branch, false, true)
		return PushDoneMsg{WorktreeName: name, Err: err}
	}
}

// doPush pushes a branch to remote.
func doPush(workdir, branch string, force, setUpstream bool) error {
	args := []string{"push"}
	if force {
		args = append(args, "--force-with-lease")
	}
	if setUpstream {
		args = append(args, "-u", "origin", branch)
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = workdir
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git push: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

// getCurrentBranch returns the current branch name.
func getCurrentBranch(workdir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = workdir
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}
