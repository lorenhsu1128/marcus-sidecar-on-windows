package plugin

import (
	"log/slog"

	"github.com/lorenhsu1128/marcus-sidecar-on-windows/internal/adapter"
	"github.com/lorenhsu1128/marcus-sidecar-on-windows/internal/config"
	"github.com/lorenhsu1128/marcus-sidecar-on-windows/internal/event"
	"github.com/lorenhsu1128/marcus-sidecar-on-windows/internal/terminal"
)

// BindingRegistrar allows plugins to register key bindings dynamically.
// This is implemented by keymap.Registry.
type BindingRegistrar interface {
	RegisterPluginBinding(key, command, context string)
}

// Context provides shared resources to plugins during initialization.
type Context struct {
	WorkDir     string // Actual working directory (worktree path for linked worktrees)
	ProjectRoot string // Main repo root for shared state (same as WorkDir for non-worktrees)
	ConfigDir   string
	Config    *config.Config
	Adapters  map[string]adapter.Adapter
	EventBus  *event.Dispatcher
	Logger    *slog.Logger
	Keymap    BindingRegistrar // For plugins to register dynamic bindings
	Terminal  terminal.Manager // Cross-platform terminal session manager
	Epoch     uint64           // Incremented on project switch to invalidate stale async messages
}
