# example-json

Minimal project layout using hand-written JSON.

```
example-json/
├── commands.json        <- command definitions (hand-written; baton never writes it)
├── lists/               <- selection lists for placeholders (value \t label)
├── vars.tsv             <- variable table: {$name} globals + saved commands' fixed values
├── workflows.json       <- workflows (written by the TUI; do not edit)
└── commands.local.json  <- commands saved from a template (written by the TUI; do not edit)
```

workflows.json and commands.local.json appear once you save a workflow or
a command from a template; [AGENTS.md](../../AGENTS.md) lists every file
and who may write it.

- Being a template is a per-row property, not a file: any command
  containing `{slot}` placeholders becomes a source for
  **Edit project → Commands → Add → From template**
- Commands without slots (`hello`) run as-is
- `slots` maps a slot name to a selection-list name (defaults to the
  slot name itself when omitted)
- `{env...}` in `deploy-all` is a variadic slot: Tab toggles
  multiple entries and the values are joined with spaces
- `{$root}` in the `where` command resolves from vars.tsv — edit that
  one line when the project moves to another folder or machine
