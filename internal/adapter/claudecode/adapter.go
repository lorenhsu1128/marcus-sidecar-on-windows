package claudecode

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/marcus/sidecar/internal/adapter"
)

const (
	adapterID   = "claude-code"
	adapterName = "Claude Code"
)

// Adapter implements the adapter.Adapter interface for Claude Code sessions.
type Adapter struct {
	projectsDir  string
	sessionIndex map[string]string // sessionID -> file path cache
}

// New creates a new Claude Code adapter.
func New() *Adapter {
	home, _ := os.UserHomeDir()
	return &Adapter{
		projectsDir:  filepath.Join(home, ".claude", "projects"),
		sessionIndex: make(map[string]string),
	}
}

// ID returns the adapter identifier.
func (a *Adapter) ID() string { return adapterID }

// Name returns the human-readable adapter name.
func (a *Adapter) Name() string { return adapterName }

// Icon returns the adapter icon for badge display.
func (a *Adapter) Icon() string { return "◆" }

// Detect checks if Claude Code sessions exist for the given project.
func (a *Adapter) Detect(projectRoot string) (bool, error) {
	dir := a.projectDirPath(projectRoot)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".jsonl") {
			return true, nil
		}
	}
	return false, nil
}

// Capabilities returns the supported features.
func (a *Adapter) Capabilities() adapter.CapabilitySet {
	return adapter.CapabilitySet{
		adapter.CapSessions: true,
		adapter.CapMessages: true,
		adapter.CapUsage:    true,
		adapter.CapWatch:    true,
	}
}

// Sessions returns all sessions for the given project, sorted by update time.
func (a *Adapter) Sessions(projectRoot string) ([]adapter.Session, error) {
	dir := a.projectDirPath(projectRoot)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var sessions []adapter.Session
	// Reset cache on full session enumeration
	a.sessionIndex = make(map[string]string)
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}

		path := filepath.Join(dir, e.Name())
		meta, err := a.parseSessionMetadata(path)
		if err != nil {
			continue
		}

		// Use first user message as name, with fallbacks
		name := ""
		if meta.FirstUserMessage != "" {
			name = truncateTitle(meta.FirstUserMessage, 50)
		}
		if name == "" && meta.Slug != "" {
			name = meta.Slug
		}
		if name == "" {
			name = shortID(meta.SessionID)
		}

		// Detect sub-agent by filename prefix
		isSubAgent := strings.HasPrefix(e.Name(), "agent-")

		// Cache session path for fast lookup
		a.sessionIndex[meta.SessionID] = path

		sessions = append(sessions, adapter.Session{
			ID:           meta.SessionID,
			Name:         name,
			Slug:         meta.Slug,
			AdapterID:    adapterID,
			AdapterName:  adapterName,
			AdapterIcon:  a.Icon(),
			CreatedAt:    meta.FirstMsg,
			UpdatedAt:    meta.LastMsg,
			Duration:     meta.LastMsg.Sub(meta.FirstMsg),
			IsActive:     time.Since(meta.LastMsg) < 5*time.Minute,
			TotalTokens:  meta.TotalTokens,
			EstCost:      meta.EstCost,
			IsSubAgent:   isSubAgent,
			MessageCount: meta.MsgCount,
		})
	}

	// Sort by UpdatedAt descending (newest first)
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].UpdatedAt.After(sessions[j].UpdatedAt)
	})

	return sessions, nil
}

// Messages returns all messages for the given session.
func (a *Adapter) Messages(sessionID string) ([]adapter.Message, error) {
	path := a.sessionFilePath(sessionID)
	if path == "" {
		return nil, nil
	}

	// First pass: collect tool results from user messages
	toolResults := make(map[string]toolResultInfo)
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024)

	for scanner.Scan() {
		var raw RawMessage
		if err := json.Unmarshal(scanner.Bytes(), &raw); err != nil {
			continue
		}
		if raw.Type != "user" || raw.Message == nil {
			continue
		}
		a.collectToolResults(raw.Message.Content, toolResults)
	}
	file.Close()

	// Second pass: build messages with linked tool results
	file, err = os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var messages []adapter.Message
	scanner = bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024)

	for scanner.Scan() {
		var raw RawMessage
		if err := json.Unmarshal(scanner.Bytes(), &raw); err != nil {
			continue
		}

		// Skip non-message types
		if raw.Type != "user" && raw.Type != "assistant" {
			continue
		}
		if raw.Message == nil {
			continue
		}

		msg := adapter.Message{
			ID:        raw.UUID,
			Role:      raw.Message.Role,
			Timestamp: raw.Timestamp,
			Model:     raw.Message.Model,
		}

		// Parse content with tool results linking
		content, toolUses, thinkingBlocks, contentBlocks := a.parseContentWithResults(raw.Message.Content, toolResults)
		msg.Content = content
		msg.ToolUses = toolUses
		msg.ThinkingBlocks = thinkingBlocks
		msg.ContentBlocks = contentBlocks

		// Parse usage
		if raw.Message.Usage != nil {
			msg.TokenUsage = adapter.TokenUsage{
				InputTokens:  raw.Message.Usage.InputTokens,
				OutputTokens: raw.Message.Usage.OutputTokens,
				CacheRead:    raw.Message.Usage.CacheReadInputTokens,
				CacheWrite:   raw.Message.Usage.CacheCreationInputTokens,
			}
		}

		messages = append(messages, msg)
	}

	return messages, scanner.Err()
}

// toolResultInfo holds parsed tool result data.
type toolResultInfo struct {
	content string
	isError bool
}

// Usage returns aggregate usage stats for the given session.
func (a *Adapter) Usage(sessionID string) (*adapter.UsageStats, error) {
	messages, err := a.Messages(sessionID)
	if err != nil {
		return nil, err
	}

	stats := &adapter.UsageStats{}
	for _, m := range messages {
		stats.TotalInputTokens += m.InputTokens
		stats.TotalOutputTokens += m.OutputTokens
		stats.TotalCacheRead += m.CacheRead
		stats.TotalCacheWrite += m.CacheWrite
		stats.MessageCount++
	}

	return stats, nil
}

// Watch returns a channel that emits events when session data changes.
func (a *Adapter) Watch(projectRoot string) (<-chan adapter.Event, error) {
	return NewWatcher(a.projectDirPath(projectRoot))
}

// projectDirPath converts a project root path to the Claude Code projects directory path.
// Claude Code uses the path with slashes replaced by dashes.
func (a *Adapter) projectDirPath(projectRoot string) string {
	// Ensure absolute path for consistent hashing
	absPath, err := filepath.Abs(projectRoot)
	if err != nil {
		absPath = projectRoot
	}
	// Convert /Users/foo/code/project to -Users-foo-code-project
	hash := strings.ReplaceAll(absPath, "/", "-")
	return filepath.Join(a.projectsDir, hash)
}

// sessionFilePath finds the JSONL file for a given session ID.
func (a *Adapter) sessionFilePath(sessionID string) string {
	// Check cache first
	if path, ok := a.sessionIndex[sessionID]; ok {
		return path
	}

	// Fallback: scan all project directories
	entries, err := os.ReadDir(a.projectsDir)
	if err != nil {
		return ""
	}

	for _, projDir := range entries {
		if !projDir.IsDir() {
			continue
		}
		path := filepath.Join(a.projectsDir, projDir.Name(), sessionID+".jsonl")
		if _, err := os.Stat(path); err == nil {
			// Cache for future lookups
			a.sessionIndex[sessionID] = path
			return path
		}
	}
	return ""
}

// parseSessionMetadata extracts metadata from a session file.
func (a *Adapter) parseSessionMetadata(path string) (*SessionMetadata, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	meta := &SessionMetadata{
		Path:      path,
		SessionID: strings.TrimSuffix(filepath.Base(path), ".jsonl"),
	}

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024)

	modelCounts := make(map[string]int)
	modelTokens := make(map[string]struct{ in, out, cache int })

	for scanner.Scan() {
		var raw RawMessage
		if err := json.Unmarshal(scanner.Bytes(), &raw); err != nil {
			continue
		}

		// Skip non-message types
		if raw.Type != "user" && raw.Type != "assistant" {
			continue
		}

		if meta.FirstMsg.IsZero() {
			meta.FirstMsg = raw.Timestamp
			meta.CWD = raw.CWD
			meta.Version = raw.Version
			meta.GitBranch = raw.GitBranch
		}
		// Extract slug from first message that has it
		if meta.Slug == "" && raw.Slug != "" {
			meta.Slug = raw.Slug
		}
		// Extract first user message content for title
		if meta.FirstUserMessage == "" && raw.Type == "user" && raw.Message != nil {
			content, _, _ := a.parseContent(raw.Message.Content)
			if content != "" {
				meta.FirstUserMessage = content
			}
		}
		meta.LastMsg = raw.Timestamp
		meta.MsgCount++

		// Accumulate token usage from assistant messages
		if raw.Message != nil && raw.Message.Usage != nil {
			usage := raw.Message.Usage
			meta.TotalTokens += usage.InputTokens + usage.OutputTokens

			// Track per-model usage for cost calculation
			model := raw.Message.Model
			if model != "" {
				modelCounts[model]++
				mt := modelTokens[model]
				mt.in += usage.InputTokens
				mt.out += usage.OutputTokens
				mt.cache += usage.CacheReadInputTokens
				modelTokens[model] = mt
			}
		}
	}

	// Determine primary model and calculate cost
	var maxCount int
	for model, count := range modelCounts {
		if count > maxCount {
			maxCount = count
			meta.PrimaryModel = model
		}
	}

	// Calculate cost per model
	for model, mt := range modelTokens {
		var inRate, outRate float64
		switch {
		case strings.Contains(model, "opus"):
			inRate, outRate = 15.0, 75.0
		case strings.Contains(model, "sonnet"):
			inRate, outRate = 3.0, 15.0
		case strings.Contains(model, "haiku"):
			inRate, outRate = 0.25, 1.25
		default:
			inRate, outRate = 3.0, 15.0
		}
		regularIn := mt.in - mt.cache
		if regularIn < 0 {
			regularIn = 0
		}
		meta.EstCost += float64(mt.cache)*inRate*0.1/1_000_000 +
			float64(regularIn)*inRate/1_000_000 +
			float64(mt.out)*outRate/1_000_000
	}

	if meta.FirstMsg.IsZero() {
		meta.FirstMsg = time.Now()
		meta.LastMsg = time.Now()
	}

	return meta, nil
}

// shortID returns the first 8 characters of an ID, or the full ID if shorter.
func shortID(id string) string {
	if len(id) >= 8 {
		return id[:8]
	}
	return id
}

// truncateTitle truncates text to maxLen, adding "..." if truncated.
// It also replaces newlines with spaces for display.
func truncateTitle(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.TrimSpace(s)

	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// collectToolResults extracts tool_result content from user messages.
func (a *Adapter) collectToolResults(rawContent json.RawMessage, results map[string]toolResultInfo) {
	if len(rawContent) == 0 {
		return
	}

	var blocks []ContentBlock
	if err := json.Unmarshal(rawContent, &blocks); err != nil {
		return
	}

	for _, block := range blocks {
		if block.Type == "tool_result" && block.ToolUseID != "" {
			content := ""
			if s, ok := block.Content.(string); ok {
				content = s
			} else if block.Content != nil {
				if b, err := json.Marshal(block.Content); err == nil {
					content = string(b)
				}
			}
			results[block.ToolUseID] = toolResultInfo{
				content: content,
				isError: block.IsError,
			}
		}
	}
}

// parseContent extracts text content, tool uses, and thinking blocks from the content field.
// This is a simplified version for metadata parsing that doesn't need ContentBlocks.
func (a *Adapter) parseContent(rawContent json.RawMessage) (string, []adapter.ToolUse, []adapter.ThinkingBlock) {
	content, toolUses, thinkingBlocks, _ := a.parseContentWithResults(rawContent, nil)
	return content, toolUses, thinkingBlocks
}

// parseContentWithResults extracts content and builds ContentBlocks with linked tool results.
func (a *Adapter) parseContentWithResults(rawContent json.RawMessage, toolResults map[string]toolResultInfo) (string, []adapter.ToolUse, []adapter.ThinkingBlock, []adapter.ContentBlock) {
	if len(rawContent) == 0 {
		return "", nil, nil, nil
	}

	// Try parsing as string first
	var strContent string
	if err := json.Unmarshal(rawContent, &strContent); err == nil {
		contentBlocks := []adapter.ContentBlock{{Type: "text", Text: strContent}}
		return strContent, nil, nil, contentBlocks
	}

	// Parse as array of content blocks
	var blocks []ContentBlock
	if err := json.Unmarshal(rawContent, &blocks); err != nil {
		return "", nil, nil, nil
	}

	var texts []string
	var toolUses []adapter.ToolUse
	var thinkingBlocks []adapter.ThinkingBlock
	var contentBlocks []adapter.ContentBlock
	toolResultCount := 0

	for _, block := range blocks {
		switch block.Type {
		case "text":
			texts = append(texts, block.Text)
			contentBlocks = append(contentBlocks, adapter.ContentBlock{
				Type: "text",
				Text: block.Text,
			})
		case "thinking":
			tokenCount := len(block.Thinking) / 4
			thinkingBlocks = append(thinkingBlocks, adapter.ThinkingBlock{
				Content:    block.Thinking,
				TokenCount: tokenCount,
			})
			contentBlocks = append(contentBlocks, adapter.ContentBlock{
				Type:       "thinking",
				Text:       block.Thinking,
				TokenCount: tokenCount,
			})
		case "tool_use":
			inputStr := ""
			if block.Input != nil {
				if b, err := json.Marshal(block.Input); err == nil {
					inputStr = string(b)
				}
			}
			// Lookup tool result by ID
			var output string
			var isError bool
			if toolResults != nil {
				if result, ok := toolResults[block.ID]; ok {
					output = result.content
					isError = result.isError
				}
			}
			toolUses = append(toolUses, adapter.ToolUse{
				ID:     block.ID,
				Name:   block.Name,
				Input:  inputStr,
				Output: output,
			})
			contentBlocks = append(contentBlocks, adapter.ContentBlock{
				Type:       "tool_use",
				ToolUseID:  block.ID,
				ToolName:   block.Name,
				ToolInput:  inputStr,
				ToolOutput: output,
				IsError:    isError,
			})
		case "tool_result":
			toolResultCount++
			// Add tool_result to content blocks for user messages
			content := ""
			if s, ok := block.Content.(string); ok {
				content = s
			} else if block.Content != nil {
				if b, err := json.Marshal(block.Content); err == nil {
					content = string(b)
				}
			}
			contentBlocks = append(contentBlocks, adapter.ContentBlock{
				Type:       "tool_result",
				ToolUseID:  block.ToolUseID,
				ToolOutput: content,
				IsError:    block.IsError,
			})
		}
	}

	// If we have tool results but no text, show a placeholder
	content := strings.Join(texts, "\n")
	if content == "" && toolResultCount > 0 {
		content = fmt.Sprintf("[%d tool result(s)]", toolResultCount)
	}

	return content, toolUses, thinkingBlocks, contentBlocks
}
