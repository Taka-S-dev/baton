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
- Save and reuse command sequences as workflows
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
│   ├── model/                # Shared data types (Command, Workflow, RunItem)
│   ├── config/               # Config loading (JSON/TSV), projects/ discovery, diagnostics
│   ├── slot/                 # {placeholder} and {$var} resolution, lists and vars.tsv
│   ├── store/                # App-managed state (workflows, saved commands, last workflow)
│   ├── runner/               # Command execution via tea.ExecProcess (suspends TUI)
│   └── tui/                  # Bubbletea Model / Update / View
│       ├── model.go                  # Model struct, screen enum, sub-states, New()
│       ├── update.go                 # Update() entry point, message dispatch
│       ├── update_menu.go            # Project select, main menu, config switch
│       ├── update_run.go             # Run workflow, per-run step selection, confirm, retry
│       ├── update_resolve.go         # Multi-select, slot resolution and answer reuse
│       ├── update_workflow.go        # Workflow CRUD
│       ├── update_manage_commands.go # Command CRUD (direct input / from template)
│       ├── update_vars.go            # Project variable CRUD, reference rebase offer
│       ├── update_rename.go          # Hand-edited rename detection and reference repair
│       ├── update_list.go            # Selection lists and name-input screens
│       ├── update_helpers.go         # Cursor, pick-filter and delete-flow helpers
│       ├── view.go                   # All rendering functions
│       └── styles.go                 # Lipgloss styles and helper render functions
└── projects.example/         # Sample projects (JSON and TSV) to copy as a starting point
```

The TUI follows the standard [Bubble Tea](https://github.com/charmbracelet/bubbletea) architecture (Elm-style Model/Update/View). Each screen has a corresponding `update*` and `view*` function. Slot resolution for runs is driven by a common `resolveFlowState` that walks the selected items and prompts for their placeholders.

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

### Growing a project

A project starts as one file with one line, and every feature is an optional upgrade on top of it. The typical path:

1. **One command.** Create `commands.tsv` with the header and a row — name and cmd are the only required columns. baton now lists and runs it.
2. **Ask at run time.** Write `make {target}` and baton prompts for `{target}` on each run (free-text for now).
3. **Pick from a list.** Add `lists/target.tsv`, one value per line — the prompt becomes a searchable picker.
4. **Extract machine-specific paths.** Move `C:\work\...` into a `*` row of `vars.tsv` and write `{$root}` instead — relocating the project becomes a one-line change.
5. **Save filled-in variants.** In the TUI, **Create command → From template** turns any slotted command plus concrete values into a new named command. Its files are app-managed — nothing for you to edit.
6. **Chain into workflows.** **Manage workflows** strings commands into a sequence that runs on one Enter.

Steps 1–4 are plain file edits; 5–6 live in the TUI. The complete file spec — every column, what may be left empty, what references what by name — is [AGENTS.md](AGENTS.md), written for hand-editors and AI agents alike: point an agent at that file and `baton check`, and it can generate or refactor a whole project for you.

### Which file do I edit?

One kind of data, one editable home. Everything on the left column can be edited by hand (or through the matching TUI screen); the files on the last two rows are app-managed and never need hand edits:

| To change | Edit |
|------|------|
| Command definitions — name, cmd, workdir, group, shell, slots | `commands.tsv` / `commands.json` (or **Manage commands**) |
| A command's name only | the `name` column alone — references are repaired on the next start (see [Renaming commands](#renaming-commands)) |
| Project variables — `{$name}` values | the `*` rows of `vars.tsv` (or **Manage vars**) |
| A saved command's fixed slot values | its rows in `vars.tsv` (or **Edit command → Change values**) |
| Selection-list entries | `lists/<name>.tsv` (or **Manage lists**) |
| Workflows — steps, order, names | the TUI (**Manage workflows**) — `workflows.json` is app-managed |
| A saved command's identity or template | the TUI (**Manage commands**) — `commands.local.json` is app-managed |

Hand edits are picked up on the next start or **Switch config**; `baton check` verifies everything still resolves.

A project has two command layers:

| File | Written by | Contents |
|------|-----------|----------|
| `commands.json` / `commands.tsv` | Shared: you edit freely, and the TUI appends new rows (**Write directly**) or rewrites/deletes **exactly the row you target** via Edit/Delete command. Lines you didn't touch are preserved byte-for-byte, and every write re-reads the file first | Command definitions — plain ones, and slotted ones that double as templates |
| `commands.local.json` | baton (via **Create command → From template**) | Template-derived commands (identity only; values live in vars.tsv) |

Both layers can coexist; names in `commands.local.json` take priority.

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
name	group	workdir	cmd	shell	slots
build	make	{projDir}	echo building {projCmd}		projDir=project,projCmd=project
test	make	{project}	echo testing {project}
deploy	deploy		echo deploying {env}
```

### Fields

| Field   | Required | Description |
|---------|----------|-------------|
| `name`  | Yes      | Command name |
| `group` | No       | Group label for filtering. Referenced by nothing — rename or empty it freely. A template-derived entry with an empty group inherits the template's |
| `workdir`   | No       | Working directory (leave empty to use current). Supports `{placeholders}` |
| `cmd`   | Yes*     | Command to execute. Supports `{placeholders}` |
| `shell` | No       | `"ps"` for PowerShell (`powershell` on Windows, `pwsh` on Linux/macOS), omit to use the platform default (`cmd /C` on Windows, `sh -c` elsewhere). A template-derived entry with an empty shell inherits the template's |
| `slots` | No       | Maps slot names to list names (see Placeholders) |
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

**Naming.** Saving one pre-fills a name built from the template and the values picked — `build` on `./src` in fast mode becomes `build-fast-build-src` — so saved commands stay distinguishable without inventing a scheme each time. `Tab` inserts that name in any name input, which is what brings an entry named before the rule existed back in line with it: **Edit command → Rename**, then `Tab`. Renaming a workflow works the same way, offering the `step1+step2` name built from its steps.

### Renaming commands

Command names are the reference key everywhere — workflow steps, the `template` field of derived commands, and per-command `vars.tsv` rows all point at them. Renames keep those references intact from either direction:

- **In the TUI** (Edit command), every reference is rewritten in the same save.
- **By hand in `commands.tsv` / `commands.json`** — just change the name. baton records each command's name and content fingerprint in `.command_names` (app-managed, like `.last_workflow`), so the next start recognizes "a referenced name vanished, the same content reappeared under a new one" as a rename and offers a one-key repair before the main menu:

```
  Rename detected

  1 command(s) look renamed outside baton:

    apistart → api-start   2 workflow step(s)

  Update these references to the new names?

  [  No  ] [ ▶Yes  ]
```

The match is by content, so renaming a command *and* editing its definition in the same session is not detectable — that case falls back to the usual broken-reference warnings. Declining keeps every file untouched and the offer is not repeated.

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

### Variadic placeholders — `{name...}`

Write `{name...}` when a placeholder should accept **multiple values** from its list:

```
compose-up	docker	docker compose up {services...}
```

At run time the picker switches to multi-select — `Tab` toggles entries (typed custom values can be toggled in too), `Enter` confirms, and the values are joined with single spaces: `docker compose up api web worker`. With nothing toggled, `Enter` picks the hovered entry alone, exactly like a normal slot.

Saved commands can fix a variadic slot the same way as any other — the stored value is simply the joined string.

> Note: values containing spaces are not quoted when joined, so variadic slots work best with space-free values (service names, package paths, flags).

### Placeholder resolution

- **Run commands / Run workflow** — baton prompts for each placeholder before execution
- **Fixed values** belong to saved commands: create one from a template with the values filled in (they land in `vars.tsv`), and use that command anywhere — directly, or inside workflows
- Placeholders can be **skipped** when creating a template-derived command — skipped ones are prompted at run time instead
- **Repeated questions carry their answer forward.** When several commands in one run share a placeholder — `clean` and `build` both asking for the same directory — the later picker opens on the earlier answer and says where it came from (`same as clean — Enter keeps it`), so repeating it costs one keypress. The answer is never applied on your behalf: picking a different value works exactly as before, so a workflow that deploys to staging and then to production still asks twice. Two placeholders count as the same question only when both the name and the list behind them match, so a `{target}` mapped to different lists in the `slots` column is always asked separately.

### Placeholder picker

When writing a command directly (**Manage commands → Create command → Write directly**), the form asks for name, cmd, workdir, group, and shell (leave empty for the platform default, or `ps` for PowerShell). The result is **appended to the project's TSV** — the form is a guided way to author a hand-editable row without remembering the column layout or placeholder syntax. Press `Tab` in the cmd / workdir field to open a two-pane picker window:

- the **left pane** lists the selection lists — `Enter` inserts a `{placeholder}` at the cursor
- the **right pane** (focus with `→`) shows the selected list's entries — `Enter` inserts the concrete value instead

Hand-typed `{slots}` are validated as you type: `✓` when a matching list exists, `⚠` when the value will fall back to free input at run time.

#### Navigation during placeholder selection

- Type to filter the list
- `Esc` — clear the filter if active, otherwise go back
- `Enter` — confirm selection

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

When the project moves — a new phase folder, another drive, someone else's machine — change `root` once and every command, saved value, and list entry that references it follows.

### Manage vars

**Manage vars** in the TUI shows the whole table — globals first, then every saved command's fixed values — and edits it in place:

- **Globals**: create, change a value (with a "referenced by" preview), delete. Changing a value is not string replacement — everything written as `{$root}` follows automatically because it references the variable, so unrelated values that merely look similar can never be caught.
- **Creating a global extracts literals**: if existing values match the new variable's value, baton offers to rewrite them into `{$name}` references on the spot (resolution unchanged). That is the intended way to make several values move together — two rows that merely hold the same string are never propagated into each other.
- **Fixed values** (`command.slot` rows): edit the value directly — a shortcut past **Edit command → Change values** when you just want to fix a path or a typo. Deleting one **un-fixes the slot**: that placeholder is prompted at run time again. New rows are never created here; they appear when you save a template-derived command.
- **Editing a fixed value whose old value other rows share** opens a "change matching values too?" offer — same prefix-anchored, checkbox-previewed mechanics as the rebase window, applied to the literals you check. Nothing is ever propagated silently; for values that should *always* move together, extract a global instead.

Values that contain the old value as a **literal** (a hand-typed path in a saved value or a list entry) don't follow by themselves. After a change, baton scans for literals that *start with* the old value and offers to rewrite them into `{$name}` references:

```
  [ Rebase values onto {$root} ]
  ▶ [x] saved value build-api.workdir   C:\demo\phase1\api  →  {$root}\api
    [x] list project                    C:\demo\phase1\web  →  {$root}\web
    [ ] list project                    C:\demo\phase1docs  →  (odd boundary: off by default)
  ↑↓  Tab: toggle   Enter: apply   Esc: keep literals
```

The match is prefix-anchored — values containing the old value somewhere in the middle are never candidates — and a match whose next character isn't a path separator defaults to unchecked. Applied values become references, so the next move is a one-line change.

Rules:

- `{$name}` is substituted silently in previews and at run time; it never prompts. It is invisible to the interactive `{slot}` machinery — a plain `{root}` elsewhere is a normal slot and is never captured by a variable of the same name.
- Undefined references stay literal and are reported as a startup warning.
- Substitution is a single pass; values are never expanded recursively.
- Edits are picked up on the next start or **Switch config**.

### Renames inside vars.tsv

Unlike command renames, name changes inside `vars.tsv` are not auto-repaired — the columns of this file are the references themselves:

- **Renaming a global** (`*` row): every `{$oldname}` reference stays literal, and the startup warning names each command and list still using it. Update the references yourself, or create the new name via **Manage vars** and let the rebase offer rewrite the values.
- **Changing the command column** of a fixed-value row detaches the value: that slot is prompted at run time again, the row is flagged as belonging to an unknown command, and the next TUI save removes it.
- **Changing the name column** of a fixed-value row to something that isn't a slot of the template: the value stops applying, the slot is prompted at run time again, and the startup warning flags the row.

In short: rename commands in `commands.tsv`, rename variables through their references, and treat the first two columns of `vars.tsv` as addresses rather than free text.

## Usage

```
baton [--dry-run]
baton check [project|path]
```

`--dry-run` prints what would be executed without running any commands.

`baton check` validates projects without starting the TUI: it prints every
warning (undefined variables, missing templates, unknown shells, broken
workflow references, …) and exits non-zero when anything is wrong. Use it
in CI, or as the verification loop when generating project files with an
AI agent — the complete file spec, shared by hand-editors and agents,
lives in [AGENTS.md](AGENTS.md).

```
  [ baton ]

  ▶ Run workflow
    Run commands
    Manage workflows
    Manage commands
    Manage lists
    Switch config
    Exit
```

### Selecting commands

| Key | Action |
|-----|--------|
| `↑` / `↓` | Move cursor |
| `Tab` | Toggle selection |
| `Enter` | Confirm — with nothing toggled, acts on the hovered row alone |
| `Esc` | Clear the search if active, otherwise go back (with selections, a second press confirms discarding them) |
| Type | Search — matches name, group, command body, and embedded values; space-separated terms are ANDed (e.g. `make auth`) |

### Workflows

Workflows are saved sequences of commands. Creating one is just picking commands and naming them (**Manage workflows → Create workflow**); remaining `{slots}` are prompted at run time, and fixed values belong to the saved commands the workflow contains.

Run one from **Run workflow** — the list is searchable like the command selector: the workflow name, its step names, and the step command bodies all match.

**Running part of a workflow.** `Enter` runs every step; `→` opens the step picker for the highlighted workflow, where `Tab` toggles individual steps and `Enter` runs the chosen ones. Selected steps always execute in the workflow's own order — the picker chooses *which* steps run, never the order — so a fixed sequence you deliberately advance a few steps at a time stays one workflow instead of a search each time. With nothing toggled, `Enter` runs just the highlighted step, and steps whose command no longer exists are shown but cannot be selected.

The management pick screens (Edit/Delete of commands, workflows, and lists) are searchable the same way — just start typing to filter; `Esc` clears the filter first, then goes back.

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
