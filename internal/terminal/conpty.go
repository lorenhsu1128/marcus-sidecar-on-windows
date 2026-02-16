//go:build windows

package terminal

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"

	"github.com/charmbracelet/x/conpty"
)

// ConPTYManager manages ConPTY sessions on Windows.
type ConPTYManager struct {
	sessions sync.Map // name -> *ConPTYSession
}

// NewConPTYManager creates a new ConPTY-based terminal manager.
func NewConPTYManager() *ConPTYManager {
	return &ConPTYManager{}
}

func (m *ConPTYManager) IsAvailable() bool {
	return true // ConPTY is available on Windows 10 1809+
}

func (m *ConPTYManager) InstallInstructions() string {
	return "ConPTY requires Windows 10 version 1809 or later."
}

func (m *ConPTYManager) CreateSession(name, workDir, cmd string, args []string) (Session, error) {
	// Determine shell command
	if cmd == "" {
		cmd = defaultWindowsShell()
	}

	sess, err := newConPTYSession(name, workDir, cmd, args)
	if err != nil {
		return nil, err
	}

	m.sessions.Store(name, sess)
	return sess, nil
}

func (m *ConPTYManager) GetSession(name string) Session {
	if s, ok := m.sessions.Load(name); ok {
		sess := s.(*ConPTYSession)
		if sess.IsAlive() {
			return sess
		}
		m.sessions.Delete(name)
	}
	return nil
}

func (m *ConPTYManager) HasSession(name string) bool {
	return m.GetSession(name) != nil
}

func (m *ConPTYManager) ListSessions(prefix string) ([]string, error) {
	var names []string
	m.sessions.Range(func(key, value any) bool {
		name := key.(string)
		sess := value.(*ConPTYSession)
		if strings.HasPrefix(name, prefix) && sess.IsAlive() {
			names = append(names, name)
		}
		return true
	})
	return names, nil
}

func (m *ConPTYManager) KillSession(name string) error {
	if s, ok := m.sessions.LoadAndDelete(name); ok {
		return s.(*ConPTYSession).Kill()
	}
	return nil
}

func (m *ConPTYManager) SetHistoryLimit(name string, lines int) error {
	if s, ok := m.sessions.Load(name); ok {
		s.(*ConPTYSession).historyLimit = lines
		return nil
	}
	return fmt.Errorf("session %s not found", name)
}

func (m *ConPTYManager) GetPaneID(name string) string {
	// For ConPTY, the session name itself serves as the unique identifier.
	if m.GetSession(name) != nil {
		return name
	}
	return ""
}

func (m *ConPTYManager) QueryPaneSize(name string) (width, height int, ok bool) {
	if s, ok := m.sessions.Load(name); ok {
		sess := s.(*ConPTYSession)
		w, h, _ := sess.pty.Size()
		return w, h, true
	}
	return 0, 0, false
}

// ConPTYSession wraps a Windows ConPTY pseudo-terminal.
type ConPTYSession struct {
	name         string
	pty          *conpty.ConPty
	pid          int
	procHandle   uintptr
	historyLimit int

	// Output buffer: background goroutine reads from pty and appends to buffer
	mu     sync.Mutex
	buf    []byte   // raw accumulated output
	lines  []string // parsed lines cache (invalidated on new data)
	dirty  bool     // lines need re-parsing
	closed bool

	// Cursor tracking (estimated from VT sequences)
	cursorRow int
	cursorCol int
}

func newConPTYSession(name, workDir, cmd string, args []string) (*ConPTYSession, error) {
	pty, err := conpty.New(80, 25, 0)
	if err != nil {
		return nil, fmt.Errorf("conpty.New: %w", err)
	}

	// Build command line
	allArgs := append([]string{cmd}, args...)

	pid, handle, err := pty.Spawn(cmd, allArgs, &syscall.ProcAttr{
		Dir: workDir,
		Env: os.Environ(),
	})
	if err != nil {
		pty.Close()
		return nil, fmt.Errorf("conpty spawn %s: %w", cmd, err)
	}

	sess := &ConPTYSession{
		name:         name,
		pty:          pty,
		pid:          pid,
		procHandle:   handle,
		historyLimit: 10000,
	}

	// Start background output reader
	go sess.readLoop()

	return sess, nil
}

// readLoop continuously reads output from the ConPTY.
func (s *ConPTYSession) readLoop() {
	buf := make([]byte, 4096)
	for {
		n, err := s.pty.Read(buf)
		if n > 0 {
			s.mu.Lock()
			s.buf = append(s.buf, buf[:n]...)
			s.dirty = true
			// Trim buffer if too large (keep last ~1MB)
			const maxBuf = 1 << 20
			if len(s.buf) > maxBuf {
				// Find a newline boundary to trim at
				trimAt := len(s.buf) - maxBuf
				for trimAt < len(s.buf) && s.buf[trimAt] != '\n' {
					trimAt++
				}
				if trimAt < len(s.buf) {
					s.buf = s.buf[trimAt+1:]
				}
			}
			s.mu.Unlock()
		}
		if err != nil {
			s.mu.Lock()
			s.closed = true
			s.mu.Unlock()
			return
		}
	}
}

func (s *ConPTYSession) ID() string { return s.name }

func (s *ConPTYSession) SendKey(key string) error {
	// Convert tmux-style key names to VT sequences
	seq := keyNameToVT(key)
	_, err := s.pty.Write([]byte(seq))
	return err
}

func (s *ConPTYSession) SendLiteral(text string) error {
	_, err := s.pty.Write([]byte(text))
	return err
}

func (s *ConPTYSession) SendPaste(text string) error {
	_, err := s.pty.Write([]byte(text))
	return err
}

func (s *ConPTYSession) SendBracketedPaste(text string) error {
	full := "\x1b[200~" + text + "\x1b[201~"
	_, err := s.pty.Write([]byte(full))
	return err
}

func (s *ConPTYSession) SendSGRMouse(button, col, row int, release bool) error {
	if col <= 0 || row <= 0 {
		return nil
	}
	suffix := "M"
	if release {
		suffix = "m"
	}
	seq := fmt.Sprintf("\x1b[<%d;%d;%d%s", button, col, row, suffix)
	_, err := s.pty.Write([]byte(seq))
	return err
}

func (s *ConPTYSession) CaptureOutput(scrollback int) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.dirty {
		s.lines = strings.Split(string(s.buf), "\n")
		s.dirty = false
	}

	// Return the last N lines to match tmux capture-pane behavior
	total := len(s.lines)
	if scrollback > 0 && total > scrollback {
		return strings.Join(s.lines[total-scrollback:], "\n"), nil
	}
	return string(s.buf), nil
}

func (s *ConPTYSession) QueryCursor() (row, col, paneHeight, paneWidth int, visible, ok bool) {
	w, h, _ := s.pty.Size()
	s.mu.Lock()
	r, c := s.cursorRow, s.cursorCol
	s.mu.Unlock()
	return r, c, h, w, true, true
}

func (s *ConPTYSession) Resize(width, height int) error {
	return s.pty.Resize(width, height)
}

func (s *ConPTYSession) IsAlive() bool {
	s.mu.Lock()
	closed := s.closed
	s.mu.Unlock()
	if closed {
		return false
	}
	// Check if process is still running
	handle := syscall.Handle(s.procHandle)
	var exitCode uint32
	err := syscall.GetExitCodeProcess(handle, &exitCode)
	if err != nil {
		return false
	}
	return exitCode == 259 // STILL_ACTIVE
}

func (s *ConPTYSession) Kill() error {
	s.mu.Lock()
	s.closed = true
	s.mu.Unlock()

	// Terminate process
	handle := syscall.Handle(s.procHandle)
	_ = syscall.TerminateProcess(handle, 1)
	_ = syscall.CloseHandle(handle)

	return s.pty.Close()
}

// defaultWindowsShell returns the default shell on Windows.
func defaultWindowsShell() string {
	if ps, err := exec.LookPath("pwsh.exe"); err == nil {
		return ps
	}
	return "powershell.exe"
}

// keyNameToVT converts tmux-style key names to VT escape sequences.
func keyNameToVT(key string) string {
	switch key {
	case "Enter":
		return "\r"
	case "Escape":
		return "\x1b"
	case "Tab":
		return "\t"
	case "BSpace":
		return "\x7f"
	case "Space":
		return " "
	case "Up":
		return "\x1b[A"
	case "Down":
		return "\x1b[B"
	case "Right":
		return "\x1b[C"
	case "Left":
		return "\x1b[D"
	case "Home":
		return "\x1b[H"
	case "End":
		return "\x1b[F"
	case "DC": // Delete
		return "\x1b[3~"
	case "IC": // Insert
		return "\x1b[2~"
	case "PgUp", "PageUp":
		return "\x1b[5~"
	case "PgDn", "PageDown":
		return "\x1b[6~"
	case "C-c":
		return "\x03"
	case "C-d":
		return "\x04"
	case "C-z":
		return "\x1a"
	case "C-l":
		return "\x0c"
	case "C-a":
		return "\x01"
	case "C-e":
		return "\x05"
	case "C-k":
		return "\x0b"
	case "C-u":
		return "\x15"
	case "C-w":
		return "\x17"
	case "C-r":
		return "\x12"
	case "C-p":
		return "\x10"
	case "C-n":
		return "\x0e"
	default:
		// F-keys
		if strings.HasPrefix(key, "F") {
			switch key {
			case "F1":
				return "\x1bOP"
			case "F2":
				return "\x1bOQ"
			case "F3":
				return "\x1bOR"
			case "F4":
				return "\x1bOS"
			case "F5":
				return "\x1b[15~"
			case "F6":
				return "\x1b[17~"
			case "F7":
				return "\x1b[18~"
			case "F8":
				return "\x1b[19~"
			case "F9":
				return "\x1b[20~"
			case "F10":
				return "\x1b[21~"
			case "F11":
				return "\x1b[23~"
			case "F12":
				return "\x1b[24~"
			}
		}
		// Single character or unknown — pass through
		return key
	}
}
