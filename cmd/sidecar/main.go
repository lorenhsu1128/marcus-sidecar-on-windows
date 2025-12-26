package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sst/sidecar/internal/adapter"
	"github.com/sst/sidecar/internal/adapter/claudecode"
	"github.com/sst/sidecar/internal/app"
	"github.com/sst/sidecar/internal/config"
	"github.com/sst/sidecar/internal/event"
	"github.com/sst/sidecar/internal/keymap"
	"github.com/sst/sidecar/internal/plugin"
	"github.com/sst/sidecar/internal/plugins/conversations"
	"github.com/sst/sidecar/internal/plugins/gitstatus"
	"github.com/sst/sidecar/internal/plugins/tdmonitor"
)

// Version is set at build time via ldflags
var Version = ""

var (
	configPath   = flag.String("config", "", "path to config file")
	projectRoot  = flag.String("project", ".", "project root directory")
	debugFlag    = flag.Bool("debug", false, "enable debug logging")
	versionFlag  = flag.Bool("version", false, "print version and exit")
	shortVersion = flag.Bool("v", false, "print version and exit (short)")
)

func main() {
	flag.Parse()

	// Handle version flag
	if *versionFlag || *shortVersion {
		fmt.Printf("sidecar version %s\n", effectiveVersion(Version))
		os.Exit(0)
	}

	// Setup logging
	logLevel := slog.LevelInfo
	if *debugFlag {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	}))

	// Load configuration
	cfg, err := loadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Create event dispatcher
	dispatcher := event.NewWithLogger(logger)
	defer dispatcher.Close()

	// Create plugin context
	pluginCtx := &plugin.Context{
		WorkDir:   *projectRoot,
		ConfigDir: config.ConfigPath(),
		Adapters:  make(map[string]adapter.Adapter),
		EventBus:  dispatcher,
		Logger:    logger,
	}

	// Register adapters
	ccAdapter := claudecode.New()
	pluginCtx.Adapters["claude-code"] = ccAdapter

	// Create plugin registry
	registry := plugin.NewRegistry(pluginCtx)

	// Register plugins
	registry.Register(gitstatus.New())
	registry.Register(tdmonitor.New())
	registry.Register(conversations.New())

	// Create keymap registry
	km := keymap.NewRegistry()
	keymap.RegisterDefaults(km)

	// Apply user keymap overrides
	for key, cmdID := range cfg.Keymap.Overrides {
		km.SetUserOverride(key, cmdID)
	}

	// Create and run application
	model := app.New(registry, km)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running application: %v\n", err)
		os.Exit(1)
	}
}

func loadConfig(path string) (*config.Config, error) {
	if path != "" {
		return config.LoadFrom(path)
	}
	return config.Load()
}

// effectiveVersion returns the version string, with fallback to build info.
func effectiveVersion(v string) string {
	if v != "" {
		return v
	}

	// Try to get version from Go build info
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}

	// Check module version
	if info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}

	// Fall back to VCS info
	var revision string
	var dirty bool

	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.modified":
			dirty = setting.Value == "true"
		}
	}

	if revision != "" {
		ver := "devel+" + revision
		if len(ver) > 20 {
			ver = ver[:20]
		}
		if dirty {
			ver += "+dirty"
		}
		return ver
	}

	return "devel"
}

// getShortRevision returns the first 12 chars of a revision.
func getShortRevision(rev string) string {
	if len(rev) > 12 {
		return rev[:12]
	}
	return rev
}

func init() {
	// Customize usage output
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: sidecar [options]\n\n")
		fmt.Fprintf(os.Stderr, "A TUI dashboard for AI coding agents.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}
}

// Ensure strings import is used
var _ = strings.TrimSpace
