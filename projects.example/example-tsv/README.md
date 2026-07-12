# example-tsv

TSVで手書きする場合の構成例。

```
example-tsv/
├── commands.tsv         ← コマンド定義（手書きの層。batonは読むだけで書き換えない）
├── lists/               ← スロットの選択肢（value \t label）
├── workflows.json       ← ワークフロー（TUIが作成・管理）
└── commands.local.json  ← TUIの Manage commands で作ったコマンド（自動生成）
```

## commands.tsv の列

| 列 | 内容 |
|---|---|
| name | コマンド名 |
| group | グループ（任意） |
| workdir | 作業ディレクトリ。`{slot}` 可 |
| cmd | コマンド。`{slot}` 可 |
| shell | シェル指定（任意） |
| vars | スロット名=リスト名 をカンマ区切り（例: `projDir=project`） |

- 「テンプレート」はファイルではなく行の性質: `{slot}` を含む行が
  TUIの Create command → From template の元ネタになる
- `{slot}` がない行（hello）はそのまま実行できる
- 旧名（templates.tsv / config.tsv）も互換で読める
