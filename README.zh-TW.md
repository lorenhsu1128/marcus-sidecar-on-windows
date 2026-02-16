# Sidecar

你可能再也不需要打開編輯器了。

**狀態：已可用於日常開發。** 如遇問題請[回報](https://github.com/lorenhsu1128/marcus-sidecar-on-windows/issues)。

[快速入門](docs/getting-started.zh-TW.md)

![Git 狀態](docs/screenshots/sidecar-git.png)

## 概覽

Sidecar 將你的整個開發工作流整合在一個終端中：使用 [td](https://github.com/marcus/td) 規劃任務、與 AI 代理對話、檢視差異、暫存提交、回顧過往對話、管理工作區——全部無需離開 Sidecar。

## 快速安裝

### macOS（建議）

```bash
brew install marcus/tap/sidecar
```

從原始碼建置，避免 macOS Gatekeeper 警告。

### Linux / 其他

```bash
curl -fsSL https://raw.githubusercontent.com/lorenhsu1128/marcus-sidecar-on-windows/main/scripts/setup.sh | bash
```

### Windows

**一鍵安裝（建議）：**

```powershell
irm https://raw.githubusercontent.com/lorenhsu1128/marcus-sidecar-on-windows/main/scripts/install.ps1 | iex
```

**從原始碼建置：**

```bash
go install github.com/lorenhsu1128/marcus-sidecar-on-windows/cmd/sidecar@latest
```

此指令會將 `sidecar.exe` 安裝到 `%GOPATH%\bin`（預設為 `%USERPROFILE%\go\bin`），需確保該路徑已加入系統 PATH。

**手動安裝：**

從 [Releases](https://github.com/lorenhsu1128/marcus-sidecar-on-windows/releases) 下載 Windows 版本。

Windows 版本使用 ConPTY 作為終端後端（取代 Unix 上的 tmux），無需額外安裝相依套件。

**更多選項：** [下載二進位檔](https://github.com/lorenhsu1128/marcus-sidecar-on-windows/releases) · [手動安裝](docs/getting-started.zh-TW.md)

## 系統需求

- macOS、Linux、WSL 或 Windows 11
- Go 1.21+（僅從原始碼建置時需要）
- tmux（僅 macOS/Linux，用於工作區 shell session）

## 快速開始

安裝後，在任意專案目錄執行：

```bash
sidecar
```

## 建議使用方式

將終端水平分割：左側執行你的 AI 編碼代理（Claude Code、Cursor 等），右側執行 sidecar。

```
┌─────────────────────────────┬─────────────────────┐
│                             │                     │
│   Claude Code / Cursor      │      Sidecar        │
│                             │                     │
│   $ claude                  │   [Git] [Files]     │
│   > fix the auth bug...     │   [Tasks] [Workspaces]│
│                             │                     │
└─────────────────────────────┴─────────────────────┘
```

**提示：** 你可以同時執行兩個 sidecar 實例並排顯示，建立儀表板視圖。例如一個顯示 [Tasks] 外掛，另一個顯示 [Git] 或 [Workspaces]，同時監控一切。

當代理工作時，你可以：

- 在 TD Monitor 中觀察任務在工作流中移動
- 在 Git 外掛中即時看到檔案變更
- 在檔案瀏覽器中自行瀏覽和編輯程式碼
- 查看和恢復所有支援的代理轉接器的對話
- 在內建和社群主題之間切換，並即時預覽

這種設定讓你能看到代理正在做什麼，而不會中斷你的工作流。整個開發循環——規劃、監控、審查、提交——都在終端中進行，而代理負責撰寫程式碼。

## 使用方式

```bash
# 在任意專案目錄執行
sidecar

# 指定專案根目錄
sidecar --project /path/to/project

# 啟用除錯日誌
sidecar --debug

# 檢查版本
sidecar --version
```

## 更新

Sidecar 啟動時會檢查更新。當有新版本可用時，會出現提示通知。按 `!` 開啟診斷視窗查看更新指令。

## 外掛

### Git 狀態

以分割視窗介面查看已暫存、已修改和未追蹤的檔案。側欄顯示檔案和最近的提交；主窗格顯示語法高亮的差異。
![Git 狀態與差異](docs/screenshots/sidecar-git.png)

**功能：**

- 使用 `s`/`u` 暫存/取消暫存檔案
- 使用 `d` 檢視差異（全螢幕）
- 使用 `v` 切換並排差異視圖
- 瀏覽提交歷史並查看提交差異
- 檔案系統變更時自動重新整理

### 對話記錄

瀏覽多個 AI 編碼代理的對話歷史，包含訊息內容、Token 用量和搜尋功能。支援 Amp Code、Claude Code、Codex、Cursor CLI、Gemini CLI、Kiro、OpenCode 和 Warp。
![對話記錄](docs/screenshots/sidecar-conversations.png)

**功能：**

- 跨所有支援代理的統一視圖
- 按日期分組查看所有 session
- 使用 `/` 搜尋 session
- 展開訊息查看完整內容
- 追蹤每個 session 的 Token 用量

### TD Monitor

與 [TD](https://github.com/marcus/td) 整合——專為跨上下文視窗工作的 AI 代理設計的任務管理系統。TD 幫助代理追蹤工作、記錄進度，並在對話之間維持上下文——對於上下文視窗會在對話之間重置的 AI 輔助開發至關重要。
![TD Monitor](docs/screenshots/sidecar-td.png)

**功能：**

- 顯示當前聚焦的任務
- 可捲動的任務列表與狀態指示器
- 活動日誌與 session 上下文
- 使用 `r` 快速提交審查

詳見 [TD 儲存庫](https://github.com/marcus/td) 了解安裝與 CLI 使用方式。

### 檔案瀏覽器

以樹狀視圖和語法高亮預覽瀏覽專案檔案。
![檔案瀏覽器](docs/screenshots/sidecar-files.png)

**功能：**

- 可摺疊的目錄樹
- 語法高亮的程式碼預覽
- 檔案變更時自動重新整理

### 工作區

管理工作區以進行平行開發，整合代理支援。建立隔離的分支作為同層目錄，從 TD 連結任務，並直接從 sidecar 啟動編碼代理。
![工作區](docs/screenshots/sidecar-workspaces.png)

**功能：**

- 使用 `n`/`D` 建立和刪除工作區
- 連結 TD 任務到工作區以追蹤上下文
- 使用 `a` 啟動 Claude Code、Cursor 或 OpenRouter 代理
- 合併工作流：使用 `m` 執行提交、推送、建立 PR 和清理
- 自動將 sidecar 狀態檔加入 .gitignore
- 在分割視窗中預覽差異和任務詳情

## 專案切換器

按 `@` 在已設定的專案之間切換，無需重新啟動 sidecar。

1. 在設定檔中新增專案：

   - macOS/Linux：`~/.config/sidecar/config.json`
   - Windows：`%APPDATA%\sidecar\config.json`

```json
{
  "projects": {
    "list": [
      { "name": "sidecar", "path": "~/code/sidecar" },
      { "name": "td", "path": "~/code/td" },
      { "name": "my-app", "path": "~/projects/my-app" }
    ]
  }
}
```

2. 按 `@` 開啟專案切換視窗
3. 使用 `j/k` 或點擊選擇，按 `Enter` 切換

所有外掛會以新的專案上下文重新初始化。每個專案的狀態（使用中的外掛、游標位置）會被記住。

## Worktree 切換器

按 `W` 在當前儲存庫的 git worktree 之間切換。當你切換離開一個專案後再返回時，sidecar 會記住你之前使用的 worktree 並自動恢復。

## 主題

按 `#` 開啟主題切換器。從內建主題（default、dracula）中選擇，或按 `Tab` 瀏覽 453 個源自 iTerm2-Color-Schemes 的社群配色方案。

社群瀏覽器支援搜尋篩選、瀏覽時即時預覽，以及每個方案的色票顯示。按 `Enter` 將方案儲存為使用中的主題。

詳見 [主題建立技能](.claude/skills/create-theme/SKILL.md) 了解自訂主題建立和色票參考。

## 鍵盤快捷鍵

| 按鍵                | 動作                       |
| ------------------- | -------------------------- |
| `q`、`ctrl+c`       | 退出                       |
| `@`                 | 開啟專案切換器             |
| `W`                 | 開啟 worktree 切換器       |
| `#`                 | 開啟主題切換器             |
| `tab` / `shift+tab` | 切換外掛                   |
| `1-9`               | 依編號聚焦外掛             |
| `j/k`、`↓/↑`        | 瀏覽項目                   |
| `ctrl+d/u`          | 在可捲動視圖中向下/向上翻頁 |
| `g/G`               | 跳到頂部/底部              |
| `enter`             | 選擇                       |
| `esc`               | 返回/關閉                  |
| `r`                 | 重新整理                   |
| `?`                 | 切換說明                   |

### Git 狀態快捷鍵

| 按鍵  | 動作                 |
| ----- | -------------------- |
| `s`   | 暫存檔案             |
| `u`   | 取消暫存檔案         |
| `d`   | 檢視差異（全螢幕）   |
| `v`   | 切換並排差異視圖     |
| `h/l` | 切換側欄/差異焦點    |
| `c`   | 提交已暫存的變更     |

### 工作區快捷鍵

| 按鍵 | 動作                     |
| ---- | ------------------------ |
| `n`  | 建立新工作區             |
| `D`  | 刪除工作區               |
| `a`  | 啟動/連接代理            |
| `t`  | 連結/取消連結 TD 任務    |
| `m`  | 開始合併工作流           |
| `p`  | 推送分支                 |
| `o`  | 在檔案管理器/終端中開啟  |

## 平台注意事項

### Windows

- **終端後端**：Windows 使用 ConPTY（Windows Pseudo Console），自動取代 Unix 上的 tmux，無需額外安裝
- **設定檔位置**：`%APPDATA%\sidecar\config.json`（而非 `~/.config/sidecar/`）
- **狀態檔位置**：`%APPDATA%\sidecar\state.json`
- **預設編輯器**：Windows 上預設使用 `notepad`（可透過 `$EDITOR` 環境變數覆蓋）
- **檔案鎖定**：使用 Windows 原生 `LockFileEx` / `UnlockFileEx`（而非 Unix flock）

### macOS / Linux

- **終端後端**：使用 tmux 管理 shell session
- **設定檔位置**：`~/.config/sidecar/config.json`

## 設定

設定檔路徑：`~/.config/sidecar/config.json`（macOS/Linux）或 `%APPDATA%\sidecar\config.json`（Windows）

```json
{
  "plugins": {
    "git-status": { "enabled": true, "refreshInterval": "1s" },
    "td-monitor": { "enabled": true, "refreshInterval": "2s" },
    "conversations": { "enabled": true },
    "file-browser": { "enabled": true },
    "workspaces": { "enabled": true }
  },
  "ui": {
    "showClock": true,
    "theme": {
      "name": "default",
      "overrides": {}
    }
  }
}
```

## 貢獻

- **回報錯誤**：[開啟 Issue](https://github.com/lorenhsu1128/marcus-sidecar-on-windows/issues)
- **功能請求**：[開啟 Issue](https://github.com/lorenhsu1128/marcus-sidecar-on-windows/issues) 提交你的建議

## 開發

```bash
make build        # 建置至 ./bin/sidecar
make test         # 執行測試
make test-v       # 詳細測試輸出
make install-dev  # 安裝並帶入 git 版本資訊
make fmt          # 格式化程式碼
make fmt-check    # 檢查已變更 Go 檔案的格式
make fmt-check-all # 檢查整個程式碼庫的格式
make lint         # 僅檢查新增的 lint 問題（相對於 main）
make lint-all     # 檢查整個程式碼庫的 lint（含歷史債務）
```

### Go Lint 基準

- 格式化：已變更的 Go 檔案必須通過 `gofmt`（`make fmt-check`）
- 正確性檢查：`errcheck`、`govet`、`ineffassign`、`staticcheck`、`unused`
- 強制執行：CI 在 PR 上執行測試並阻擋新的 lint 問題（`.github/workflows/go-ci.yml`）
- 債務追蹤：執行 `make lint-all` 測量並逐步消除歷史 lint 債務

## 隱私

Sidecar 完全在本地執行，不會發送任何遙測、分析或追蹤請求。唯一的網路呼叫是啟動時的 GitHub API 版本檢查（快取 3 小時）和使用者主動觸發的更新日誌取得。詳見 [PRIVACY.md](PRIVACY.md) 了解完整的資料存取、檔案讀寫和網路行為說明。

## 授權

MIT
