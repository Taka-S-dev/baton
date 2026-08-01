# baton project files — the spec

This file is the complete spec of a baton project's files — written for
humans editing them by hand and for AI agents (or scripts) generating
them. Everything needed to produce a valid project is on this page, and
`baton check` verifies the result.

## Ownership — which files you may write

| File | Edit / generate? |
|------|-----------|
| `commands.tsv` | yes — command definitions (the baton TUI also appends, edits, or deletes individual rows on user request; untargeted lines are never touched) |
| `lists/<name>.tsv` | yes — selection lists for `{slot}` placeholders |
| `vars.tsv` | `*` rows: freely. Command-scoped rows: edit the **value** column only — baton creates the rows, and the first two columns are addresses (see vars.tsv below) |
| `commands.local.json`, `workflows.json`, `.last_workflow`, `.command_names` | **never** — owned and written by the baton TUI |

## Directory layout

```
<projects-dir>/<project-name>/
├── commands.tsv
├── lists/
│   └── <listname>.tsv
└── vars.tsv            (optional)
```

The projects directory is `$BATON_PROJECTS_DIR`, else `projects/` next
to the baton executable, else `~/.config/baton/projects/`. You can also
generate a project anywhere and check it by path.

## How the files fit together

Command **names are the reference key** — every cross-file link below
is by name. Anything not listed here (notably `group`) is referenced by
nothing and can be changed or emptied freely.

```
workflows.json step ──────────────────▶ command name
template field (derived command) ─────▶ slotted command's name
vars.tsv command column ──────────────▶ saved command's name
{slot} / slots column ────────────────▶ lists/<name>.tsv
{$name} in cmd, workdir, list values ─▶ vars.tsv "*" row name
.last_workflow ───────────────────────▶ workflow name
```

## What happens on load

1. `commands.json` + `commands.tsv` load as the hand-written layer;
   `commands.local.json` as the app layer. On a name clash the app
   layer wins.
2. Command-scoped `vars.tsv` values merge into template-derived
   commands (`vars.tsv` wins over values stored in the local file).
3. Template-derived commands are re-baked from their template, so
   template edits propagate on every load. A derived entry whose
   `group` or `shell` is empty inherits the template's.
4. Diagnostics run (same output as `baton check`), and command renames
   made by hand are detected via `.command_names` (see Renames).

## Critical rule: literal tabs

Every `.tsv` file is separated by **literal TAB characters (U+0009)**.
Spaces are not separators and will silently produce wrong commands.
Encode files as UTF-8.

## commands.tsv

First line is a header and is skipped — columns are positional:

```
name	group	workdir	cmd	shell	slots
build	make	{projDir}	echo building {projCmd}		projDir=project,projCmd=project
deploy	deploy		deploy --env {env} --dir {$root}
```

| Column | Required | Rules |
|--------|----------|-------|
| name | yes | unique within the project; the key everything else references |
| group | no | free label, used for search and display; referenced by nothing |
| workdir | no | working directory; `{slot}` and `{$var}` allowed; empty = directory baton was started from |
| cmd | yes | command line; `{slot}` and `{$var}` allowed |
| shell | no | empty = `cmd.exe` on Windows / `sh` elsewhere; `ps` = PowerShell. Any other value falls back to the default and warns |
| slots | no | comma-separated `slot=listname` pairs mapping a slot to a list; an unmapped slot uses the list with the same name |

`commands.json` is the JSON equivalent (same fields, one object per
command under a `"commands"` array); both files can coexist and merge.
JSON entries may additionally carry `template` + `values` — that shape
is normally produced by the TUI, not by hand.

Two placeholder kinds — do not mix them up:

- `{name}` — **interactive slot**: the user picks a value at run time,
  from `lists/<name>.tsv` (or the list mapped in the slots column).
  If no such list exists, the picker falls back to free-text input.
- `{$name}` — **variable**: substituted silently from a `*` row in
  vars.tsv. Never prompts. Use for paths and other constants that
  change per machine or per project phase.

A slot written as `{name...}` is **variadic**: the picker allows
multiple selections and joins them with single spaces
(`docker compose up {services...}` → `docker compose up api web`).
Use it only where the command accepts space-separated arguments.
There is no `{$name...}` — variables are single fixed values.

Rows containing `{slot}` also serve as templates for the TUI's
**Create command → From template**.

## lists/<name>.tsv

**No header row** — every line is an entry (a header line would become
a selectable entry). One entry per line: `value<TAB>label`, label
optional. Values may contain `{$var}`.

```
{$root}\api	API project
{$root}\web	Web project
```

## vars.tsv

Header line, then three columns: `command<TAB>name<TAB>value`.

```
command	name	value
*	root	C:\work\Phase1
build-api	workdir	{$root}\api
```

- `*` rows are globals, referenced as `{$name}`. Generate and edit
  these freely.
- Rows naming a command hold that saved command's fixed slot values.
  baton creates them; editing the **value** is fine and propagates on
  the next load. The command and name columns are addresses: changing
  them detaches the value (the slot goes back to prompting at run
  time) and is flagged as a warning.
- Substitution is a single pass (no recursion). An undefined `{$name}`
  stays literal and is reported as a warning.

## Renames

Renaming a command is safe from either direction:

- **In the TUI** (Edit command), every reference — workflow steps,
  template fields, scoped vars.tsv rows — is rewritten in the same
  save.
- **By hand in commands.tsv / commands.json**: change the name column
  only. baton records each command's name and content fingerprint in
  `.command_names`; on the next start it recognizes the rename and
  offers a one-key repair of all references. Changing the name *and*
  the content in the same edit defeats the match — the repair falls
  back to warnings, so do those as two separate edits.

Variable names (`{$name}`) have no such repair: renaming a `*` row
breaks its references, and each affected command and list is named in
the warnings.

## Validation loop

After generating or editing, always run:

```
baton check <path-to-project-dir>     # or: baton check <project-name>
```

Exit code 0 means valid; otherwise fix each reported `WARN` line and
re-run. The checks cover: unparseable files, duplicate command names,
unknown shell values, undefined `{$name}` references (commands and
list values), missing templates, workflow steps naming unknown
commands, orphaned vars.tsv rows, and fixed-value rows that match no
slot of their template.

## Common mistakes

1. Spaces instead of tabs in a `.tsv` — the most frequent failure.
2. Adding a header row to `lists/*.tsv` (only commands.tsv and
   vars.tsv have headers).
3. Using `{$name}` for a value the user should pick at run time
   (that is `{name}`), or `{name}` for a fixed path (that is `{$name}`
   plus a `*` row in vars.tsv).
4. A `shell` value other than empty or `ps`.
5. A slot whose name matches no list and has no `slots=` mapping —
   it still runs, but degrades to free-text input.
6. Writing `commands.local.json` or `workflows.json` — never generate
   these.
7. Renaming a command and editing its definition in one edit — split
   it in two so the rename repair can match (see Renames).
