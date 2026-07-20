# baton

[![CI](https://github.com/Taka-S-dev/baton/actions/workflows/ci.yml/badge.svg)](https://github.com/Taka-S-dev/baton/actions/workflows/ci.yml)
[![Release](https://github.com/Taka-S-dev/baton/actions/workflows/release.yml/badge.svg)](https://github.com/Taka-S-dev/baton/actions/workflows/release.yml)

```
  _           _
 | |__   __ _| |_ ___  _ __
 | '_ \ / _` | __/ _ \| '_ \
 | |_) | (_| | || (_) | | | |
 |_.__/ \__,_|\__\___/|_| |_|
```

A terminal-based workflow runner for Windows, Linux, and macOS. Define commands in a config file, select and execute them interactively.

## Why baton?

Honestly: laziness. I didn't want to memorize each project's command combinations. I didn't want to babysit the terminal, typing the next command each time one finished. And I wanted a new command to be a near-copy of an existing one — pick a slotted command as a template, fill in different values, done. baton is those habits turned into a tool.

Task runners are a crowded space — [just](https://github.com/casey/just), [go-task](https://github.com/go-task/task), or a dozen lines of [fzf](https://github.com/junegunn/fzf) glue cover much of what baton does. baton exists for a combination none of them offers on its own:

- **Windows as a first-class citizen.** The TUI and every feature behave the same in cmd.exe, PowerShell, and Unix shells — no WSL or bash dependency, and each command chooses its shell (`cmd`/`sh` by default, `ps` opt-in).
- **No DSL to learn.** Commands are plain JSON or TSV, and once a project is set up the TUI creates and edits them for you.
- **Interactive placeholders.** `{slots}` resolve at run time from selection lists with a live command preview — closer to a snippet manager like [pet](https://github.com/knqyf263/pet) than to make, but with saved workflows on top.
- **Workflows as data.** Frequent sequences are saved, searched, and re-run from a menu instead of being encoded in a build file.

If you live happily in a Justfile or an fzf pipeline, those remain great choices. baton is for keeping a per-project command palette that anyone can drive from a menu.

Named after the relay baton: each step hands off to the next.

## Features

- Interactive TUI with real-time search and multi-select
- Save and reuse command combinations as workflows
- Combine multiple commands into a single alias
- `{placeholder}` substitution — pick values from a selection list at runtime
- Create commands from the TUI: write one directly, or derive one from any
  slotted command (template) with pre-filled values
- Optional `slots` field to map slot names to named lists
- Supports `sh` (Linux/macOS), `cmd.exe` (Windows), and PowerShell per command
- Remembers the last used workflow
- Retry from the failed step when a command fails

## Architecture

```
baton/
├── main.go                   # Entry point, CLI flags, bubbletea program setup
├── internal/
│   ├── model/                # Shared data types (Command, Workflow, Alias, RunItem)
│   ├── config/               # Config loading (JSON/TSV) and projects/ directory discovery
│   ├── slot/                 # {placeholder} parsing, resolution, and .tsv list loading
│   ├── store/                # Workflow and alias persistence (JSON files)
│   ├── runner/               # Command execution via tea.ExecProcess (suspends TUI)
│   └── tui/                  # Bubbletea Model / Update / View
│       ├── model.go          # Model struct, screen enum, sub-states, New()
│       ├── update.go         # Update() entry point, message dispatch
│       ├── update_menu.go    # Project select, main menu, config switch
│       ├── update_run.go     # Run, confirm, retry
│       ├── update_resolve.go # Multi-select, slot resolution, confirm vars
│       ├── update_workflow.go# Workflow CRUD
│       ├── update_alias.go   # Alias CRUD
│       ├── update_manage_commands.go # Command CRUD (direct input / from template)
│       ├── update_list.go    # List and name-input screens
│       ├── view.go           # All rendering functions
│       └── styles.go         # Lipgloss styles and helper render functions
└── projects.example/         # Sample projects (JSON and TSV) to copy as a starting point
```

The TUI follows the standard [Bubble Tea](https://github.com/charmbracelet/bubbletea) architecture (Elm-style Model/Update/View). Each screen has a corresponding `update*` and `view*` function. Slot resolution and workflow/alias creation share a common `resolveFlowState` that drives multi-step placeholder prompting across multiple commands.

## Dependencies

| Library | Purpose |
|---------|---------|
| [charmbracelet/bubbletea](https://github.com/charmbracelet/bubbletea) | TUI framework (Elm-style Model/Update/View) |
| [charmbracelet/lipgloss](https://github.com/charmbracelet/lipgloss) | Terminal styling and layout |
| [charmbracelet/bubbles](https://github.com/charmbracelet/bubbles) | UI components (textinput, viewport, spinner) |

## Installation

### Pre-built binaries

Download the latest binary for your platform from the [Releases](../../releases) page.

### Build from source

```
go build -o baton .
```

Requires Go 1.24+.

## Quick start

Sample projects are included in `projects.example/` — `example-json` (JSON) and `example-tsv` (TSV). Copy them to get started:

```
cp -r projects.example projects   # Linux/macOS
```

```
xcopy projects.example projects /E /I   # Windows
```

Then run `baton`.

## Setup

baton looks for a `projects/` directory in this order:

1. `$BATON_PROJECTS_DIR` environment variable
2. `projects/` folder adjacent to the executable
3. `~/.config/baton/projects/` (Linux / macOS)

Create a subfolder for each project and add `commands.json` or `commands.tsv` inside it:

```
~/.config/baton/projects/    (Linux/macOS)
├── projectA/
│   └── commands.json
└── projectB/
    └── commands.tsv
```

```
any-folder/                  (Windows, or portable install)
├── baton.exe
└── projects/
    ├── projectA/
    │   └── commands.json
    └── projectB/
        └── commands.tsv
```

If multiple projects exist, baton shows a selection screen on startup. Use **Switch config** from the main menu to switch projects at any time.

## Configuration

A project has two command layers:

| File | Written by | Contents |
|------|-----------|----------|
| `commands.json` / `commands.tsv` | You (hand-written; baton never modifies it) | Command definitions — plain ones, and slotted ones that double as templates |
| `commands.local.json` | baton (via **Manage commands**) | Commands you created in the TUI |

Both layers can coexist; names in `commands.local.json` take priority.
Legacy file names (`templates.json`, `template.json`, `templates.tsv`, `config.tsv`, `config.json`) are still readable.

### commands.json

```json
{
  "commands": [
    { "name": "build",  "group": "make",   "workdir": "{projDir}", "cmd": "echo building {projCmd}", "slots": { "projDir": "project", "projCmd": "project" } },
    { "name": "test",   "group": "make",   "workdir": "{project}", "cmd": "echo testing {project}" },
    { "name": "deploy", "group": "deploy", "cmd": "echo deploying {env}" }
  ]
}
```

### commands.tsv

Tab-separated alternative to `commands.json` (both can coexist; entries are merged).

```
name	group	workdir	cmd	shell	vars
build	make	{projDir}	echo building {projCmd}		projDir=project,projCmd=project
test	make	{project}	echo testing {project}
deploy	deploy		echo deploying {env}
```

### Fields

| Field   | Required | Description |
|---------|----------|-------------|
| `name`  | Yes      | Command name |
| `group` | No       | Group label for filtering |
| `workdir`   | No       | Working directory (leave empty to use current). Supports `{placeholders}` |
| `cmd`   | Yes*     | Command to execute. Supports `{placeholders}` |
| `shell` | No       | `"ps"` for PowerShell (`powershell` on Windows, `pwsh` on Linux/macOS), omit to use the platform default (`cmd /C` on Windows, `sh -c` elsewhere) |
| `slots` | No       | Maps slot names to list names (see Placeholders). `vars` is accepted as a legacy alias (and is the TSV column name) |
| `template` | No    | Name of a slotted command this entry derives from (used by TUI-created commands) |
| `values` | No      | Slot values applied to the template. The baked `cmd` is recomputed from `template` + `values` on every load, so template edits propagate |

*Required unless `template` is set.

### Template-derived commands

Any command containing `{slots}` can act as a template. **Manage commands → Create command → From template** picks one, fills its slots (each can be skipped to resolve at run time), and saves the result — the identity goes to `commands.local.json`, the chosen values to `vars.tsv` (see Project Variables):

```json
{ "name": "build-api", "template": "build" }
```

```
command	name	value
build-api	projDir	Z:\api
build-api	projCmd	Z:\api
```

The actual command line is recomputed from `template` + `vars.tsv` on every load, so editing either propagates immediately. Deleting the template breaks its derived commands, which the startup warning points out.

## Placeholders and Selection Lists

Use `{name}` placeholders in `cmd` or `workdir` to prompt for a value at runtime.

### Selection lists

Create lists via **Manage lists** from the main menu. Each list is stored as a `.tsv` file in `projects/<name>/lists/`:

```
/home/user/api    api
/home/user/web    web
/home/user/worker worker
```

By default, `{name}` selects from the list named `name`. The same placeholder in `cmd` and `workdir` is prompted once and applied to both.

### slots — mapping slot names to lists

Use `slots` to map different slot names to the same list:

```json
{ "workdir": "{projDir}", "cmd": "echo building {projCmd}", "slots": { "projDir": "project", "projCmd": "project" } }
```

Both `{projDir}` and `{projCmd}` will select from the `project` list, each prompted separately.

### Placeholder resolution

- **Run commands** — baton prompts for each placeholder before execution
- **Workflows and aliases** — baton prompts when creating; values are saved and reused at run time
- Placeholders can be **skipped** when creating a workflow or a template-derived command — skipped ones are prompted at run time instead

### Placeholder picker

When writing a command directly (**Manage commands → Create command → Write directly**), the form asks for name, cmd, workdir, group, and shell (leave empty for the platform default, or `ps` for PowerShell). Press `Tab` in the cmd / workdir field to open a two-pane picker window:

- the **left pane** lists the selection lists — `Enter` inserts a `{placeholder}` at the cursor
- the **right pane** (focus with `→`) shows the selected list's entries — `Enter` inserts the concrete value instead

Hand-typed `{slots}` are validated as you type: `✓` when a matching list exists, `⚠` when the value will fall back to free input at run time.

#### Navigation during placeholder selection

- Type to filter the list
- `Esc` — clear the filter if active, otherwise go back
- `Enter` — confirm selection
- On the **Confirm variables** screen: `Confirm` to save, `Edit` to re-pick values

## Project Variables

`vars.tsv` in the project folder is the project's variable table — three columns: which command the entry belongs to (`*` = global), the variable name, and the value:

```
command	name	value
*	root	C:\Users\you\work\Phase2
build-api	workdir	{$root}\api
```

**Global variables** (`*` rows) are hand-written constants, referenced as `{$name}` — note the `$` — in `cmd`, `workdir`, or list values:

```
name	group	workdir	cmd
build	make	{$root}\api	make build
deploy	deploy		deploy --env {env} --dir {$root}
```

**Per-command fixed values** (rows with a command name) are written by baton itself: when you save a template-derived command, the slot values you picked land here instead of inside `commands.local.json`. Every fixed value of every saved command is editable in this one file, and baton keeps the table in sync when commands are renamed or deleted.

When the project moves — a new phase folder, another drive, someone else's machine — edit `root` (or find & replace inside this single file) and every command, saved value, and list entry that references it follows.

Rules:

- `{$name}` is substituted silently in previews and at run time; it never prompts. It is invisible to the interactive `{slot}` machinery — a plain `{root}` elsewhere is a normal slot and is never captured by a variable of the same name.
- Undefined references stay literal and are reported as a startup warning.
- Substitution is a single pass; values are never expanded recursively.
- Edits are picked up on the next start or **Switch config**.

## Usage

```
baton [--dry-run]
```

`--dry-run` prints what would be executed without running any commands.

```
  [ baton ]

  ▶ Run workflow
    Run commands
    Manage workflows
    Manage commands
    Manage aliases
    Manage lists
    Switch config
    Exit
```

### Selecting commands

| Key | Action |
|-----|--------|
| `↑` / `↓` | Move cursor |
| `Tab` | Toggle selection |
| `Enter` | Confirm |
| `Esc` | Clear the search if active, otherwise go back (with selections, a second press confirms discarding them) |
| Type | Search — matches name, group, command body, and embedded values; space-separated terms are ANDed (e.g. `make auth`) |

### Workflows

Workflows are saved combinations of commands with pre-set placeholder values. Create one via **Manage workflows → Create workflow**, then run it instantly from **Run workflow**.

The **Run workflow** list is searchable the same way as the command selector: type to filter, space-separated terms are ANDed, and matches cover the workflow name, the commands it runs, and preset placeholder values.

### Aliases

Aliases combine multiple commands into a single selectable item. Create one via **Manage aliases → Create alias**.

In the command selection screen, aliases appear with an `@` prefix and `[alias]` group tag:

```
  [ ] @ clean-build  [alias]  clean > build
```

Selecting an alias expands to its component commands at execution time.

### Retry on failure

When a command fails, baton offers a recovery menu:

```
  ▶ Retry from step 2
    Retry all
    Abort
```

On each subsequent retry, the header shows `(retry #N)` so you can tell a retry is in progress.

## Working Directory

The `workdir` field sets the working directory for a command. It accepts any absolute path or a `{placeholder}`:

```json
{ "name": "build", "workdir": "{project}", "cmd": "make build" }
```

Leave `workdir` empty to inherit the current working directory when baton is launched.

## License

MIT License — see [LICENSE](LICENSE) for details.
