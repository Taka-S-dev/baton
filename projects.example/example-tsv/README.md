# example-tsv

Minimal project layout using hand-written TSV.

```
example-tsv/
├── commands.tsv         <- command definitions (hand-written; baton never modifies it)
├── lists/               <- selection lists for placeholders (value \t label)
├── vars.tsv             <- variable table: {$name} globals + saved commands' fixed values
├── workflows.json       <- workflows (managed by the TUI; hand-editable)
└── commands.local.json  <- commands created via Manage commands (generated)
```

Legacy names (`templates.tsv` / `config.tsv`) are still readable;
`commands.tsv` wins when both exist.

## commands.tsv columns

| Column | Description |
|---|---|
| name | command name |
| group | group label (optional) |
| workdir | working directory; `{slot}` allowed |
| cmd | command to run; `{slot}` allowed |
| shell | shell override (optional) |
| slots | slot-to-list mapping, comma separated (e.g. `workdir=build_directory`) |

- Being a template is a per-row property, not a file: rows containing
  `{slot}` placeholders become sources for
  **Create command → From template**
- Rows without slots (`pwd`, `list-files`) run as-is
- Excel-style quoting is supported: cells like
  `"message=messages,env=environments"` are unquoted on load
- Write ownership is fixed per file: humans edit commands.tsv, baton
  writes commands.local.json — hand edits are never overwritten by the app
- `{$root}` in the `where` command resolves from vars.tsv — edit that
  one line when the project moves to another folder or machine
