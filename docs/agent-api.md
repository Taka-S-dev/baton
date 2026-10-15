# Agent interface

Use `baton api` for project operations instead of rewriting app-owned files.
It exposes the TUI's project definitions without automating keyboard events.
Command reference rewrites and saved-input persistence share code with the TUI;
TSV edits use the existing row-preserving writer.

Start with [the agent quickstart](agent-quickstart.md) for the shortest working sequence.

## Transport

```text
baton api schema
baton api --file request.json
```

Without arguments, the API reads one UTF-8 JSON request from stdin and returns
one JSON response to stdout: `{"version":1,"ok":true,"data":{}}`.
Errors have `ok:false`, `code` and `error`, and exit nonzero.
Execution failures also include the plan and captured results in `data`.
Unknown JSON fields, trailing requests, and requests over 8 MiB are rejected.
Only `run.execute` executes commands.

`project` accepts a project name or explicit directory path. Creation takes a
plain name under the normal `BATON_PROJECTS_DIR` directory.

## Inspect

```json
{"action":"project.list"}
```

```json
{"action":"project.get","project":"demo"}
```

The state includes commands, workflows, list entries, shared variables,
remaining input definitions, diagnostics and a `revision` hash.
Derived commands include their saved inputs in `values`.
`project.check` returns the same state and diagnostics without writing.

Inspection and run planning are available while the TUI is open.
Close TUI sessions before API edits/execution: TUI and API writers share an
exclusive projects-directory lock, preventing an open stale form from
overwriting agent changes. Raw editors do not honor this lock; do not edit
project files concurrently. The API also rechecks the revision before commit.

## Apply

Read the revision, then preview a batch with `dry_run:true`:

```json
{
  "action": "apply",
  "project": "demo",
  "revision": "<revision from project.get>",
  "dry_run": true,
  "operations": [
    {"op":"variable.create","name":"root","value":"C:/work"},
    {"op":"list.create","name":"env","entries":[
      {"value":"dev","label":"Development"},
      {"value":"prod","label":"Production"}
    ]},
    {"op":"command.create","command":{
      "name":"show-env","cmd":"echo {env}","workdir":"{$root}"
    }},
    {"op":"command.create","command":{
      "name":"show-dev","template":"show-env","values":{"env":"dev"}
    }},
    {"op":"workflow.create","workflow":{
      "name":"check-dev","commands":["show-dev"]
    }}
  ]
}
```

Inspect the before/after changes, then submit the same request with
`dry_run:false` and the original revision (`base_revision` in the preview).
The preview's `state.revision` is the proposed state, not the current revision.

Operations run in order against a private copy. Only a fully valid final
project is committed. Failed operations/validation leave the project unchanged.
Successful responses include `committed:true` and the new state/revision.

| Operation | Arguments |
|---|---|
| `command.create` | `command`: full definition |
| `command.update` | `name`, `command`: full replacement with the same name |
| `command.rename` | `name`, `new_name`; retargets workflow/template/saved-input references |
| `command.delete` | `name`; refuses referenced commands |
| `command.inputs` | `name`, `values`: replaces all saved inputs; empty object restores prompts |
| `workflow.create` | `workflow`: `name` and ordered `commands` |
| `workflow.update` | `name`, `workflow`: replace/reorder steps |
| `workflow.rename` | `name`, `new_name` |
| `workflow.delete` | `name` |
| `list.create` / `list.update` | `name`, complete ordered `entries` (value, optional label) |
| `list.delete` | `name`; refuses lists used by command inputs |
| `variable.create` / `variable.update` | `name`, `value` |
| `variable.delete` | `name`; rejects undefined references in the resulting project |

For command.inputs, values must be supplied (use {} to clear). For list.update,
entries must be supplied (use [] to empty). Omitting either is an error.

Updates replace definitions, not partial patches: retain fields you need from
the read response. Create never silently upserts. Retrying a committed request
with its old revision is rejected, preventing duplicate entries.

New direct commands go into `commands.tsv`; derived commands and scoped inputs
use app storage. `commands.json` remains read-only, matching TUI ownership.
JSON definitions can be inspected and run.

Update/delete dependents before deleting a referenced entity in the same batch.
List values are copied inputs, not foreign keys: a list update does not silently
rewrite saved commands. Include explicit `command.inputs` and `list.update`
operations for values you want to propagate. This represents the choices a
human makes in the TUI's propagation dialog.

## Projects and recovery

- `project.create`: name, optional dry-run; creates an empty valid project.
- `project.delete`: name, revision, optional dry-run; moves settings into
  `.baton-trash` and returns the archive path. Working directories are untouched.
  Only direct children of the projects directory can be archived.
- Restore by moving the archived directory back under projects, as with TUI deletion.

An app-owned `.baton-transaction.json` retains before/after files until commit
finishes. An interrupted transaction blocks loading its project. Recover with:

```json
{"action":"project.recover","project":"demo"}
```

Recovery keeps a fully completed commit; otherwise it restores transaction
files to their previous versions. External changes cause recovery to refuse
overwriting and retain the journal for inspection. This is recoverable
multi-file persistence, not instantaneous multi-file visibility or guaranteed
power-loss atomicity.

A crash may leave `.baton-lock`, which contains the owner's PID. Verify that
process has exited before removing the stale lock and retrying. Never remove a
live process's lock. Journals may contain sensitive project data.

## Plan and execute

```json
{
  "action":"run.plan",
  "project":"demo",
  "cwd":"C:/work",
  "steps":[{"command":"show-env","inputs":{"env":"dev"}}]
}
```

The plan returns expanded commands/workdirs and a `plan_token`. Missing inputs
return step indices, slot definitions and list choices; there is no token until
all inputs are supplied. Reserved path inputs use the definition's `name` as
their input-map key. Substitutions have the same literal/quoting semantics as
the TUI.

Send the same project, steps, inputs and cwd with `action:"run.execute"` and
the token. Changed project definitions or invocation invalidate the token.
Execution is sequential and stops at the first failure. Timeout is 120 seconds
per command by default; `timeout_seconds` accepts 1–3600. Captured stdout/stderr
are limited to 1 MiB each per command with a truncation flag. Stdin is closed;
interactive programs belong in the TUI. Timeout terminates the directly launched
process; independently spawned background processes are not managed.

Use `workflow:"check-dev"` instead of steps for a workflow with saved inputs.
For workflows with remaining inputs, read the ordered command names and pass
equivalent steps with inputs. Attempts are recorded in normal recent history.

This is a local CLI transport, not an installed MCP server or network listener.
Agents with a shell tool can invoke it directly. An MCP adapter can reuse the
same service later.
