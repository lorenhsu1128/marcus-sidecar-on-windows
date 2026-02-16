# 修復 #122：自動更新功能失效

## 問題

https://github.com/lorenhsu1128/marcus-sidecar-on-windows/issues/122

使用者回報內建更新器（! → u）無法正常運作。主要影響 Homebrew 使用者，也可能影響透過 go install 安裝的使用者。

## 根本原因

`internal/app/model.go` 中有三個錯誤同時作用：

### 錯誤 1：在 `brew upgrade` 之前缺少 `brew update`
在 `runInstallPhase()` 中（約第 489 行），程式直接執行 `brew upgrade sidecar`，卻沒有先執行 `brew update`。Homebrew tap 是在本機快取的——若不重新整理，brew 不會知道 tap 中有新版本，並會以 exit code 0 回報「已安裝」。

### 錯誤 2：靜默的假成功
`brew upgrade` 在「已安裝」的情況下會回傳 exit code 0。程式只檢查 `err`，因此即使實際上什麼都沒發生，仍會將 `sidecarUpdated = true`。

### 錯誤 3：驗證未檢查實際版本
`runVerifyPhase()`（約第 542 行）只確認 `sidecar --version` 能正常執行，從未將輸出與 `installResult.NewSidecarVersion` 比對。因此即使更新沒有實際生效，驗證也會通過。

## 所需變更

### 1. `runInstallPhase()` — Homebrew 情境

在 `brew upgrade` 之前加入 `brew update`：

```go
case version.InstallMethodHomebrew:
    // Refresh tap first so brew knows about the new version
    updateCmd := exec.Command("brew", "update", "--auto-update")
    _ = updateCmd.Run() // Best-effort; upgrade may still work if tap is fresh

    cmd := exec.Command("brew", "upgrade", "sidecar")
    output, err := cmd.CombinedOutput()
    if err != nil {
        return UpdateErrorMsg{Step: "sidecar", Err: fmt.Errorf("%v: %s", err, output)}
    }
    // Check for "already installed" false positive
    if strings.Contains(string(output), "already installed") {
        return UpdateErrorMsg{Step: "sidecar", Err: fmt.Errorf("brew upgrade reported already installed — tap may be out of date")}
    }
```

### 2. `runVerifyPhase()` — 版本比對

更新後，驗證實際安裝的版本是否與預期一致：

```go
if installResult.SidecarUpdated {
    sidecarPath, err := exec.LookPath("sidecar")
    if err != nil {
        return UpdateErrorMsg{Step: "verify", Err: fmt.Errorf("sidecar not found in PATH after install")}
    }
    cmd := exec.Command(sidecarPath, "--version")
    output, err := cmd.Output()
    if err != nil {
        return UpdateErrorMsg{Step: "verify", Err: fmt.Errorf("sidecar binary not executable: %v", err)}
    }
    installedVersion := strings.TrimSpace(string(output))
    if installResult.NewSidecarVersion != "" && !strings.Contains(installedVersion, strings.TrimPrefix(installResult.NewSidecarVersion, "v")) {
        return UpdateErrorMsg{Step: "verify", Err: fmt.Errorf("version mismatch after update: expected %s, got %s — the update may not have taken effect", installResult.NewSidecarVersion, installedVersion)}
    }
}
```

## 測試方式

1. 透過 `brew install marcus/tap/sidecar` 安裝 sidecar
2. 手動將本機 tap formula 修改為舊版（或不執行 `brew update`）
3. 透過 TUI 觸發更新（! → u）
4. 確認程式現在會先執行 `brew update`，然後成功升級
5. 確認版本檢查能偵測到更新失敗的情況

## 備註
- go install 路徑似乎能正常運作——@nick4eva 的問題可能與 PATH 有關，或是需要重新啟動。值得進一步詢問更多資訊。
- 考慮在更新驗證失敗時顯示使用者可見的訊息：「更新未生效。請嘗試手動執行 `brew update && brew upgrade sidecar`。」
