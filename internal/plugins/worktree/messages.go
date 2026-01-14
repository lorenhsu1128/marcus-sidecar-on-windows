package worktree

// RefreshMsg triggers a worktree list refresh.
type RefreshMsg struct{}

// RefreshDoneMsg signals that refresh has completed.
type RefreshDoneMsg struct {
	Worktrees []*Worktree
	Err       error
}

// WatchEventMsg signals a filesystem change was detected.
type WatchEventMsg struct {
	Path string
}

// WatcherStartedMsg signals the file watcher is running.
type WatcherStartedMsg struct{}

// WatcherErrorMsg signals a file watcher error.
type WatcherErrorMsg struct {
	Err error
}

// AgentOutputMsg delivers new agent output.
type AgentOutputMsg struct {
	WorktreeName string
	Output       string
	Status       WorktreeStatus
	WaitingFor   string
}

// AgentStoppedMsg signals an agent has stopped.
type AgentStoppedMsg struct {
	WorktreeName string
	Err          error
}

// TmuxAttachFinishedMsg signals return from tmux attach.
type TmuxAttachFinishedMsg struct {
	WorktreeName string
	Err          error
}

// DiffLoadedMsg delivers diff content for a worktree.
type DiffLoadedMsg struct {
	WorktreeName string
	Content      string
	Raw          string
}

// DiffErrorMsg signals diff loading failed.
type DiffErrorMsg struct {
	WorktreeName string
	Err          error
}

// StatsLoadedMsg delivers git stats for a worktree.
type StatsLoadedMsg struct {
	WorktreeName string
	Stats        *GitStats
}

// StatsErrorMsg signals stats loading failed.
type StatsErrorMsg struct {
	WorktreeName string
	Err          error
}

// CreateWorktreeMsg requests worktree creation.
type CreateWorktreeMsg struct {
	Name       string
	BaseBranch string
	TaskID     string
}

// CreateDoneMsg signals worktree creation completed.
type CreateDoneMsg struct {
	Worktree *Worktree
	Err      error
}

// DeleteWorktreeMsg requests worktree deletion.
type DeleteWorktreeMsg struct {
	Name  string
	Force bool
}

// DeleteDoneMsg signals worktree deletion completed.
type DeleteDoneMsg struct {
	Name string
	Err  error
}

// PushMsg requests pushing a worktree branch.
type PushMsg struct {
	WorktreeName string
	Force        bool
	SetUpstream  bool
}

// PushDoneMsg signals push operation completed.
type PushDoneMsg struct {
	WorktreeName string
	Err          error
}

// TaskSearchResultsMsg delivers task search results.
type TaskSearchResultsMsg struct {
	Tasks []Task
	Err   error
}

// TaskLinkedMsg signals a task was linked to a worktree.
type TaskLinkedMsg struct {
	WorktreeName string
	TaskID       string
	Err          error
}

// Task represents a TD task for linking.
type Task struct {
	ID          string
	Title       string
	Status      string
	Description string
}
