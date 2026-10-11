# example-tsv

Minimal project layout using hand-written TSV.

```
example-tsv/
├── commands.tsv         <- command definitions (edit freely; the TUI changes only the rows you target)
├── lists/               <- selection lists for placeholders (value \t label)
├── vars.tsv             <- variable table: {$name} globals + saved commands' fixed values
├── workflows.json       <- workflows (written by the TUI; do not edit)
└── commands.local.json  <- commands saved from a template (written by the TUI; do not edit)
```

workflows.json and commands.local.json appear once you save a workflow or
a command from a template; [AGENTS.md](../../AGENTS.md) lists every file
and who may write it.

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
  **Edit project → Commands → Add → From template**
- Rows without slots (`hello`, `where`) run as-is
- Excel-style quoting is supported: cells like
  `"message=messages,env=environments"` are unquoted on load
- Write ownership: humans edit commands.tsv freely; the TUI can also
  append rows (**Write directly**) and edit or delete the specific row
  you target — every other line is preserved byte-for-byte.
  Template-derived commands live in commands.local.json + vars.tsv,
  both managed by baton
- `{env...}` in `deploy-all` is a variadic slot: Tab toggles
  multiple entries and the values are joined with spaces
- `{$root}` in the `where` command resolves from vars.tsv — edit that
  one line when the project moves to another folder or machine
