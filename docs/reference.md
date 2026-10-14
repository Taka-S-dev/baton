# baton reference

The complete description of baton's files, screens and keys. The [README](../README.md) is the short version.

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
│   └── tui/                  # Bubbletea Model / Update / View, one file group per screen
│       ├── model.go, update.go, view.go, styles.go # Model struct, message dispatch, shared rendering
│       ├── update_*.go                             # Menu, run, slot resolution, command/variable/list editing, rename repair
│       ├── explore_*.go, view_explore.go           # Edit project: tabs, in-place editors, action popup, leave prompt
│       ├── run_*.go                                # Run commands / Run workflow: queue, step review, mouse
│       ├── projects.go, run_home.go                # Project picker and Run tabs
│       └── help.go, syntax.go, search_highlight.go # Help overlay, placeholder colouring, match highlighting
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
3. `~/.config/baton/projects/` — searched on every platform, and the
   usual home for a system install on Linux / macOS

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

If multiple projects exist, baton shows a project picker on startup, most recently opened first; projects never opened follow in name order. Opening a project goes directly to the Run workspace: **Workflows** and **Commands** tabs. Use `1` / `2` (or click a tab) to switch, `e` to open the selected item in the project editor, and `p` to open the project picker (`Esc` also goes back).

The last Run tab is remembered per project in the app-managed `.last_run_tab`. On first use, projects with workflows open Workflows; otherwise they open Commands. An empty project offers `e` to add its first command. Switching Run tabs and returning from the editor preserve searches, cursors, and command execution order; deleted commands are removed from that selection. `baton r` remains the separate directory-scoped recent-command launcher.

## Configuration

### Growing a project

A project starts as one file with one line, and every feature is an optional upgrade on top of it. The typical path:

1. **One command.** Create `commands.tsv` with the header and a row — name and cmd are the only required columns. baton now lists and runs it.
2. **Ask at run time.** Write `make {target}` and baton prompts for `{target}` on each run (free-text for now).
3. **Pick from a list.** Add `lists/target.tsv`, one value per line — the prompt becomes a searchable picker.
4. **Extract machine-specific paths.** Move `C:\work\...` into a `*` row of `vars.tsv` and write `{$root}` instead — relocating the project becomes a one-line change.
5. **Save filled-in variants.** In the TUI, **Edit project → Commands → Add → From template** turns any slotted command plus concrete values into a new named command. Its files are app-managed — nothing for you to edit.
6. **Chain into workflows.** **Edit project → Workflows** strings commands into a sequence that runs on one Enter.

Steps 1–4 are plain file edits; 5–6 live in the TUI. The complete file spec — every column, what may be left empty, what references what by name — is [AGENTS.md](../AGENTS.md), written for hand-editors and AI agents alike: point an agent at that file and `baton check`, and it can generate or refactor a whole project for you.

### Which file do I edit?

One kind of data, one editable home. Everything on the left column can be edited by hand (or through the matching TUI screen); the files on the last two rows are app-managed and never need hand edits:

| To change | Edit |
|------|------|
| Command definitions — name, cmd, workdir, group, shell, slots | `commands.tsv` / `commands.json` (or **Edit project → Commands**) |
| A command's name only | the `name` column alone — references are repaired on the next start (see [Renaming commands](#renaming-commands)) |
| Project variables — `{$name}` values | the `*` rows of `vars.tsv` (or **Edit project → Variables**) |
| A saved command's fixed slot values | its rows in `vars.tsv` (or **Edit project → Commands → Edit → Change values**) |
| Selection-list entries | `lists/<name>.tsv` (or **Edit project → Lists**) |
| Workflows — steps, order, names | the TUI (**Edit project → Workflows**) — `workflows.json` is app-managed |
| A saved command's identity or template | the TUI (**Edit project → Commands**) — `commands.local.json` is app-managed |

Hand edits are picked up on the next start or **Switch project**; `baton check` verifies everything still resolves.

A project has two command layers:

| File | Written by | Contents |
|------|-----------|----------|
| `commands.tsv` | Shared: you edit freely, and the TUI appends new rows (**Write directly**) or rewrites/deletes **exactly the row you target** via Edit/Delete command. Lines you didn't touch are preserved byte-for-byte, and every write re-reads the file first | Command definitions — plain ones, and slotted ones that double as templates |
| `commands.json` | You, by hand only — the TUI shows its entries as read-only | The same definitions in JSON |
| `commands.local.json` | baton (via **Edit project → Commands → Add → From template**) | Template-derived commands (identity only; values live in vars.tsv) |

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

Any command containing `{slots}` can act as a template. **Edit project → Commands → Add → From template** picks one, fills its slots (each can be skipped to resolve at run time), and saves the result — the identity goes to `commands.local.json`, the chosen values to `vars.tsv` (see Project Variables):

```json
{ "name": "build-api", "template": "build" }
```

```
command	name	value
build-api	projDir	Z:\api
build-api	projCmd	Z:\api
```

The actual command line is recomputed from `template` + `vars.tsv` on every load, so editing either propagates immediately. Deleting the template breaks its derived commands, which the startup warning points out.

**Naming.** Saving one pre-fills a name built from the template and the values picked in slot order — `build` with `workdir=./src` and `mode=Fast Build` becomes `build-src-fast-build` — so saved commands stay distinguishable without inventing a scheme each time. `Tab` inserts that name in any name input, which is what brings an entry named before the rule existed back in line with it: **Edit project → Commands → Rename**, then `Tab`. Renaming a workflow works the same way, offering the `step1+step2` name built from its steps.

### Renaming commands

Command names are the reference key everywhere — workflow steps, the `template` field of derived commands, and per-command `vars.tsv` rows all point at them. Renames keep those references intact from either direction:

- **In the TUI** (Edit command), every reference is rewritten in the same save.
- **By hand in `commands.tsv` / `commands.json`** — just change the name. baton records each command's name and content fingerprint in `.command_names` (app-managed, like `.last_workflow`), so the next start recognizes "a referenced name vanished, the same content reappeared under a new one" as a rename and offers a one-key repair before the Run workspace:

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

Create lists via **e → Lists** from the Run workspace. Each list is stored as a `.tsv` file in `projects/<name>/lists/`:

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

### Writing a command directly

**Edit project → Commands → Add → Write directly** (or `a` on the
Commands tab) opens a form for name, cmd, workdir, group, and shell (leave
empty for the platform default, or `ps` for PowerShell). The result is
**appended to the project's TSV** — the form is a guided way to author a
hand-editable row without remembering the column layout or placeholder
syntax.

The keys mean one thing each: `Tab` / `Shift+Tab` move to the next and
previous field, `Enter` moves on (and saves from the last field),
`Ctrl+S` saves the form. `Esc` closes an open picker or returns from editing;
unsaved changes are protected by a leave confirmation.

The form asks for the cmd first and the name second, because the name is
the only part that takes thought. An empty name field shows the name the
cmd suggests, dimmed in place (`go test ./internal/{pkg}/` suggests
`go-test`); `→` takes it.

Typing `{` in the cmd or workdir field opens the candidates for what can
go there, each with what choosing it means:

```
  Cmd > go test ./{
  ▶ {pkg}        pick at run time · 5 choices
    {pkg...}     pick several at run time
    {$root}      fixed · C:\work\Phase1
```

What you type after the brace narrows them; `↑↓` choose, `Enter` inserts
the placeholder in place of the brace, `Esc` leaves what you typed as it
is. Hand-typed `{slots}` are validated as you go: `✓` when a matching
list exists, `⚠` when the value will fall back to free input at run time.

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

### Editing variables and fixed values

**Edit project → Variables** edits shared globals. Saved command answers are edited through **Commands → Change values**:

- **Globals**: create, change a value (with a "referenced by" preview), delete. Changing a value is not string replacement — everything written as `{$root}` follows automatically because it references the variable, so unrelated values that merely look similar can never be caught.
- **Creating a global extracts literals**: if existing values match the new variable's value, baton offers to rewrite them into `{$name}` references on the spot (resolution unchanged). That is the intended way to make several values move together — two rows that merely hold the same string are never propagated into each other.
- **Fixed values** (`command.slot` rows): use **Commands → Change values** to edit a saved answer. Deleting one **un-fixes the slot**: that placeholder is prompted at run time again. New rows are never created here; they appear when you save a template-derived command.
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
- Edits are picked up on the next start or **Switch project**.

### Renames inside vars.tsv

Unlike command renames, name changes inside `vars.tsv` are not auto-repaired — the columns of this file are the references themselves:

- **Renaming a global** (`*` row): every `{$oldname}` reference stays literal, and the startup warning names each command and list still using it. Update the references yourself, or create the new name via **Edit project → Variables** and let the rebase offer rewrite the values.
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
lives in [AGENTS.md](../AGENTS.md).

The Run workspace has **Workflows** (`1`) and **Commands** (`2`) tabs. `e` opens **Edit project** at the current tab and item; `Esc` goes back to the project picker; clear search and confirm discarding selections first if needed. `p` opens the project picker directly. `F1` shows the available keys; `q` quits outside search; `Ctrl+C` also quits.

### Searching

- Projects, Recent, the editor, Run lists and workflow command pickers: `/` starts or resumes search.
- `↓` focuses the current result without clearing the query; list actions such as `e` then work.
- `Enter` opens the result or proceeds to input/review; with no result or queued command, search stays active.
- In search, `Esc` clears text first, then exits when empty; in a result list it clears a retained filter before going back.
- File/folder pickers accept text immediately; `↑↓` selects candidates and `Tab` checks multiple values.

### Projects

`Enter` opens a project, `a` adds one, and `r` refreshes. `Space` or right-click opens actions; `o` opens the settings folder and `c` copies its path.

`Del` / `x` asks to delete a project. This moves its baton settings into `.baton-trash` under the projects directory; command working directories are untouched. Restore an archived project by moving its folder back into the projects directory.

### Project editor

Open **Edit project** for the shared workspace:

```text
[1 Workflows] [2 Commands] | [3 Lists] [4 Variables]

Left: items and search       Right: selected item / editor
                            Below: preview and references
```

Workflows reference commands in order. Commands reference shared lists through interactive slots and shared variables through `{$name}`. Fixed slot answers belong to a saved command; edit them through **Commands → Change values**. The Variables tab shows shared globals.

Command rows use explicit types: **Direct**, **Template**, or **From: template-name**. Group is a label, not a tree level. Lists show their entries and references; while editing an entry, the detail pane describes that value.

| Key / action | Effect |
|---|---|
| `1`–`4`, tab-label click | Switch category |
| `↑↓` | Select an item |
| `/` | Search the category |
| `Enter` | Open or edit |
| `a`, Add button | Add to the focused pane; `n` also creates a project item |
| `Space`, right-click | Contextual actions |
| `Del` / `x` in the item list | Request deletion |
| `Ctrl+S` in an editor | Save changes |
| `Esc`, Back | Return one level; unsaved changes prompt before leaving |
| `F1` | Help, including while typing |
| `?` in the item list | Help |

**Commands.** Create directly or from a template. The form provides placeholder candidates and shell choices, with live preview and validation. The action menu links commands to referenced lists and variables. Clipboard operations are available from input context menus.

**Workflows.** Open one to edit its steps. `a` inserts after the cursor, `J/K` reorder, and `x` removes the step reference. `Enter` opens its command. Removing a step does not delete that command. `Ctrl+S` saves the working copy; leaving with changes offers Save and leave, Discard changes, or Keep editing.

**Lists and variables.** Add buttons remain available when empty. Lists can be edited in the right pane; `a` adds values without scrolling to the last row. Global variable edits show references.

On narrow terminals, the editor switches to list and detail screens with compact tabs. `Esc` returns from details. Confirmation dialogs retain the underlying screen.

### Run commands

Select commands on the left and review their execution order in **Run queue**. The Run column shows each selected command’s current execution position, synchronized with queue reordering and removal. Unselected rows show `—`, and filtering preserves the positions. Other columns show Name, Group and Type; narrow panes omit Group. A separate Match column explains search hits in command text or fixed values when space permits.

The queue shows order, name and command text, with long rows abbreviated and placeholders highlighted. The detail pane shows command information, working directory and shell. Wide screens share one detail pane; narrow screens switch between candidates and queue.

| Key | Action |
|---|---|
| `s` in candidates | Cycle sort order |
| `↑↓` | Move within the focused list |
| `Tab` in candidates | Select / deselect |
| `F2` | Switch candidates / queue (`Ctrl+Q` also works) |
| `J/K` or `Shift+↓/↑` in queue | Reorder |
| `Del` / `x` in queue | Remove from this run, keeping the command definition |
| `d`, or a click on the selected queue row | Open the details popup for the highlighted command; `Esc` or a click outside closes it |
| `PgUp` / `PgDn`, mouse wheel | Scroll the details; the wheel scrolls the pane under the pointer |
| Drag the divider, right-click it | Resize the two panes, reset the width |
| `Enter` | Resolve inputs and proceed to execution review |
| `Esc` in queue | Return to candidates |
| `Esc` in candidates | Clear the filter first; otherwise leave, asking before discarding selections |
| `F1` | Help |

A queue row shows the position, the name and the start of the command; the details box below wraps the full command and working directory. Scrollbars appear only when a pane overflows — drag the thumb or click the track to page.

With nothing selected, `Enter` in candidates uses the highlighted command. In an empty queue, `Enter` does nothing. The discard dialog defaults to **Keep selecting**. `Esc` cancels it and preserves the selection; choose **Discard and go back** and press `Enter` to leave. Execution starts only after the final run confirmation.

### Run workflow

Create saved sequences in **Edit project → Workflows**. The Run workflow list searches workflow names, step names and command bodies. Wide screens show workflows and execution steps side by side.

| Key | Action |
|---|---|
| `↑↓` | Choose a workflow |
| `Enter` in the workflow list | Resolve inputs and review all steps |
| `Tab` / `→` | Choose steps for this run |
| `F2` in the workflow list | Open step details |
| `↑↓` in step selection | Inspect a step |
| `Tab` in step selection | Select / deselect a runnable step |
| Click a step / checkbox | Inspect its details / toggle selection |
| `F2` in step selection | Switch step / detail views on narrow screens; wide screens show read-only details |
| `Enter` in step selection | Review selected steps; select at least one step first |
| `Esc` | Return one level; the workflow list clears an active filter before leaving |

Details show the command, working directory and shell. Missing commands remain visible but cannot be selected. Selections apply **only to this run** and retain saved workflow order. They do not change the saved workflow. Remaining slots are prompted before execution review; fixed answers come from the saved commands. `F1` shows help.


### Recent commands in the current directory

```powershell
baton recent
baton r                 # alias
baton recent --dry-run
```

Open the current launch directory's recent commands directly, newest first.
`Tab` adds the highlighted command to a queue (or removes it); the rows show the
queue position, and `Enter` reviews the queue in that order. The queue may mix
projects: each command runs with its own project's variables, lists and working
directory. With nothing queued, `Enter` reviews the highlighted command alone:
it reloads the current definition,
prompts for interactive inputs (including file completion), then shows execution
review. On success, recent mode returns directly to the shell after one
completion summary; failures retain the retry choices. Explicit workdir settings still apply; an empty workdir uses the launch
directory. `p` opens projects, `q` quits, and `Ctrl+R` refreshes the list.
Search matches are highlighted in candidates and the command preview; each row
shows its last execution time. Delete removes only the selected reference from
this directory’s history, leaving the command definition intact. Ctrl+Z restores
the last removed reference with its original timestamp during this session.

Normal runs and workflow steps also record history when a step is started.
Dry runs, canceled selections, and steps never reached are not recorded.
History stores only the project path, command name, launch directory, and time;
it does not retain runtime input values or resolved command lines. A separate
file-picker preference remembers the last confirmed path for cursor placement
only, scoped by project, command, input token, and resolved working directory.
It never checks or confirms a value automatically; missing paths fall back to
the normal candidates. This preference is saved when the input is confirmed,
even if the subsequent execution review is canceled. Saved commands
keep their explicitly fixed values. Repeated commands occupy one entry per
project and directory; the latest 100 are shown. Deleted or renamed references
are shown as unavailable. Browse projects is always available, including when
this directory has no history yet. Runs from older versions cannot be recovered.

History is app-managed under `projects/.baton-recent/` and is excluded from the
project list and `baton check`.

### File and folder inputs

Ordinary `{target}` slots retain their list/free-text behavior, including
Tab to toggle variadic selections. Fixed variables `{$root}` are unchanged.
Path selection is opt-in through the reserved `{@...}` syntax. In the command
or workdir editor, type `{@` to see path placeholders and extension-filter
examples; use Up/Down and Enter to insert one. Edit the example extensions
to match your command.


| Placeholder | Input |
|---|---|
| `{@file}` | One file |
| `{@file:.go}` | One .go file |
| `{@file:.go,.mod}` | One .go or .mod file |
| `{@file...:.go,.mod}` | Multiple .go / .mod files |
| `{@dir}` | One directory |

Extensions include the leading dot, are comma-separated without spaces, and
match case-insensitively. Compound suffixes such as .tar.gz are supported.
Directory inputs do not accept extension filters. The `@` namespace is reserved;
invalid forms are reported by `baton check`.

The dedicated dropdown opens at the input step. Type to fuzzy-filter, use
arrows to move. File pickers show files first, followed by navigation folders without checkboxes.
Enter or Right opens a navigation folder, keeping checked files.
Inside a folder, Left or the parent row returns one level and restores focus
to the folder just left. The parent row is never selectable as a value. Only one
directory is listed at a time; there is no recursive search. Typing a directory
prefix such as `src/` also works. Directory pickers support Right to browse.
No registered list is mixed in.
For multiple values, Tab or a click toggles checkboxes, Enter confirms the
selection in selection order, and Esc clears a nonempty filter first.
With an empty filter, Esc closes suggestions and keeps checked values. Selections survive
filtering and navigation. Enter confirms the input and continues to the next input or execution review.
For a single value, Tab or a click inserts it for further editing.
Tab or Ctrl+Space reopens the dropdown. During template creation, Ctrl+K keeps
the input interactive instead of saving an answer.

Paths are relative to the resolved workdir, or the launch directory when it is
empty. Workdir inputs are resolved first for path commands; a slot that supplies
workdir itself browses from the launch directory. Only the directory being
browsed is read. Values remain literal: quote single-value slots for paths with
spaces; variadic slots retain the existing space-joined argument semantics.
Identical reserved tokens share one answer; different filters are distinct
inputs. Saved values use an encoded internal key in vars.tsv; edit only their
value column as with other command-scoped rows.

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
