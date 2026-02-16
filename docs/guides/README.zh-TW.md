# 指南已遷移至技能

原本放在此資料夾中的 Markdown 指南已遷移至**技能（Skills）**。

- 本倉庫的主要技能位置：`/Users/marcusvorwaller/code/sidecar/.claude/skills/`
- 其他共用技能可在 `/Users/marcusvorwaller/code/sidecar/AGENTS.md` 中查閱

舊版指南檔案保存於：

- `/Users/marcusvorwaller/code/sidecar/docs/deprecated/guides/`

## 技能快速教學

1. 找到相關的技能
   - 瀏覽 `/Users/marcusvorwaller/code/sidecar/AGENTS.md` 中的可用技能清單
   - 或使用 `ls /Users/marcusvorwaller/code/sidecar/.claude/skills` 列出本地倉庫的技能

2. 開啟技能說明
   - 每個技能都記錄在 `SKILL.md` 中
   - 範例：`cat /Users/marcusvorwaller/code/sidecar/.claude/skills/create-plugin/SKILL.md`

3. 依照參考的工作流程檔案操作
   - 技能可能會指向 `references/`、`scripts/` 或範本
   - 建議直接使用這些產出物，而非從頭重寫

4. 明確要求代理使用某個技能
   - 在您的請求中提及技能名稱（例如：`use create-plugin`）
   - 如果有多個技能適用，請逐一指定名稱，代理會將它們結合使用

## 遷移說明

如果您發現舊的 `docs/guides/...` 連結，請將其替換為：

- 對應的技能（`.claude/skills/<name>/SKILL.md`），或
- `docs/deprecated/guides/` 下的存檔副本
