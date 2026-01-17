# Capturing Sidecar Screenshots

This guide explains how to capture screenshots of Sidecar for documentation purposes.

## Quick Method: Manual Capture

The simplest approach is using macOS built-in screenshot tools:

1. Run `sidecar` in your terminal
2. Press `Cmd+Shift+4` then `Space` to capture the window
3. Click on the terminal window to capture

## Automated Method: Agent-Assisted Capture

When working with an AI agent (like Warp Agent Mode), use this two-step approach:

### Step 1: Start the Screenshot Timer

In one terminal, start the background screenshot timer:

```bash
./scripts/screenshot-timer.sh 2 5 sidecar-docs
```

This takes 5 screenshots at 2-second intervals with the prefix `sidecar-docs`.

### Step 2: Run Sidecar via Agent

Have the agent run sidecar and navigate through views. The background timer will capture each state.

## Using the Expect Script

For fully automated captures (useful in CI/pipelines), use the expect script:

```bash
./scripts/capture-screenshots.exp
```

**Note:** This captures full-screen screenshots, not just the terminal window. It works best when Warp is the only visible application.

## Screenshot Naming Convention

Screenshots should follow this naming pattern:
- `sidecar-{plugin}.png` - Main view of a plugin (td, git, files, conversations)
- `sidecar-{plugin}-{feature}.png` - Specific feature view (e.g., `sidecar-git-diff-side-by-side.png`)

## Tips

- Use `screencapture -x` flag to silence the capture sound
- Use `screencapture -o` to exclude window shadow
- For window-specific capture: `screencapture -l <window_id>` (get ID via AppleScript)

## Scripts

| Script | Purpose |
|--------|---------|
| `scripts/screenshot-timer.sh` | Takes screenshots at intervals (for agent workflow) |
| `scripts/capture-screenshots.exp` | Automated full capture via expect |
