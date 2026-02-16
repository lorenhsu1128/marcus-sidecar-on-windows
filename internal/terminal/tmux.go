//go:build !windows

package terminal

import (
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

// TmuxManager manages tmux sessions.
type TmuxManager struct {
	sessions sync.Map // name -> *TmuxSession
}

// NewTmuxManager creates a new tmux-based terminal manager.
func NewTmuxManager() *TmuxManager {
	return &TmuxManager{}
}

func (m *TmuxManager) IsAvailable() bool {
	_, err := exec.LookPath("tmux")
	return err == nil
}

func (m *TmuxManager) InstallInstructions() string {
	switch runtime.GOOS {
	case "darwin":
		return "brew install tmux"
	case "linux":
		return "sudo apt install tmux  # or: sudo dnf install tmux"
	default:
		return "Install tmux from your package manager"
	}
}

func (m *TmuxManager) CreateSession(name, workDir, cmd string, args []string) (Session, error) {
	// Create detached tmux session
	tmuxArgs := []string{"new-session", "-d", "-s", name, "-c", workDir}
	if cmd != "" {
		fullCmd := cmd
		if len(args) > 0 {
			fullCmd += " " + strings.Join(args, " ")
		}
		tmuxArgs = append(tmuxArgs, fullCmd)
	}
	if err := exec.Command("tmux", tmuxArgs...).Run(); err != nil {
		return nil, fmt.Errorf("tmux new-session: %w", err)
	}

	sess := &TmuxSession{name: name}
	m.sessions.Store(name, sess)
	return sess, nil
}

func (m *TmuxManager) GetSession(name string) Session {
	if s, ok := m.sessions.Load(name); ok {
		return s.(*TmuxSession)
	}
	// Check tmux directly
	if err := exec.Command("tmux", "has-session", "-t", name).Run(); err == nil {
		sess := &TmuxSession{name: name}
		m.sessions.Store(name, sess)
		return sess
	}
	return nil
}

func (m *TmuxManager) HasSession(name string) bool {
	return exec.Command("tmux", "has-session", "-t", name).Run() == nil
}

func (m *TmuxManager) ListSessions(prefix string) ([]string, error) {
	cmd := exec.Command("tmux", "list-sessions", "-F", "#{session_name}")
	output, err := cmd.Output()
	if err != nil {
		// No server running = no sessions
		return nil, nil
	}

	var names []string
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && strings.HasPrefix(line, prefix) {
			names = append(names, line)
		}
	}
	return names, nil
}

func (m *TmuxManager) KillSession(name string) error {
	m.sessions.Delete(name)
	return exec.Command("tmux", "kill-session", "-t", name).Run()
}

func (m *TmuxManager) SetHistoryLimit(name string, lines int) error {
	return exec.Command("tmux", "set-option", "-t", name, "history-limit", strconv.Itoa(lines)).Run()
}

func (m *TmuxManager) GetPaneID(name string) string {
	cmd := exec.Command("tmux", "list-panes", "-t", name, "-F", "#{pane_id}")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	paneID := strings.TrimSpace(string(output))
	if idx := strings.Index(paneID, "\n"); idx > 0 {
		paneID = paneID[:idx]
	}
	return paneID
}

func (m *TmuxManager) QueryPaneSize(name string) (width, height int, ok bool) {
	if name == "" {
		return 0, 0, false
	}
	cmd := exec.Command("tmux", "display-message", "-t", name, "-p", "#{pane_width},#{pane_height}")
	output, err := cmd.Output()
	if err != nil {
		return 0, 0, false
	}
	parts := strings.Split(strings.TrimSpace(string(output)), ",")
	if len(parts) < 2 {
		return 0, 0, false
	}
	width, _ = strconv.Atoi(parts[0])
	height, _ = strconv.Atoi(parts[1])
	return width, height, true
}

// TmuxSession wraps a tmux session.
type TmuxSession struct {
	name string
}

func (s *TmuxSession) ID() string { return s.name }

func (s *TmuxSession) SendKey(key string) error {
	return exec.Command("tmux", "send-keys", "-t", s.name, key).Run()
}

func (s *TmuxSession) SendLiteral(text string) error {
	// tmux treats bare ; as command separator — use hex encoding to bypass
	if strings.Contains(text, ";") {
		args := []string{"send-keys", "-t", s.name, "-H"}
		for _, b := range []byte(text) {
			args = append(args, fmt.Sprintf("%02x", b))
		}
		return exec.Command("tmux", args...).Run()
	}
	return exec.Command("tmux", "send-keys", "-l", "-t", s.name, text).Run()
}

func (s *TmuxSession) SendPaste(text string) error {
	loadCmd := exec.Command("tmux", "load-buffer", "-")
	loadCmd.Stdin = strings.NewReader(text)
	if err := loadCmd.Run(); err != nil {
		return err
	}
	return exec.Command("tmux", "paste-buffer", "-t", s.name).Run()
}

func (s *TmuxSession) SendBracketedPaste(text string) error {
	if err := s.SendLiteral("\x1b[200~"); err != nil {
		return err
	}
	if err := s.SendLiteral(text); err != nil {
		return err
	}
	return s.SendLiteral("\x1b[201~")
}

func (s *TmuxSession) SendSGRMouse(button, col, row int, release bool) error {
	if col <= 0 || row <= 0 {
		return nil
	}
	suffix := "M"
	if release {
		suffix = "m"
	}
	seq := fmt.Sprintf("\x1b[<%d;%d;%d%s", button, col, row, suffix)
	return s.SendLiteral(seq)
}

func (s *TmuxSession) CaptureOutput(scrollback int) (string, error) {
	args := []string{"capture-pane", "-p", "-e", "-t", s.name}
	if scrollback > 0 {
		args = append(args, "-S", fmt.Sprintf("-%d", scrollback))
	}
	output, err := exec.Command("tmux", args...).Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func (s *TmuxSession) QueryCursor() (row, col, paneHeight, paneWidth int, visible, ok bool) {
	cmd := exec.Command("tmux", "display-message", "-t", s.name,
		"-p", "#{cursor_x},#{cursor_y},#{cursor_flag},#{pane_height},#{pane_width}")
	output, err := cmd.Output()
	if err != nil {
		return 0, 0, 0, 0, false, false
	}
	parts := strings.Split(strings.TrimSpace(string(output)), ",")
	if len(parts) < 2 {
		return 0, 0, 0, 0, false, false
	}
	col, _ = strconv.Atoi(parts[0])
	row, _ = strconv.Atoi(parts[1])
	visible = len(parts) < 3 || parts[2] != "0"
	if len(parts) >= 4 {
		paneHeight, _ = strconv.Atoi(parts[3])
	}
	if len(parts) >= 5 {
		paneWidth, _ = strconv.Atoi(parts[4])
	}
	return row, col, paneHeight, paneWidth, visible, true
}

func (s *TmuxSession) Resize(width, height int) error {
	if width <= 0 && height <= 0 {
		return nil
	}
	// Set manual window size first
	_ = exec.Command("tmux", "set-option", "-t", s.name, "window-size", "manual").Run()

	args := []string{"resize-window", "-t", s.name}
	if width > 0 {
		args = append(args, "-x", strconv.Itoa(width))
	}
	if height > 0 {
		args = append(args, "-y", strconv.Itoa(height))
	}
	if err := exec.Command("tmux", args...).Run(); err == nil {
		return nil
	}
	// Fallback to resize-pane
	args = []string{"resize-pane", "-t", s.name}
	if width > 0 {
		args = append(args, "-x", strconv.Itoa(width))
	}
	if height > 0 {
		args = append(args, "-y", strconv.Itoa(height))
	}
	return exec.Command("tmux", args...).Run()
}

func (s *TmuxSession) IsAlive() bool {
	return exec.Command("tmux", "has-session", "-t", s.name).Run() == nil
}

func (s *TmuxSession) Kill() error {
	return exec.Command("tmux", "kill-session", "-t", s.name).Run()
}
