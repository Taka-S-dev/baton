# Generating baton projects

This file is for AI agents (and scripts) that generate baton project
files. It is the complete spec: everything needed to produce a valid
project is on this page, and `baton check` verifies the result.

## Ownership — which files you may write

| File | Generate? |
|------|-----------|
| `commands.tsv` | yes — command definitions (the baton TUI also appends, edits, or deletes individual rows on user request; untargeted lines are never touched) |
| `lists/<name>.tsv` | yes — selection lists for `{slot}` placeholders |
| `vars.tsv` | yes, `*` rows only (see below) |
| `commands.local.json`, `workflows.json`, `.last_workflow` | **never** — owned and written by the baton TUI |

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
| name | yes | unique within the project |
| group | no | free label, used for search |
| workdir | no | working directory; `{slot}` and `{$var}` allowed; empty = directory baton was started from |
| cmd | yes | command line; `{slot}` and `{$var}` allowed |
| shell | no | empty = `cmd.exe` on Windows / `sh` elsewhere; `ps` = PowerShell. Any other value falls back to the default and warns |
| slots | no | comma-separated `slot=listname` pairs mapping a slot to a list; an unmapped slot uses the list with the same name |

Two placeholder kinds — do not mix them up:

- `{name}` — **interactive slot**: the user picks a value at run time,
  from `lists/<name>.tsv` (or the list mapped in the slots column).
  If no such list exists, the picker falls back to free-text input.
- `{$name}` — **variable**: substituted silently from a `*` row in
  vars.tsv. Never prompts. Use for paths and other constants that
  change per machine or per project phase.

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
```

- `*` rows are globals, referenced as `{$name}`. These are the only
  rows you should generate.
- Rows naming a command hold that saved command's fixed slot values;
  baton writes and maintains them.
- Substitution is a single pass (no recursion). An undefined `{$name}`
  stays literal and is reported as a warning.

## Validation loop

After generating, always run:

```
baton check <path-to-project-dir>     # or: baton check <project-name>
```

Exit code 0 means valid; otherwise fix each reported `WARN` line and
re-run. The checks cover: unparseable files, duplicate command names,
unknown shell values, undefined `{$name}` references (commands and
list values), missing templates, workflow steps naming unknown
commands, and orphaned vars.tsv rows.

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
