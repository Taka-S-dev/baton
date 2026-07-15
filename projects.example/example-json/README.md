# example-json

Minimal project layout using hand-written JSON.

```
example-json/
├── commands.json        <- command definitions (hand-written; baton never modifies it)
├── lists/               <- selection lists for placeholders (value \t label)
├── workflows.json       <- workflows (managed by the TUI; hand-editable)
└── commands.local.json  <- commands created via Manage commands (generated)
```

- Being a template is a per-row property, not a file: any command
  containing `{slot}` placeholders becomes a source for
  **Create command → From template**
- Commands without slots (`hello`) run as-is
- `slots` maps a slot name to a selection-list name (defaults to the
  slot name itself when omitted)
- Legacy file names (`templates.json`, `template.json`, `config.json`)
  are still readable
