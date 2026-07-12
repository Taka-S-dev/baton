# example-json

JSONで手書きする場合の構成例。

```
example-json/
├── commands.json        ← コマンド定義（手書きの層。batonは読むだけで書き換えない）
├── lists/               ← スロットの選択肢（value \t label）
├── workflows.json       ← ワークフロー（TUIが管理。手書きも可）
└── commands.local.json  ← TUIの Manage commands で作ったコマンド（自動生成）
```

- 「テンプレート」はファイルではなく行の性質: `{slot}` を含むコマンドが
  TUIの Create command → From template の元ネタになる
- `{slot}` がないコマンド（hello）はそのまま実行できる
- `slots` はスロット名→リスト名の対応（省略時はスロット名と同名のリストを使う）
- 旧名（templates.json / template.json / config.json）も互換で読める
