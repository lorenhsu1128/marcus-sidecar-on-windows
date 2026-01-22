package worktree

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestSanitizeName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"with.dot", "with-dot"},
		{"with:colon", "with-colon"},
		{"with/slash", "with-slash"},
		{"multi.dot:colon/slash", "multi-dot-colon-slash"},
		{"already-clean", "already-clean"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeName(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFindWorktreeBySanitizedName(t *testing.T) {
	tests := []struct {
		name          string
		worktrees     []*Worktree
		sanitizedName string
		expectName    string // empty if no match expected
	}{
		{
			name: "exact match",
			worktrees: []*Worktree{
				{Name: "simple-name"},
				{Name: "other-name"},
			},
			sanitizedName: "simple-name",
			expectName:    "simple-name",
		},
		{
			name: "match with dot sanitized",
			worktrees: []*Worktree{
				{Name: "feature.branch"},
				{Name: "other-name"},
			},
			sanitizedName: "feature-branch",
			expectName:    "feature.branch",
		},
		{
			name: "match with slash sanitized",
			worktrees: []*Worktree{
				{Name: "feature/auth/login"},
				{Name: "other-name"},
			},
			sanitizedName: "feature-auth-login",
			expectName:    "feature/auth/login",
		},
		{
			name: "match with colon sanitized",
			worktrees: []*Worktree{
				{Name: "fix:bug:123"},
				{Name: "other-name"},
			},
			sanitizedName: "fix-bug-123",
			expectName:    "fix:bug:123",
		},
		{
			name: "match with multiple special chars",
			worktrees: []*Worktree{
				{Name: "feature/v1.0:hotfix"},
				{Name: "other-name"},
			},
			sanitizedName: "feature-v1-0-hotfix",
			expectName:    "feature/v1.0:hotfix",
		},
		{
			name: "long name with DirPrefix",
			worktrees: []*Worktree{
				{Name: "sidecar-td-c92aa56d-conversations-yank-add-y-y-key-bindings"},
			},
			sanitizedName: "sidecar-td-c92aa56d-conversations-yank-add-y-y-key-bindings",
			expectName:    "sidecar-td-c92aa56d-conversations-yank-add-y-y-key-bindings",
		},
		{
			name: "no match",
			worktrees: []*Worktree{
				{Name: "existing-wt"},
			},
			sanitizedName: "nonexistent-wt",
			expectName:    "",
		},
		{
			name:          "empty worktrees",
			worktrees:     []*Worktree{},
			sanitizedName: "any-name",
			expectName:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Plugin{worktrees: tt.worktrees}
			result := p.findWorktreeBySanitizedName(tt.sanitizedName)

			if tt.expectName == "" {
				if result != nil {
					t.Errorf("findWorktreeBySanitizedName(%q) = %q, want nil", tt.sanitizedName, result.Name)
				}
			} else {
				if result == nil {
					t.Errorf("findWorktreeBySanitizedName(%q) = nil, want %q", tt.sanitizedName, tt.expectName)
				} else if result.Name != tt.expectName {
					t.Errorf("findWorktreeBySanitizedName(%q) = %q, want %q", tt.sanitizedName, result.Name, tt.expectName)
				}
			}
		})
	}
}

func TestGetAgentCommand(t *testing.T) {
	tests := []struct {
		agentType AgentType
		expected  string
	}{
		{AgentClaude, "claude"},
		{AgentCodex, "codex"},
		{AgentAider, "aider"},
		{AgentGemini, "gemini"},
		{AgentCursor, "cursor-agent"},
		{AgentOpenCode, "opencode"},
		{AgentCustom, "claude"}, // Falls back to claude
	}

	for _, tt := range tests {
		t.Run(string(tt.agentType), func(t *testing.T) {
			result := getAgentCommand(tt.agentType)
			if result != tt.expected {
				t.Errorf("getAgentCommand(%q) = %q, want %q", tt.agentType, result, tt.expected)
			}
		})
	}
}

func TestDetectStatus(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected WorktreeStatus
	}{
		{
			name:     "waiting for y/n prompt",
			output:   "Some output\nDo you want to continue? [y/n]",
			expected: StatusWaiting,
		},
		{
			name:     "waiting for y/n in parentheses",
			output:   "Some output\nProceed? (y/n):",
			expected: StatusWaiting,
		},
		{
			name:     "allow edit prompt",
			output:   "Claude wants to edit file.go\nAllow edit? [y/n]",
			expected: StatusWaiting,
		},
		{
			name:     "allow bash prompt",
			output:   "Command: rm -rf /tmp/foo\nAllow bash? [y/n]",
			expected: StatusWaiting,
		},
		{
			name:     "approve prompt",
			output:   "Please approve this change",
			expected: StatusWaiting,
		},
		{
			name:     "task completed",
			output:   "All changes applied\nTask completed successfully",
			expected: StatusDone,
		},
		{
			name:     "finished",
			output:   "Output\nFinished processing",
			expected: StatusDone,
		},
		{
			name:     "error detected",
			output:   "Error: file not found",
			expected: StatusError,
		},
		{
			name:     "failed",
			output:   "Build failed with 3 errors",
			expected: StatusError,
		},
		{
			name:     "traceback",
			output:   "Traceback (most recent call last):\n  File...",
			expected: StatusError,
		},
		{
			name:     "normal active output",
			output:   "Processing files...\nCompiling main.go",
			expected: StatusActive,
		},
		{
			name:     "empty output",
			output:   "",
			expected: StatusActive,
		},
		{
			name:     "claude code prompt symbol",
			output:   "Some output\n❯",
			expected: StatusWaiting,
		},
		{
			name:     "claude code prompt with tree line",
			output:   "Some output\n╰─❯",
			expected: StatusWaiting,
		},
		{
			name:     "claude code prompt in multiline output",
			output:   "Processing complete\nChanges applied successfully\n\n❯",
			expected: StatusWaiting,
		},
		// Thinking status tests
		{
			name:     "thinking with claude extended thinking tag",
			output:   "Let me analyze this\n<thinking>considering the options",
			expected: StatusThinking,
		},
		{
			name:     "thinking with internal monologue",
			output:   "Processing\n<internal_monologue>evaluating approach",
			expected: StatusThinking,
		},
		{
			name:     "thinking with generic indicator",
			output:   "Working on it\nthinking... processing request",
			expected: StatusThinking,
		},
		{
			name:     "thinking with aider reasoning",
			output:   "Aider output\nreasoning about the implementation",
			expected: StatusThinking,
		},
		{
			name:     "closed thinking tag should be active not thinking",
			output:   "<thinking>analyzed</thinking>\nNow implementing the fix",
			expected: StatusActive,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectStatus(tt.output)
			if result != tt.expected {
				t.Errorf("detectStatus(%q) = %v, want %v", tt.output, result, tt.expected)
			}
		})
	}
}

func TestExtractPrompt(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected string
	}{
		{
			name:     "y/n prompt",
			output:   "Some context\nDo you want to continue? [y/n]",
			expected: "Do you want to continue? [y/n]",
		},
		{
			name:     "allow edit prompt",
			output:   "Multiple lines\nof output\nAllow edit file.go? [y/n]",
			expected: "Allow edit file.go? [y/n]",
		},
		{
			name:     "approve prompt",
			output:   "Changes:\n- foo\n- bar\nApprove these changes?",
			expected: "Approve these changes?",
		},
		{
			name:     "no prompt",
			output:   "Just normal output\nnothing special",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractPrompt(tt.output)
			if result != tt.expected {
				t.Errorf("extractPrompt() = %q, want %q", result, tt.expected)
			}
		})
	}
}


func TestDetectStatusPriorityOrder(t *testing.T) {
	// Waiting should take priority over error when both patterns present
	output := "Error occurred\nRetry? [y/n]"
	result := detectStatus(output)
	if result != StatusWaiting {
		t.Errorf("waiting should take priority over error, got %v", result)
	}

	// Waiting should take priority over thinking when both patterns present
	output2 := "<thinking>analyzing\nDo you want to proceed? [y/n]"
	result2 := detectStatus(output2)
	if result2 != StatusWaiting {
		t.Errorf("waiting should take priority over thinking, got %v", result2)
	}

	// Thinking should take priority over error
	output3 := "<thinking>analyzing the error\nError: something went wrong"
	result3 := detectStatus(output3)
	if result3 != StatusThinking {
		t.Errorf("thinking should take priority over error, got %v", result3)
	}
}

func TestTmuxSessionPrefix(t *testing.T) {
	// Verify the session prefix constant
	if !strings.HasPrefix(tmuxSessionPrefix, "sidecar-") {
		t.Errorf("tmux session prefix should start with 'sidecar-', got %q", tmuxSessionPrefix)
	}
}

func TestPaneCacheCleanup(t *testing.T) {
	// Create a test cache with short TTL
	cache := &paneCache{
		entries: make(map[string]paneCacheEntry),
		ttl:     10 * time.Millisecond,
	}

	// Add entries
	cache.entries["session1"] = paneCacheEntry{output: "output1", timestamp: time.Now()}
	cache.entries["session2"] = paneCacheEntry{output: "output2", timestamp: time.Now()}

	// Verify entries exist
	if len(cache.entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(cache.entries))
	}

	// Wait for TTL to expire
	time.Sleep(15 * time.Millisecond)

	// Cleanup should remove expired entries
	cache.cleanup()

	if len(cache.entries) != 0 {
		t.Errorf("cleanup should remove expired entries, got %d remaining", len(cache.entries))
	}
}

func TestPaneCacheRemove(t *testing.T) {
	cache := &paneCache{
		entries: make(map[string]paneCacheEntry),
		ttl:     1 * time.Hour, // Long TTL
	}

	// Add entry
	cache.entries["session1"] = paneCacheEntry{output: "output1", timestamp: time.Now()}

	// Remove it
	cache.remove("session1")

	if _, exists := cache.entries["session1"]; exists {
		t.Error("remove should delete the entry")
	}
}

func TestPaneCacheGetRemovesExpired(t *testing.T) {
	cache := &paneCache{
		entries: make(map[string]paneCacheEntry),
		ttl:     10 * time.Millisecond,
	}

	// Add expired entry
	cache.entries["session1"] = paneCacheEntry{
		output:    "output1",
		timestamp: time.Now().Add(-20 * time.Millisecond), // Already expired
	}

	// Get should return empty and remove the entry
	output, ok := cache.get("session1")
	if ok {
		t.Error("get should return false for expired entry")
	}
	if output != "" {
		t.Errorf("get should return empty string for expired, got %q", output)
	}

	// Entry should be removed
	if _, exists := cache.entries["session1"]; exists {
		t.Error("get should remove expired entries")
	}
}

func TestPaneCacheSetAllRemovesStale(t *testing.T) {
	cache := &paneCache{
		entries: make(map[string]paneCacheEntry),
		ttl:     1 * time.Hour,
	}

	// Add initial entries
	cache.entries["old-session"] = paneCacheEntry{output: "old", timestamp: time.Now()}
	cache.entries["kept-session"] = paneCacheEntry{output: "kept", timestamp: time.Now()}

	// SetAll with new batch (only kept-session)
	cache.setAll(map[string]string{
		"kept-session": "new-kept",
		"new-session":  "new",
	})

	// old-session should be removed
	if _, exists := cache.entries["old-session"]; exists {
		t.Error("setAll should remove entries not in new batch")
	}

	// kept-session and new-session should exist
	if _, exists := cache.entries["kept-session"]; !exists {
		t.Error("setAll should keep entries that are in new batch")
	}
	if _, exists := cache.entries["new-session"]; !exists {
		t.Error("setAll should add new entries")
	}
}

func TestShouldShowSkipPermissions(t *testing.T) {
	tests := []struct {
		agentType AgentType
		expected  bool
	}{
		{AgentNone, false},     // No agent, no checkbox
		{AgentClaude, true},    // Has --dangerously-skip-permissions
		{AgentCodex, true},     // Has --dangerously-bypass-approvals-and-sandbox
		{AgentGemini, true},    // Has --yolo
		{AgentCursor, true},    // Has -f flag
		{AgentOpenCode, false}, // No known flag
	}

	p := &Plugin{}
	for _, tt := range tests {
		t.Run(string(tt.agentType), func(t *testing.T) {
			p.createAgentType = tt.agentType
			result := p.shouldShowSkipPermissions()
			if result != tt.expected {
				t.Errorf("shouldShowSkipPermissions(%q) = %v, want %v", tt.agentType, result, tt.expected)
			}
		})
	}
}

func TestBuildAgentCommand(t *testing.T) {
	tests := []struct {
		name      string
		agentType AgentType
		skipPerms bool
		taskID    string
		wantFlag  string   // Expected skip-perms flag in output
		wantPrompt bool    // Whether prompt should be included
	}{
		// Claude tests
		{
			name:       "claude no skip no task",
			agentType:  AgentClaude,
			skipPerms:  false,
			taskID:     "",
			wantFlag:   "",
			wantPrompt: false,
		},
		{
			name:       "claude with skip no task",
			agentType:  AgentClaude,
			skipPerms:  true,
			taskID:     "",
			wantFlag:   "--dangerously-skip-permissions",
			wantPrompt: false,
		},
		// Codex tests
		{
			name:       "codex no skip no task",
			agentType:  AgentCodex,
			skipPerms:  false,
			taskID:     "",
			wantFlag:   "",
			wantPrompt: false,
		},
		{
			name:       "codex with skip no task",
			agentType:  AgentCodex,
			skipPerms:  true,
			taskID:     "",
			wantFlag:   "--dangerously-bypass-approvals-and-sandbox",
			wantPrompt: false,
		},
		// Gemini tests
		{
			name:       "gemini no skip no task",
			agentType:  AgentGemini,
			skipPerms:  false,
			taskID:     "",
			wantFlag:   "",
			wantPrompt: false,
		},
		{
			name:       "gemini with skip no task",
			agentType:  AgentGemini,
			skipPerms:  true,
			taskID:     "",
			wantFlag:   "--yolo",
			wantPrompt: false,
		},
		// Cursor tests
		{
			name:       "cursor no skip no task",
			agentType:  AgentCursor,
			skipPerms:  false,
			taskID:     "",
			wantFlag:   "",
			wantPrompt: false,
		},
		{
			name:       "cursor with skip no task",
			agentType:  AgentCursor,
			skipPerms:  true,
			taskID:     "",
			wantFlag:   "-f",
			wantPrompt: false,
		},
		// OpenCode tests (no skip flag)
		{
			name:       "opencode no skip no task",
			agentType:  AgentOpenCode,
			skipPerms:  false,
			taskID:     "",
			wantFlag:   "",
			wantPrompt: false,
		},
		{
			name:       "opencode with skip no task (no flag available)",
			agentType:  AgentOpenCode,
			skipPerms:  true,
			taskID:     "",
			wantFlag:   "",
			wantPrompt: false,
		},
		// Aider tests
		{
			name:       "aider no skip no task",
			agentType:  AgentAider,
			skipPerms:  false,
			taskID:     "",
			wantFlag:   "",
			wantPrompt: false,
		},
		{
			name:       "aider with skip no task",
			agentType:  AgentAider,
			skipPerms:  true,
			taskID:     "",
			wantFlag:   "--yes",
			wantPrompt: false,
		},
	}

	// Create a minimal plugin (no ctx needed for these tests without taskID)
	p := &Plugin{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wt := &Worktree{TaskID: tt.taskID}
			result := p.buildAgentCommand(tt.agentType, wt, tt.skipPerms, nil)

			// Check base command
			baseCmd := getAgentCommand(tt.agentType)
			if !strings.HasPrefix(result, baseCmd) {
				t.Errorf("command should start with %q, got %q", baseCmd, result)
			}

			// Check skip permissions flag
			if tt.wantFlag != "" {
				if !strings.Contains(result, tt.wantFlag) {
					t.Errorf("command should contain flag %q, got %q", tt.wantFlag, result)
				}
			} else if tt.skipPerms {
				// If skipPerms but no wantFlag, ensure no flag was added
				for agent, flag := range SkipPermissionsFlags {
					if agent == tt.agentType && flag != "" {
						t.Errorf("command should not contain flag for %s when wantFlag is empty", tt.agentType)
					}
				}
			}
		})
	}
}

func TestBuildAgentCommandSyntax(t *testing.T) {
	// Test expected output format for each agent
	tests := []struct {
		agentType AgentType
		skipPerms bool
		expected  string
	}{
		{AgentClaude, false, "claude"},
		{AgentClaude, true, "claude --dangerously-skip-permissions"},
		{AgentCodex, false, "codex"},
		{AgentCodex, true, "codex --dangerously-bypass-approvals-and-sandbox"},
		{AgentGemini, false, "gemini"},
		{AgentGemini, true, "gemini --yolo"},
		{AgentCursor, false, "cursor-agent"},
		{AgentCursor, true, "cursor-agent -f"},
		{AgentOpenCode, false, "opencode"},
		{AgentOpenCode, true, "opencode"}, // No skip flag
		{AgentAider, false, "aider"},
		{AgentAider, true, "aider --yes"},
	}

	p := &Plugin{}
	for _, tt := range tests {
		name := string(tt.agentType)
		if tt.skipPerms {
			name += "_skip"
		}
		t.Run(name, func(t *testing.T) {
			wt := &Worktree{TaskID: ""} // No task context
			result := p.buildAgentCommand(tt.agentType, wt, tt.skipPerms, nil)
			if result != tt.expected {
				t.Errorf("buildAgentCommand(%s, skipPerms=%v) = %q, want %q",
					tt.agentType, tt.skipPerms, result, tt.expected)
			}
		})
	}
}

func TestWriteAgentLauncher(t *testing.T) {
	// Test that launcher scripts are created correctly for complex prompts
	tmpDir := t.TempDir()

	p := &Plugin{}

	tests := []struct {
		name      string
		agentType AgentType
		baseCmd   string
		prompt    string
		wantCmd   string
	}{
		{
			name:      "claude with simple prompt",
			agentType: AgentClaude,
			baseCmd:   "claude",
			prompt:    "Task: fix bug",
			wantCmd:   "bash '" + tmpDir + "/.sidecar-start.sh'",
		},
		{
			name:      "claude with complex markdown",
			agentType: AgentClaude,
			baseCmd:   "claude",
			prompt:    "Task: implement feature\n\n```go\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n```\n\nDon't break the user's code!",
			wantCmd:   "bash '" + tmpDir + "/.sidecar-start.sh'",
		},
		{
			name:      "aider uses --message flag",
			agentType: AgentAider,
			baseCmd:   "aider --yes",
			prompt:    "Task: fix bug",
			wantCmd:   "bash '" + tmpDir + "/.sidecar-start.sh'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := p.writeAgentLauncher(tmpDir, tt.agentType, tt.baseCmd, tt.prompt)
			if err != nil {
				t.Fatalf("writeAgentLauncher failed: %v", err)
			}

			if cmd != tt.wantCmd {
				t.Errorf("command = %q, want %q", cmd, tt.wantCmd)
			}

			// Verify launcher script exists and is executable
			launcherInfo, err := os.Stat(tmpDir + "/.sidecar-start.sh")
			if err != nil {
				t.Fatalf("launcher script not created: %v", err)
			}
			if launcherInfo.Mode()&0100 == 0 {
				t.Error("launcher script is not executable")
			}

			// Verify the script contains the prompt embedded in a heredoc
			scriptContent, err := os.ReadFile(tmpDir + "/.sidecar-start.sh")
			if err != nil {
				t.Fatalf("failed to read launcher script: %v", err)
			}
			scriptStr := string(scriptContent)

			// Check that the heredoc delimiter is present
			if !strings.Contains(scriptStr, "SIDECAR_PROMPT_EOF") {
				t.Error("launcher script should contain heredoc delimiter SIDECAR_PROMPT_EOF")
			}

			// Check that the prompt content is embedded in the script
			if !strings.Contains(scriptStr, tt.prompt) {
				t.Errorf("launcher script should contain prompt %q", tt.prompt)
			}

			// Check that the script starts with shebang
			if !strings.HasPrefix(scriptStr, "#!/bin/bash") {
				t.Error("launcher script should start with #!/bin/bash")
			}

			// Cleanup for next test
			os.Remove(tmpDir + "/.sidecar-start.sh")
		})
	}
}
