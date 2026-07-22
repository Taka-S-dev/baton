# example-json

Minimal project layout using hand-written JSON.

```
example-json/
├── commands.json        <- command definitions (hand-written; baton never modifies it)
├── lists/               <- selection lists for placeholders (value \t label)
├── vars.tsv             <- variable table: {$name} globals + saved commands' fixed values
├── workflows.json       <- workflows (managed by the TUI; hand-editable)
└── commands.local.json  <- commands created via Manage commands (generated)
```

- Being a template is a per-row property, not a file: any command
  containing `{slot}` placeholders becomes a source for
  **Create command → From template**
- Commands without slots (`hello`) run as-is
- `slots` maps a slot name to a selection-list name (defaults to the
  slot name itself when omitted)
- `{env...}` in `deploy-all` is a variadic slot: Tab toggles
  multiple entries and the values are joined with spaces
- `{$root}` in the `where` command resolves from vars.tsv — edit that
  one line when the project moves to another folder or machine
