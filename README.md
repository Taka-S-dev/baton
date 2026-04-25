# baton

```
  _           _
 | |__   __ _| |_ ___  _ __
 | '_ \ / _` | __/ _ \| '_ \
 | |_) | (_| | || (_) | | | |
 |_.__/ \__,_|\__\___/|_| |_|
```

A terminal-based workflow runner for Windows, Linux, and macOS. Define commands in a config file, select and execute them interactively.

## Features

- Interactive TUI with real-time search and multi-select
- Save and reuse command combinations as workflows
- Combine multiple commands into a single alias
- `{placeholder}` substitution — pick values from a selection list at runtime
- Optional `vars` field to map slot names to named lists
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
│       ├── update_list.go    # List and name-input screens
│       ├── view.go           # All rendering functions
│       └── styles.go         # Lipgloss styles and helper render functions
└── projects.example/         # Sample project to copy as a starting point
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

Requires Go 1.21+.

## Quick start

A sample project is included in `projects.example/`. Copy it to get started:

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

Create a subfolder for each project and add `config.json` or `config.tsv` inside it:

```
~/.config/baton/projects/    (Linux/macOS)
├── projectA/
│   └── config.json
└── projectB/
    └── config.tsv
```

```
any-folder/                  (Windows, or portable install)
├── baton.exe
└── projects/
    ├── projectA/
    │   └── config.json
    └── projectB/
        └── config.tsv
```

If multiple projects exist, baton shows a selection screen on startup. Use **Switch config** from the main menu to switch projects at any time.

## Configuration

### config.json

```json
{
  "commands": [
    { "name": "build",  "group": "make",   "dir": "{projDir}", "cmd": "echo building {projCmd}", "vars": { "projDir": "project", "projCmd": "project" } },
    { "name": "test",   "group": "make",   "dir": "{project}", "cmd": "echo testing {project}" },
    { "name": "deploy", "group": "deploy", "dir": "",          "cmd": "echo deploying {env}" }
  ]
}
```

### config.tsv

Tab-separated alternative to `config.json`. If both files exist, `config.json` takes priority.

```
name	group	dir	cmd	shell	vars
build	make	{projDir}	echo building {projCmd}		projDir=project,projCmd=project
test	make	{project}	echo testing {project}
deploy	deploy		echo deploying {env}
```

### Fields

| Field   | Required | Description |
|---------|----------|-------------|
| `name`  | Yes      | Command name |
| `group` | No       | Group label for filtering |
| `dir`   | No       | Working directory (leave empty to use current). Supports `{placeholders}` |
| `cmd`   | Yes      | Command to execute. Supports `{placeholders}` |
| `shell` | No       | `"ps"` for PowerShell (`powershell` on Windows, `pwsh` on Linux/macOS), omit to use the platform default (`cmd /C` on Windows, `sh -c` elsewhere) |
| `vars`  | No       | Maps slot names to list names (see Placeholders) |

## Placeholders and Selection Lists

Use `{name}` placeholders in `cmd` or `dir` to prompt for a value at runtime.

### Selection lists

Create lists via **Manage lists** from the main menu. Each list is stored as a `.tsv` file in `projects/<name>/lists/`:

```
/home/user/api    api
/home/user/web    web
/home/user/worker worker
```

By default, `{name}` selects from the list named `name`. The same placeholder in `cmd` and `dir` is prompted once and applied to both.

### vars — mapping slot names to lists

Use `vars` to map different slot names to the same list:

```json
{ "dir": "{projDir}", "cmd": "echo building {projCmd}", "vars": { "projDir": "project", "projCmd": "project" } }
```

Both `{projDir}` and `{projCmd}` will select from the `project` list, each prompted separately.

### Placeholder resolution

- **Run manually** — baton prompts for each placeholder before execution
- **Workflows and aliases** — baton prompts when creating; values are saved and reused at run time

#### Navigation during placeholder selection

- Type to filter the list
- `Esc` — clear the filter if active, otherwise go back
- `Enter` — confirm selection
- On the **Confirm variables** screen: `Confirm` to save, `Edit` to re-pick values

## Usage

```
baton [--dry-run]
```

`--dry-run` prints what would be executed without running any commands.

```
  [ baton ]

  ▶ Run workflow
    Run manually
    Create workflow
    Edit workflow
    Delete workflow
    Manage aliases
    Manage lists
    Switch config
    Exit
```

### Selecting commands

| Key | Action |
|-----|--------|
| `↑` / `↓` | Move cursor |
| `Space` | Toggle selection |
| `Enter` | Confirm |
| `Esc` | Cancel / go back |
| Type | Search by name or group |
| `Tab` | Switch between name and group search fields |

### Workflows

Workflows are saved combinations of commands with pre-set placeholder values. Create one via **Create workflow**, then run it instantly from **Run workflow**.

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

The `dir` field sets the working directory for a command. It accepts any absolute path or a `{placeholder}`:

```json
{ "name": "build", "dir": "{project}", "cmd": "make build" }
```

Leave `dir` empty to inherit the current working directory when baton is launched.

## License

MIT License — see [LICENSE](LICENSE) for details.
