# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 必讀指引

使用 `td usage --new-session` 查看待辦工作與任務/史詩。若使用者未提供現有任務，先用 td 建立任務再開始工作。

## 建置與開發指令

```bash
make build            # 編譯至 ./bin/sidecar
make install-dev      # 安裝並帶入 git 版本資訊
make test             # 執行測試
go test ./internal/adapter/cursor/...   # 執行單一套件測試
go test -run TestName ./internal/...    # 執行單一測試
make fmt              # 格式化程式碼
make lint             # 僅檢查相對於 main 的新增 lint 問題
make lint-all         # 檢查整個程式碼庫（含歷史債務）
```

版本透過 ldflags 在建置時設定（`-X main.Version=vX.Y.Z`）。未設定時回退至 git revision。

## 架構概覽

Sidecar 是一個 AI 編碼代理的 TUI 儀表板，使用 **Bubble Tea** 框架。模組路徑：`github.com/lorenhsu1128/marcus-sidecar-on-windows`。

### 核心分層

```
cmd/sidecar/main.go    → 進入點：旗標解析、設定載入、外掛註冊、啟動 Bubble Tea
internal/app/           → 根 Model：外掛編排、模態管理、全域按鍵路由
internal/plugin/        → 外掛介面定義、Context、Registry 生命週期
internal/plugins/       → 6 個外掛實作（tdmonitor, gitstatus, filebrowser, conversations, workspace, notes）
internal/adapter/       → 9 個 AI 代理階段資料轉接器（claudecode, cursor, warp, amp, codex 等）
internal/keymap/        → 鍵盤綁定系統：三層匹配（Command ID → Binding → Context）
internal/modal/         → 宣告式模態系統
internal/event/         → 事件匯流排（發佈/訂閱，跨外掛通訊）
internal/features/      → 功能旗標系統
internal/config/        → 設定管理（全域 + 專案層級）
internal/state/         → 持久化 UI 狀態（state.json）
internal/theme/         → 主題解析與套用
internal/styles/        → Lipgloss 樣式、膠囊元件渲染
internal/mouse/         → 滑鼠支援（點擊區域、拖曳、懸停）
```

### 外掛系統

外掛實作 `plugin.Plugin` 介面（ID, Name, Init, Start, Stop, Update, View, Commands 等）。在 `main.go` 中註冊，順序決定分頁順序。

**生命週期**：Registration → Init() → Start() → Update()/View() 循環 → Stop()

**關鍵渲染規則**：
- **務必限制外掛輸出高度**，否則標題列會被捲出畫面
- **不要**在外掛的 View() 中渲染頁尾 — app 層統一渲染頁尾，使用 `Commands()` 回傳
- 指令名稱保持簡短（一個詞為佳：「Stage」而非「Stage file」）

### 轉接器系統

轉接器實作 `adapter.Adapter` 介面，透過 `init()` 自動註冊。支援 Detect、Sessions、Messages、Watch 等能力。可選介面：`ProjectDiscoverer`、`TargetedRefresher`、`WatchScopeProvider`。

### 跨外掛通訊

所有外掛透過 `tea.Msg` 廣播接收訊息：
- `FocusPluginByIDMsg{PluginID}` — 切換焦點
- `NavigateToFileMsg{Path}` — 導向檔案瀏覽器
- 事件匯流排：在 Start() 中訂閱，透過 `ctx.EventBus.Publish()` 發佈

### Epoch 模式

專案/工作樹切換後，用 epoch 標記丟棄過期訊息：
```go
type MyMsg struct { Epoch uint64; Data string }
func (m MyMsg) GetEpoch() uint64 { return m.Epoch }
```

## 程式碼規範

- 所有 Go 程式碼必須通過 `gofmt`
- CI 會阻擋 PR 中的新 lint 問題（errcheck, govet, ineffassign, staticcheck, unused）
- 日誌只能輸出到檔案，**絕不可輸出到 stderr**（會破壞 TUI）
- 技能文件位於 `.claude/skills/`，包含外掛開發、UI 功能、發佈流程等詳細指引

## 設定檔位置

- `~/.config/sidecar/config.json` — 使用者設定
- `.sidecar/config.json` — 專案層級設定
- `~/.config/sidecar/state.json` — 持久化 UI 狀態

## 功能旗標

透過 `features.IsEnabled(name)` 查詢。優先順序：CLI 旗標 > 設定檔 > 程式碼預設值。
目前可用：`tmux_interactive_input`（預設開）、`tmux_inline_edit`（預設開）、`notes_plugin`（預設關）。
