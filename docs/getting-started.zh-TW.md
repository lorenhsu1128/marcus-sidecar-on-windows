# Sidecar 入門指南

## 快速安裝

```bash
curl -fsSL https://raw.githubusercontent.com/lorenhsu1128/marcus-sidecar-on-windows/main/scripts/setup.sh | bash
```

或透過 Homebrew 安裝（僅限 macOS）：

```bash
brew install marcus/tap/sidecar
```

安裝腳本會詢問你想安裝什麼：
- **同時安裝 td 和 sidecar**（建議）— td 提供 AI 工作流程的任務管理功能
- **僅安裝 sidecar** — 可獨立運作，不需要 td

### Windows

**快速安裝（建議）：**

```powershell
irm https://raw.githubusercontent.com/lorenhsu1128/marcus-sidecar-on-windows/main/scripts/install.ps1 | iex
```

**從原始碼安裝：**

```bash
go install github.com/lorenhsu1128/marcus-sidecar-on-windows/cmd/sidecar@latest
```

這會將 `sidecar.exe` 安裝到 `%GOPATH%\bin`（預設為 `%USERPROFILE%\go\bin`）。請確保該路徑已加入系統 PATH。

Windows 使用 ConPTY 作為終端後端（取代 Unix 上的 tmux），因此不需要額外的相依套件。

## 系統需求

- macOS、Linux、WSL 或 Windows 11
- 終端機存取權限
- Go 1.21+（僅在從原始碼建置時需要 — 透過 Homebrew 或二進位檔安裝則不需要 Go）

## 安裝腳本的運作流程

1. 顯示目前 Go、td 和 sidecar 的安裝狀態
2. 詢問你想安裝什麼
3. 在執行每個指令前先顯示內容（由你逐一批准）
4. 若缺少 Go 則自動安裝
5. 設定 PATH
6. 安裝你選擇的工具
7. 驗證安裝結果

## 更新

執行同一個指令即可 — 腳本會偵測已安裝的版本，僅更新需要更新的部分。

```bash
curl -fsSL https://raw.githubusercontent.com/lorenhsu1128/marcus-sidecar-on-windows/main/scripts/setup.sh | bash
```

## 無介面/CI 安裝

```bash
# 安裝全部（預設）
curl -fsSL https://raw.githubusercontent.com/lorenhsu1128/marcus-sidecar-on-windows/main/scripts/setup.sh | bash -s -- --yes

# 僅安裝 sidecar
curl -fsSL https://raw.githubusercontent.com/lorenhsu1128/marcus-sidecar-on-windows/main/scripts/setup.sh | bash -s -- --yes --sidecar-only

# 即使已是最新版也強制重新安裝
curl -fsSL https://raw.githubusercontent.com/lorenhsu1128/marcus-sidecar-on-windows/main/scripts/setup.sh | bash -s -- --yes --force
```

## 二進位檔下載

直接從 [GitHub Releases](https://github.com/lorenhsu1128/marcus-sidecar-on-windows/releases) 下載預先編譯的二進位檔。提供 macOS、Linux 和 Windows 版本（amd64 及 arm64）。

**macOS / Linux：**

1. 下載適合你平台的壓縮檔
2. 解壓縮：`tar -xzf sidecar_*.tar.gz`
3. 移至 PATH 目錄：`mv sidecar /usr/local/bin/`（或 `~/go/bin/`）

**Windows：**

1. 下載適合你架構的 `.zip` 檔（amd64 或 arm64）
2. 解壓縮 zip 檔
3. 將 `sidecar.exe` 移至 PATH 中的目錄（例如 `%LOCALAPPDATA%\sidecar\`）

## 手動安裝

如果你偏好手動安裝：

### 1. 安裝 Go

```bash
# macOS
brew install go

# Ubuntu/Debian
sudo apt install golang

# 其他
# 從 https://go.dev/dl/ 下載
```

### 2. 設定 PATH

加入 ~/.zshrc 或 ~/.bashrc：

```bash
export PATH="$HOME/go/bin:$PATH"
```

### 3. 安裝 sidecar

```bash
go install github.com/lorenhsu1128/marcus-sidecar-on-windows/cmd/sidecar@latest
```

### 4.（選用）安裝 td

```bash
go install github.com/marcus/td@latest
```

## 快速開始

安裝完成後，在任何專案目錄中執行：

```bash
sidecar
```

**提示：** 你可以同時執行兩個 sidecar 實例並排顯示（例如在分割終端中），建立儀表板視圖。這讓你能在一個窗格中監控任務（TD Monitor），同時在另一個窗格中檢視程式碼變更（Git Status）或管理並行工作（Workspaces）。

## 檢查更新

在 sidecar 中按下 `!` 開啟診斷畫面。你會看到已安裝工具的版本資訊，以及可用更新的升級指令。

## 疑難排解

### 「command not found: sidecar」

你的 PATH 可能未包含 ~/go/bin。請執行：

```bash
echo 'export PATH="$HOME/go/bin:$PATH"' >> ~/.zshrc && source ~/.zshrc
```

### 「permission denied」

修正 Go 目錄的擁有權：

```bash
sudo chown -R $USER ~/go
```

### 網路問題

安裝腳本需要網路連線才能從 GitHub 下載。若在代理伺服器後方，請設定 HTTPS_PROXY 環境變數。

### Go 版本過舊

安裝腳本需要 Go 1.21+。請更新 Go：

```bash
# macOS
brew upgrade go

# Linux — 從 https://go.dev/dl/ 下載最新版
```

### Windows：「sidecar is not recognized as an internal or external command」

sidecar 二進位檔不在你的 PATH 中。請新增：

```powershell
# 若透過 go install 安裝：
$env:Path += ";$env:USERPROFILE\go\bin"
[Environment]::SetEnvironmentVariable("Path", $env:Path + ";$env:USERPROFILE\go\bin", "User")

# 若手動安裝：
[Environment]::SetEnvironmentVariable("Path", $env:Path + ";$env:LOCALAPPDATA\sidecar", "User")
```

更新 PATH 後請重新啟動終端機。

### Windows：PowerShell 執行原則

若 PowerShell 阻擋安裝腳本，請執行：

```powershell
Set-ExecutionPolicy RemoteSigned -Scope CurrentUser
```

### Windows：Windows Defender 誤判

若 Windows Defender 標記 `sidecar.exe`，請新增排除項目：

```powershell
Add-MpPreference -ExclusionPath "$env:LOCALAPPDATA\sidecar"
```
