# 跨專案概覽 — 願景與探索

## 核心構想

Sidecar 目前以**單一專案模式**運作 — 你可以在專案間切換，但一次只能看到一個。願景分為三個層次：

### 第一層：專案總覽儀表板
一個新的頂層檢視畫面（可能使用 `0` 或專用快捷鍵），**一覽所有已設定的專案**：

```
┌─────────────────────────────────────────────────────────────┐
│  📊 Project Overview                                         │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  sidecar          td               nightshift    betamax    │
│  ├─ main ✓       ├─ main ✓        ├─ main ✓    ├─ main ✓  │
│  ├─ fix/122 🔄   ├─ feat/sync 🔄  └─ fix/path  └──────    │
│  ├─ feat/hooks   ├─ fix/board                               │
│  └─ 8 open PRs   └─ 6 open PRs    4 open PRs   4 open PRs │
│                                                              │
│  Active worktrees: 7 │ Open PRs: 22 │ Failing CI: 3        │
├─────────────────────────────────────────────────────────────┤
│  Recent activity:                                            │
│  • sidecar#125 — fuzzy search (yashas) — 2h ago             │
│  • td#19 — remove .td-root (yashas) — 3h ago                │
│  • nightshift — Event Taxonomy timed out — 6h ago            │
└─────────────────────────────────────────────────────────────┘
```

每張專案卡片顯示：
- 分支/工作樹列表及狀態指示器
- 開放中的 PR 數量
- CI 狀態（綠色/紅色/黃色）
- 最後活動時間戳記

在專案上按 Enter → 切換至該專案（現有行為）。

### 第二層：跨專案 Kanban
看板檢視，將**工作樹作為卡片**，依狀態分組：

```
┌──────────────┬───────────────┬──────────────┬──────────────┐
│  🆕 New       │  🔄 Active     │  📝 Review    │  ✅ Done      │
├──────────────┼───────────────┼──────────────┼──────────────┤
│              │ sidecar       │ sidecar      │ sidecar      │
│              │  fix/122      │  feat/hooks  │  #112 merged │
│              │  "3 commits"  │  "PR #105"   │  "2d ago"    │
│              │               │              │              │
│              │ td            │ td           │              │
│              │  feat/sync    │  #19 PR open │              │
│              │  "wip"        │              │              │
├──────────────┼───────────────┼──────────────┼──────────────┤
│ nightshift   │ betamax       │              │              │
│  fix/path    │  feat/flaky   │              │              │
│  "blocked"   │  "CI failing" │              │              │
└──────────────┴───────────────┴──────────────┴──────────────┘
```

狀態依據下列條件推導：
- **New**：工作樹存在，但無領先 main 的提交
- **Active**：有領先 main 的提交，但尚未建立 PR
- **Review**：PR 已開啟
- **Done**：PR 已合併（顯示近期記錄）

本質上就是一個橫跨所有儲存庫的 git 工作流程 Kanban。

### 第三層：AI 整合（Kestrel / OpenClaw）
這是最令人興奮的部分。透過跨專案檢視，AI 代理（經由 OpenClaw、對話轉接器或未來的聊天外掛）可以：

- **報告**：「你在 4 個專案中有 7 個活躍工作樹，3 個 PR 需要審查」
- **導航**：「顯示 sidecar/fix-122 上失敗的 CI」→ 跳轉至該工作樹
- **分流**：「我接下來該處理什麼？」→ 依閒置時間、CI 狀態、PR 審查排定優先順序
- **管理工作**：關閉過時的工作樹、從議題建立新的工作樹
- **上下文**：已擁有跨所有專案的對話歷史（透過對話轉接器）

我們建立的對話搜尋資料庫（橫跨 Claude Code、Codex、OpenClaw 共 1,977 個會話）直接為此服務 — Kestrel 已經擁有 sidecar 可以呈現的跨專案上下文。

## 現有架構（已實作的部分）

```go
// config.go
type ProjectsConfig struct {
    Mode string          `json:"mode"` // "single" for now
    Root string          `json:"root"`
    List []ProjectConfig `json:"list"` // project switcher
}

type ProjectConfig struct {
    Name  string       `json:"name"`
    Path  string       `json:"path"`
    Theme *ThemeConfig `json:"theme,omitempty"`
}
```

- 專案在 `sidecar.json` 的 `projects.list` 下設定
- `Mode: "single"` — 一次只有一個專案為活動狀態
- 專案切換器：模糊篩選列表，切換工作目錄
- 每個外掛（git、td、conversations、workspace）僅對當前專案運作
- 工作樹管理在 workspace 外掛中以單一專案為單位

## 需要變更的部分

### 階段一：資料層
- 新增 `ProjectOverview` 結構體，彙總所有專案的資料：
  ```go
  type ProjectOverview struct {
      Projects []ProjectStatus
  }
  type ProjectStatus struct {
      Config     ProjectConfig
      Worktrees  []WorktreeStatus  // from git
      OpenPRs    int               // from gh CLI or API
      CIStatus   string            // from gh CLI
      LastCommit time.Time
  }
  type WorktreeStatus struct {
      Path       string
      Branch     string
      CommitsAhead int
      PRNumber   int    // 0 if no PR
      PRState    string // open, merged, closed
      CIStatus   string
  }
  ```

### 階段二：總覽外掛
- 與 git-status、td-monitor、conversations、workspace 並列的新外掛
- 定期輪詢所有已設定的專案
- 渲染儀表板與 Kanban 檢視
- 鍵盤操作：在專案間導航，按 Enter 切換

### 階段三：跨專案模式
- `projects.mode: "overview"` 啟用新的檢視畫面
- 專案切換器增加狀態標誌
- 選配：彙總各專案的 td 看板

### 階段四：AI 橋接
- 透過本地 API 或檔案公開總覽資料
- Kestrel 在報告或按需時讀取
- 未來：與 OpenClaw 溝通的 sidecar 聊天外掛

## 與現有工作的關聯

| 現有功能 | 如何銜接 |
|----------|---------|
| 專案切換器 | 加入狀態徽章，成為進入點 |
| Workspace 外掛 | 工作樹資料匯入 Kanban |
| Git status 外掛 | 各專案的 git 資料彙總至總覽 |
| td-monitor | 各專案的任務數量顯示在總覽中 |
| Conversations 轉接器 | 跨專案對話上下文（已建置完成！） |
| Sit rep 腳本 | 總覽資料可以匯入 sitrep.py |
| @SilentCommandoGames 留言 | 「30 個儲存庫，微服務架構」— 正是他想要的功能 |

## 待討論問題

1. 這些功能有多少應該內建在 sidecar 中，又有多少該做成獨立工具？
2. 總覽應該做成新外掛還是新的「模式」？
3. 對多個儲存庫輪詢 PR/CI 時，GitHub API 速率限制如何處理？
4. 是否應整合 td 看板的任務狀態，還是只專注於 git？
5. 這與 td-watch（管理員儀表板）之間的關係為何？

## 相關資源
- td-10bd20：關於跨專案 AI 管理的 YouTube 影片
- td-ee717a：對話搜尋資料庫的 Sidecar 轉接器
- Issue #126：多儲存庫專案請求
- @SilentCommandoGames：「30 個儲存庫，微服務架構」
