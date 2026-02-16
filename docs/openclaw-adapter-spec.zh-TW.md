# 任務：為 Sidecar 對話外掛建立 OpenClaw Adapter

## 目標
建立一個新的 adapter，讀取 OpenClaw 會話的 JSONL 檔案，使其與 Claude Code 和 Codex 的會話一起顯示在 sidecar 的對話外掛中。這讓 OpenClaw 使用者可以直接在 sidecar 中查看他們的 AI 代理對話。

## 背景
OpenClaw 是一個 AI 代理平台（類似 Claude Code，但附帶訊息傳遞、排程任務、工具等功能）。它將會話以 JSONL 檔案的形式儲存在 `~/.openclaw/agents/main/sessions/` 目錄中。其格式與 Claude Code 相似，但有一些關鍵差異。在測試環境中已有 211 個以上的會話，總計約 14MB。

## 架構

### 現有模式
以 Claude Code adapter（`internal/adapter/claudecode/`）作為主要範本。檔案結構應為：

```
internal/adapter/openclaw/
├── adapter.go          # Main adapter implementation
├── adapter_test.go     # Tests
├── doc.go              # Package doc
├── register.go         # init() registration
├── search.go           # Content search (optional, can add later)
├── search_test.go
├── stats.go            # Cost estimation
├── types.go            # OpenClaw-specific types
└── watcher.go          # fsnotify watcher
```

### 監視範圍
OpenClaw 是一個**全域範圍**的 adapter（與 Codex 相同）。所有會話都存放在同一個目錄中（`~/.openclaw/agents/main/sessions/`），不分專案。Adapter 應實作 `WatchScopeProvider` 並回傳 `WatchScopeGlobal`。

### 會話過濾
與 Claude Code（每個專案有各自的目錄）不同，OpenClaw 將所有會話儲存在同一個扁平目錄中。每個會話的 JSONL 第一行是 `type: "session"` 標頭，其中包含 `cwd`。Adapter 必須：
1. 讀取會話標頭以取得 `cwd`
2. 僅回傳 `cwd` 與傳入 `Sessions()` 的 `projectRoot` 相符的會話
3. 快取 cwd 對應關係，避免每次呼叫時重新讀取標頭

## OpenClaw JSONL 格式

### 行類型

#### 會話標頭（第一行）
```json
{
  "type": "session",
  "version": 3,
  "id": "ec314c57-bdad-4045-ba03-c0c57add5291",
  "timestamp": "2026-02-01T22:02:17.040Z",
  "cwd": "/Users/marcusvorwaller/.openclaw/workspace"
}
```

#### 模型變更
```json
{
  "type": "model_change",
  "id": "6ef314c4",
  "parentId": null,
  "timestamp": "2026-02-03T17:35:22.200Z",
  "provider": "anthropic",
  "modelId": "claude-opus-4-5"
}
```

#### 思考層級變更
```json
{
  "type": "thinking_level_change",
  "id": "93a5a453",
  "parentId": "6ef314c4",
  "timestamp": "2026-02-03T17:35:22.200Z",
  "thinkingLevel": "low"
}
```

#### 自訂事件（模型快照等）
```json
{
  "type": "custom",
  "customType": "model-snapshot",
  "data": {"timestamp": 1770140122200, "provider": "anthropic", "modelApi": "anthropic-messages", "modelId": "claude-opus-4-5"},
  "id": "e0158071",
  "parentId": "30de8b3f"
}
```

#### 使用者訊息
```json
{
  "type": "message",
  "id": "d0759492",
  "parentId": "30de8b3f",
  "timestamp": "2026-02-01T22:06:59.076Z",
  "message": {
    "role": "user",
    "content": [
      {"type": "text", "text": "the actual user message"},
      {"type": "image", "...": "..."}
    ]
  }
}
```

#### 助理訊息（包含思考 + 文字 + 工具呼叫）
```json
{
  "type": "message",
  "id": "e33712cc",
  "parentId": "e5241729",
  "timestamp": "2026-02-01T22:06:59.076Z",
  "message": {
    "role": "assistant",
    "content": [
      {
        "type": "thinking",
        "thinking": "Let me think about this...",
        "thinkingSignature": "ErACCkYI..."
      },
      {
        "type": "text",
        "text": "Here's my response."
      },
      {
        "type": "toolCall",
        "id": "toolu_014idxUeCvvdBYWfhCHzWyJV",
        "name": "edit",
        "arguments": {"path": "/some/file.md", "oldText": "...", "newText": "..."}
      }
    ],
    "api": "anthropic-messages",
    "provider": "anthropic",
    "model": "claude-opus-4-5",
    "usage": {
      "input": 15000,
      "output": 2000,
      "cacheRead": 12000,
      "cacheWrite": 3000,
      "totalTokens": 17000,
      "cost": {
        "input": 0.045,
        "output": 0.15,
        "cacheRead": 0.0036,
        "cacheWrite": 0.01125,
        "total": 0.20985
      }
    },
    "stopReason": "end_turn",
    "timestamp": 1770140122202
  }
}
```

#### 工具結果
```json
{
  "type": "message",
  "id": "02316264",
  "parentId": "e33712cc",
  "timestamp": "2026-02-01T22:06:59.090Z",
  "message": {
    "role": "toolResult",
    "toolCallId": "toolu_014idxUeCvvdBYWfhCHzWyJV",
    "toolName": "edit",
    "content": [
      {"type": "text", "text": "Successfully replaced text in /some/file.md."}
    ],
    "details": {
      "diff": "  1 - old line\n  1 + new line"
    }
  }
}
```

## 與 Claude Code 格式的主要差異

| 面向 | Claude Code | OpenClaw |
|--------|------------|----------|
| 訊息行類型 | `"type": "user"` 或 `"type": "assistant"` | 一律為 `"type": "message"`，角色在 `message.role` 中 |
| 角色值 | `user`, `assistant` | `user`, `assistant`, `toolResult` |
| 工具呼叫 | `content[].type = "tool_use"` | `content[].type = "toolCall"` |
| 工具呼叫輸入 | `content[].input` (any) | `content[].arguments` (object) |
| 工具結果 | 嵌入在下一則 `user` 訊息中作為 `tool_result` 區塊 | 獨立的 `message`，角色為 `role: "toolResult"` |
| 工具結果連結 | `content[].tool_use_id` | `message.toolCallId` |
| ID 欄位 | `uuid` | `id` |
| 父 ID | `parentUuid` | `parentId` |
| 會話 ID | `sessionId`（每行重複出現） | 僅出現在 `type: "session"` 標頭行 |
| 使用量 token 數 | `usage.input_tokens`, `usage.output_tokens` | `usage.input`, `usage.output` |
| Cache token 數 | `usage.cache_read_input_tokens`, `usage.cache_creation_input_tokens` | `usage.cacheRead`, `usage.cacheWrite` |
| 費用 | 根據 token 數量計算 | 預先計算於 `usage.cost.total` 中 |
| 會話目錄 | `~/.claude/projects/<encoded-path>/`（依專案分類） | `~/.openclaw/agents/main/sessions/`（全域） |
| 會話中繼資料 | 分散於各訊息行中（`cwd`、`version`、`gitBranch`） | 集中於 `type: "session"` 標頭 + `type: "model_change"` 行 |
| 子代理 | 檔案名稱前綴 `agent-` | 未知 — 可能需要從會話內容中偵測，或先行忽略 |
| 思考區塊 | `content[].type = "thinking"`, `content[].thinking` | 相同：`content[].type = "thinking"`, `content[].thinking`（另有 `thinkingSignature`） |

## 實作指南

### types.go

定義 OpenClaw 專用型別：

```go
package openclaw

import (
    "encoding/json"
    "time"
)

// RawLine represents any JSONL line from an OpenClaw session.
type RawLine struct {
    Type      string          `json:"type"`
    ID        string          `json:"id"`
    ParentID  string          `json:"parentId"`
    Timestamp time.Time       `json:"timestamp"`
    Message   *MessageContent `json:"message,omitempty"`

    // Session header fields (type="session")
    Version int    `json:"version,omitempty"`
    CWD     string `json:"cwd,omitempty"`

    // Model change fields (type="model_change")
    Provider string `json:"provider,omitempty"`
    ModelID  string `json:"modelId,omitempty"`

    // Custom event fields (type="custom")
    CustomType string          `json:"customType,omitempty"`
    Data       json.RawMessage `json:"data,omitempty"`
}

type MessageContent struct {
    Role       string          `json:"role"`    // "user", "assistant", "toolResult"
    Content    json.RawMessage `json:"content"` // array of content blocks
    Model      string          `json:"model,omitempty"`
    Provider   string          `json:"provider,omitempty"`
    API        string          `json:"api,omitempty"`
    Usage      *Usage          `json:"usage,omitempty"`
    StopReason string          `json:"stopReason,omitempty"`
    ToolCallID string          `json:"toolCallId,omitempty"` // for toolResult role
    ToolName   string          `json:"toolName,omitempty"`   // for toolResult role
    Details    *Details        `json:"details,omitempty"`    // for toolResult extra info
}

type Usage struct {
    Input       int      `json:"input"`
    Output      int      `json:"output"`
    CacheRead   int      `json:"cacheRead"`
    CacheWrite  int      `json:"cacheWrite"`
    TotalTokens int      `json:"totalTokens"`
    Cost        *Cost    `json:"cost,omitempty"`
}

type Cost struct {
    Input      float64 `json:"input"`
    Output     float64 `json:"output"`
    CacheRead  float64 `json:"cacheRead"`
    CacheWrite float64 `json:"cacheWrite"`
    Total      float64 `json:"total"`
}

type Details struct {
    Diff string `json:"diff,omitempty"`
}

// ContentBlock represents a block in the content array.
type ContentBlock struct {
    Type              string          `json:"type"`                         // "text", "thinking", "toolCall", "image"
    Text              string          `json:"text,omitempty"`               // for text blocks
    Thinking          string          `json:"thinking,omitempty"`           // for thinking blocks
    ThinkingSignature string          `json:"thinkingSignature,omitempty"`  // for thinking blocks
    ID                string          `json:"id,omitempty"`                 // for toolCall (the tool_use_id)
    Name              string          `json:"name,omitempty"`               // for toolCall (tool name)
    Arguments         json.RawMessage `json:"arguments,omitempty"`          // for toolCall (tool input)
}

type SessionMetadata struct {
    Path             string
    SessionID        string
    CWD              string
    Version          int
    FirstMsg         time.Time
    LastMsg          time.Time
    MsgCount         int
    TotalTokens      int
    EstCost          float64
    PrimaryModel     string
    FirstUserMessage string
}
```

### adapter.go 關鍵決策

1. **目錄**：`~/.openclaw/agents/main/sessions/` — 暫時寫死，之後可改為可設定
2. **Detect()**：檢查目錄是否存在且有 `.jsonl` 檔案。同時檢查是否有任何會話的 `cwd` 與 `projectRoot` 相符
3. **Sessions()**：讀取所有 `.jsonl` 檔案，解析標頭以取得 `cwd`，依 `projectRoot` 過濾，按 UpdatedAt 排序後回傳
4. **Messages()**：解析訊息行，將 `toolCall` 對應至 adapter 型別中的 `tool_use`，將 `toolResult` 訊息連結至對應的 `toolCall` 區塊
5. **Usage()**：OpenClaw 在 `usage.cost.total` 中預先計算了費用 — 直接使用，無需根據模型費率估算

### 工具結果連結

這是最棘手的部分。在 Claude Code 中，工具結果嵌入在下一則 `user` 訊息中，作為帶有 `tool_use_id` 的 `tool_result` 內容區塊。在 OpenClaw 中，工具結果是獨立的 `message` 行，角色為 `role: "toolResult"`，並帶有 `toolCallId` 欄位。

策略：
- 解析時，透過 `id`（`toolCall` 區塊的 `id` 欄位）追蹤待處理的工具呼叫
- 當 `toolResult` 訊息到達時，在待處理對應表中查找 `toolCallId`
- 將結果內容連結回助理訊息中的工具呼叫
- 這與 Claude Code adapter 的做法類似，只是欄位名稱不同

### register.go

```go
package openclaw

import "github.com/lorenhsu1128/marcus-sidecar-on-windows/internal/adapter"

func init() {
    adapter.RegisterFactory(func() adapter.Adapter {
        return New()
    })
}
```

### watcher.go

監視 `~/.openclaw/agents/main/sessions/` 目錄中新增或修改的 `.jsonl` 檔案。由於這是全域目錄，需實作 `WatchScopeProvider` 並回傳 `WatchScopeGlobal`，以避免重複的監視器。

監視器的實作可以與 Claude Code 的監視器幾乎相同 — 對會話目錄使用 fsnotify，為快速連續的事件進行防抖處理，發出 `EventSessionCreated`/`EventMessageAdded`。

### CWD 快取

由於所有會話都在同一個目錄中，但需要依專案過濾，因此需維護一個 `cwdCache map[string]string`，將會話檔案路徑對應到 CWD。這避免了每次呼叫 `Sessions()` 時重新讀取會話標頭。當檔案被修改時（檢查 mtime），使快取項目失效。

### 效能考量

- 目前有 211 個會話，且會持續增長。CWD 過濾意味著即使只針對特定專案，我們也需要掃描所有會話標頭。
- 使用兩層快取：(1) CWD cache 用於過濾，(2) metadata cache 用於會話詳細資訊
- CWD 一定在檔案的第一行 — 讀取速度很快
- 增量式中繼資料解析（如 Claude Code adapter）在此也適用，因為 JSONL 是只追加寫入的格式

### 圖示與名稱

```go
func (a *Adapter) ID() string   { return "openclaw" }
func (a *Adapter) Name() string { return "OpenClaw" }
func (a *Adapter) Icon() string { return "🐾" }
```

## 測試資料

實際的 OpenClaw 會話檔案位於 `~/.openclaw/agents/main/sessions/`。測試時，建立最小化的 JSONL 固定測試檔案：

```jsonl
{"type":"session","version":3,"id":"test-session-1","timestamp":"2026-02-01T00:00:00Z","cwd":"/test/project"}
{"type":"model_change","id":"mc1","parentId":null,"timestamp":"2026-02-01T00:00:01Z","provider":"anthropic","modelId":"claude-opus-4-5"}
{"type":"message","id":"m1","parentId":"mc1","timestamp":"2026-02-01T00:00:02Z","message":{"role":"user","content":[{"type":"text","text":"Hello, what files are in this directory?"}]}}
{"type":"message","id":"m2","parentId":"m1","timestamp":"2026-02-01T00:00:03Z","message":{"role":"assistant","content":[{"type":"thinking","thinking":"Let me check the files."},{"type":"text","text":"Let me look at the directory contents."},{"type":"toolCall","id":"tc1","name":"exec","arguments":{"command":"ls"}}],"model":"claude-opus-4-5","usage":{"input":500,"output":100,"cacheRead":400,"cacheWrite":100,"totalTokens":600,"cost":{"total":0.015}}}}
{"type":"message","id":"m3","parentId":"m2","timestamp":"2026-02-01T00:00:04Z","message":{"role":"toolResult","toolCallId":"tc1","toolName":"exec","content":[{"type":"text","text":"file1.go\nfile2.go\nREADME.md"}]}}
{"type":"message","id":"m4","parentId":"m3","timestamp":"2026-02-01T00:00:05Z","message":{"role":"assistant","content":[{"type":"text","text":"I can see three files: file1.go, file2.go, and README.md."}],"model":"claude-opus-4-5","usage":{"input":600,"output":50,"cacheRead":500,"cacheWrite":100,"totalTokens":650,"cost":{"total":0.012}}}}
```

## 需建立的檔案

1. `internal/adapter/openclaw/types.go` — 型別定義
2. `internal/adapter/openclaw/adapter.go` — 主要 adapter（Detect、Sessions、Messages、Usage、Watch）
3. `internal/adapter/openclaw/adapter_test.go` — 使用固定測試資料的單元測試
4. `internal/adapter/openclaw/register.go` — 透過 init() 自動註冊
5. `internal/adapter/openclaw/doc.go` — 套件文件
6. `internal/adapter/openclaw/watcher.go` — fsnotify 檔案監視器
7. `internal/adapter/openclaw/stats.go` — 費用計算（可使用預先計算的費用）
8. `internal/adapter/openclaw/testdata/` — 測試用固定 JSONL 檔案

## 驗證

實作完成後：
1. `go test ./internal/adapter/openclaw/...`
2. `go build ./...` — 確保沒有匯入循環
3. 在 `~/.openclaw/workspace` 中開啟 sidecar — OpenClaw 會話應顯示在對話外掛中，並帶有 🐾 標記
4. 確認會話列表顯示正確的標題、時間戳記、費用
5. 確認訊息檢視顯示使用者訊息、助理回應、思考區塊，以及工具呼叫與結果
6. 確認即時更新功能正常（監視器偵測到進行中會話的新訊息）

## 社群影響
此 adapter 讓所有 OpenClaw 使用者都能在 sidecar 中查看他們的代理對話。OpenClaw 社群將從這項整合中獲得顯著的助益 — 這是自然而然的契合，因為這兩個工具都被 AI 輔助開發者所使用。
