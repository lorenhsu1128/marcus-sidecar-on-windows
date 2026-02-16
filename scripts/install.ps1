# install.ps1 — Sidecar Windows 安裝腳本 / Sidecar Windows Install Script
#
# 使用方式 / Usage:
#   irm https://raw.githubusercontent.com/lorenhsu1128/marcus-sidecar-on-windows/main/scripts/install.ps1 | iex
#
# 此腳本會自動：
# This script will automatically:
#   1. 偵測最新版本 / Detect the latest release version
#   2. 偵測系統架構 / Detect system architecture (amd64 or arm64)
#   3. 下載對應的 .zip 檔案 / Download the corresponding .zip file
#   4. 解壓縮到 $env:LOCALAPPDATA\sidecar\ / Extract to $env:LOCALAPPDATA\sidecar\
#   5. 將安裝目錄加入使用者 PATH / Add the install directory to user PATH
#   6. 驗證安裝 / Verify installation
#
# 需求 / Requirements: PowerShell 5.1+ (Windows PowerShell) 或 PowerShell 7+
# ---------------------------------------------------------------------------

$ErrorActionPreference = "Stop"

# ---------------------------------------------------------------------------
# 輔助函式 / Helper functions
# ---------------------------------------------------------------------------

function Write-Status {
    # 印出藍色狀態訊息 / Print a blue status message
    param([string]$Message)
    Write-Host "[*] " -ForegroundColor Cyan -NoNewline
    Write-Host $Message
}

function Write-Success {
    # 印出綠色成功訊息 / Print a green success message
    param([string]$Message)
    Write-Host "[+] " -ForegroundColor Green -NoNewline
    Write-Host $Message
}

function Write-Failure {
    # 印出紅色錯誤訊息 / Print a red error message
    param([string]$Message)
    Write-Host "[!] " -ForegroundColor Red -NoNewline
    Write-Host $Message
}

function Write-Warn {
    # 印出黃色警告訊息 / Print a yellow warning message
    param([string]$Message)
    Write-Host "[~] " -ForegroundColor Yellow -NoNewline
    Write-Host $Message
}

# ---------------------------------------------------------------------------
# 主要安裝流程 / Main installation flow
# ---------------------------------------------------------------------------

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  Sidecar Installer for Windows" -ForegroundColor Cyan
Write-Host "  Sidecar Windows 安裝程式" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# --- 步驟 1：偵測最新版本 / Step 1: Detect the latest version ---

$RepoApiUrl = "https://api.github.com/repos/lorenhsu1128/marcus-sidecar-on-windows/releases/latest"

Write-Status "正在查詢最新版本... / Fetching latest release version..."

try {
    # 使用 TLS 1.2（PowerShell 5.1 預設可能未啟用）
    # Ensure TLS 1.2 is enabled (may not be default in PowerShell 5.1)
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

    $ReleaseInfo = Invoke-RestMethod -Uri $RepoApiUrl -Headers @{ "Accept" = "application/vnd.github+json" }
    $Version = $ReleaseInfo.tag_name

    if (-not $Version) {
        throw "無法從 API 回應中取得版本標籤 / Could not extract version tag from API response."
    }

    Write-Success "最新版本 / Latest version: $Version"
}
catch {
    Write-Failure "無法取得最新版本資訊 / Failed to fetch the latest release."
    Write-Failure "錯誤 / Error: $_"
    Write-Host ""
    Write-Host "請確認您的網路連線，或手動前往以下網址下載："
    Write-Host "Please check your internet connection, or download manually from:"
    Write-Host "  https://github.com/lorenhsu1128/marcus-sidecar-on-windows/releases" -ForegroundColor Yellow
    exit 1
}

# --- 步驟 2：偵測系統架構 / Step 2: Detect system architecture ---

Write-Status "正在偵測系統架構... / Detecting system architecture..."

$Arch = $null
$SysArch = $env:PROCESSOR_ARCHITECTURE

switch ($SysArch) {
    "AMD64"   { $Arch = "amd64" }
    "x86"     { $Arch = "amd64" }   # 32 位元系統嘗試使用 amd64 / 32-bit fallback to amd64
    "ARM64"   { $Arch = "arm64" }
    default   {
        # 嘗試透過 WMI 偵測 / Try WMI as a fallback
        try {
            $CpuArch = (Get-CimInstance -ClassName Win32_Processor).Architecture
            # Architecture: 9 = x64, 12 = ARM64
            if ($CpuArch -eq 9) { $Arch = "amd64" }
            elseif ($CpuArch -eq 12) { $Arch = "arm64" }
        }
        catch {
            # WMI 查詢失敗，無法偵測 / WMI query failed
        }
    }
}

if (-not $Arch) {
    Write-Failure "無法偵測系統架構：$SysArch / Unsupported architecture: $SysArch"
    Write-Host "支援的架構 / Supported architectures: amd64 (x64), arm64"
    exit 1
}

Write-Success "系統架構 / Architecture: $Arch"

# --- 步驟 3：下載對應的 .zip 檔案 / Step 3: Download the release zip ---

$AssetName = "sidecar_windows_${Arch}.zip"

# 從 release assets 中尋找下載連結 / Find the download URL from release assets
$DownloadUrl = $null
foreach ($Asset in $ReleaseInfo.assets) {
    if ($Asset.name -eq $AssetName) {
        $DownloadUrl = $Asset.browser_download_url
        break
    }
}

if (-not $DownloadUrl) {
    Write-Failure "在此版本中找不到 $AssetName / Asset '$AssetName' not found in release $Version."
    Write-Host ""
    Write-Host "可用的檔案 / Available assets:"
    foreach ($Asset in $ReleaseInfo.assets) {
        Write-Host "  - $($Asset.name)" -ForegroundColor Yellow
    }
    exit 1
}

# 建立暫存目錄 / Create a temporary directory for the download
$TempDir = Join-Path $env:TEMP "sidecar-install-$(Get-Random)"
New-Item -ItemType Directory -Path $TempDir -Force | Out-Null
$ZipPath = Join-Path $TempDir $AssetName

Write-Status "正在下載 $AssetName ... / Downloading $AssetName ..."
Write-Status "下載連結 / URL: $DownloadUrl"

try {
    # 使用 Invoke-WebRequest 下載檔案（相容 PS 5.1 和 PS 7+）
    # Use Invoke-WebRequest for compatibility with both PS 5.1 and PS 7+
    $ProgressPreference = 'SilentlyContinue'  # 加速下載（隱藏進度列）/ Speed up download (hide progress bar)
    Invoke-WebRequest -Uri $DownloadUrl -OutFile $ZipPath -UseBasicParsing
    $ProgressPreference = 'Continue'

    Write-Success "下載完成 / Download complete."
}
catch {
    Write-Failure "下載失敗 / Download failed."
    Write-Failure "錯誤 / Error: $_"

    # 清理暫存檔案 / Clean up temp files
    if (Test-Path $TempDir) { Remove-Item -Recurse -Force $TempDir -ErrorAction SilentlyContinue }
    exit 1
}

# --- 步驟 4：解壓縮到安裝目錄 / Step 4: Extract to install directory ---

$InstallDir = Join-Path $env:LOCALAPPDATA "sidecar"

Write-Status "正在安裝到 / Installing to: $InstallDir"

try {
    # 建立安裝目錄（如不存在）/ Create install directory if it doesn't exist
    if (-not (Test-Path $InstallDir)) {
        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    }

    # 解壓縮（使用 .NET API 以相容 PS 5.1）/ Extract using .NET API for PS 5.1 compatibility
    Add-Type -AssemblyName System.IO.Compression.FileSystem

    # 先解壓到暫存子目錄，再搬移到安裝目錄
    # Extract to a temp subfolder first, then move to install directory
    $ExtractDir = Join-Path $TempDir "extracted"
    [System.IO.Compression.ZipFile]::ExtractToDirectory($ZipPath, $ExtractDir)

    # 將解壓縮的檔案複製到安裝目錄（覆蓋既有檔案）
    # Copy extracted files to install directory (overwrite existing files)
    $ExtractedFiles = Get-ChildItem -Path $ExtractDir -Recurse -File
    foreach ($File in $ExtractedFiles) {
        $RelativePath = $File.FullName.Substring($ExtractDir.Length + 1)
        $DestPath = Join-Path $InstallDir $RelativePath
        $DestDir = Split-Path $DestPath -Parent

        if (-not (Test-Path $DestDir)) {
            New-Item -ItemType Directory -Path $DestDir -Force | Out-Null
        }

        Copy-Item -Path $File.FullName -Destination $DestPath -Force
    }

    Write-Success "解壓縮完成 / Extraction complete."
}
catch {
    Write-Failure "解壓縮或安裝失敗 / Extraction or installation failed."
    Write-Failure "錯誤 / Error: $_"

    # 清理暫存檔案 / Clean up temp files
    if (Test-Path $TempDir) { Remove-Item -Recurse -Force $TempDir -ErrorAction SilentlyContinue }
    exit 1
}

# --- 步驟 5：將安裝目錄加入使用者 PATH / Step 5: Add install dir to user PATH ---

Write-Status "正在檢查 PATH 環境變數... / Checking PATH environment variable..."

$UserPath = [Environment]::GetEnvironmentVariable("Path", [EnvironmentVariableTarget]::User)

if ($UserPath -split ";" | Where-Object { $_ -eq $InstallDir }) {
    Write-Success "安裝目錄已在 PATH 中 / Install directory is already in PATH."
}
else {
    try {
        # 將安裝目錄附加到使用者 PATH / Append install directory to user PATH
        $NewPath = if ($UserPath -and $UserPath[-1] -ne ";") {
            "$UserPath;$InstallDir"
        } else {
            "${UserPath}${InstallDir}"
        }
        [Environment]::SetEnvironmentVariable("Path", $NewPath, [EnvironmentVariableTarget]::User)

        # 同時更新目前工作階段的 PATH / Also update PATH for the current session
        $env:Path = "$env:Path;$InstallDir"

        Write-Success "已將 $InstallDir 加入使用者 PATH / Added $InstallDir to user PATH."
        Write-Warn "新開的終端機視窗才會自動套用新 PATH。"
        Write-Warn "New terminal windows will automatically have the updated PATH."
    }
    catch {
        Write-Warn "無法自動更新 PATH / Could not automatically update PATH."
        Write-Warn "請手動將以下路徑加入 PATH / Please manually add this to your PATH:"
        Write-Host "  $InstallDir" -ForegroundColor Yellow
    }
}

# --- 清理暫存檔案 / Clean up temporary files ---

Write-Status "正在清理暫存檔案... / Cleaning up temporary files..."

if (Test-Path $TempDir) {
    Remove-Item -Recurse -Force $TempDir -ErrorAction SilentlyContinue
}

Write-Success "暫存檔案已清理 / Temporary files cleaned up."

# --- 步驟 6：驗證安裝 / Step 6: Verify installation ---

Write-Status "正在驗證安裝... / Verifying installation..."

$SidecarExe = Join-Path $InstallDir "sidecar.exe"

if (Test-Path $SidecarExe) {
    try {
        $VersionOutput = & $SidecarExe --version 2>&1
        Write-Success "版本資訊 / Version: $VersionOutput"
    }
    catch {
        Write-Warn "sidecar.exe 存在但無法執行 --version / sidecar.exe exists but --version failed."
        Write-Warn "錯誤 / Error: $_"
    }
}
else {
    Write-Warn "找不到 sidecar.exe / sidecar.exe not found at: $SidecarExe"
    Write-Warn "zip 檔案內的結構可能與預期不同 / The zip file structure may differ from expected."
    Write-Host "安裝目錄內容 / Install directory contents:" -ForegroundColor Yellow
    Get-ChildItem -Path $InstallDir | ForEach-Object { Write-Host "  $($_.Name)" }
}

# --- 步驟 7：成功訊息 / Step 7: Print success message ---

Write-Host ""
Write-Host "========================================" -ForegroundColor Green
Write-Host "  安裝完成！ / Installation complete!" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Green
Write-Host ""
Write-Host "  版本 / Version:   $Version" -ForegroundColor White
Write-Host "  架構 / Arch:      $Arch" -ForegroundColor White
Write-Host "  安裝路徑 / Path:  $InstallDir" -ForegroundColor White
Write-Host ""
Write-Host "  請開啟新的終端機視窗，然後輸入：" -ForegroundColor White
Write-Host "  Open a new terminal window and run:" -ForegroundColor White
Write-Host ""
Write-Host "    sidecar" -ForegroundColor Cyan
Write-Host ""
Write-Host "  如需更多資訊 / For more information:" -ForegroundColor White
Write-Host "    https://github.com/lorenhsu1128/marcus-sidecar-on-windows" -ForegroundColor Yellow
Write-Host ""
