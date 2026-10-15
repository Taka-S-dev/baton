# Baton API: agent quickstart

Use this interface for project changes. Do not write app-owned JSON or scoped
vars.tsv rows directly. This is a JSON CLI, not positional subcommands:
`baton api project.get` is **not** valid.

## 1. Discover and read

Run `baton api schema` once. Its output includes examples, required operation
fields, response paths, and error recovery instructions.

Write a UTF-8 request file outside the project definition files:

```json
{"action":"project.get","project":"demo"}
```

Run `baton api --file request.json`. Alternatively pass the JSON directly on
stdin using your process tool's stdin argument, without shell quoting.

Check the exit status and `ok`. The current revision is `data.revision`.
Inspect `data.commands`, `data.command_sources` and `data.references`.
Use `project.list` if the project name is unknown.

## 2. Preview a change

Insert the actual revision you just received:

```json
{
  "action":"apply",
  "project":"demo",
  "revision":"<data.revision>",
  "dry_run":true,
  "operations":[
    {"op":"command.create","command":{"name":"hello","cmd":"echo hello"}}
  ]
}
```

Inspect `data.changes` before/after content.
`data.committed` is false. No project definition has been changed.
The proposed `data.state.revision` is **not** the revision to submit.

## 3. Commit and verify

Send the same request with `dry_run:false` and the original revision
(also returned as preview `data.base_revision`). Check `ok` and
`data.committed:true`. Use `data.state.revision` for the next change.
Call `project.check` if a separate validation result is needed.

A failed batch does not partially save operations. Order dependencies within a
batch: create templates before derived commands; remove workflow references
before deleting commands.

## 4. Edit without dropping data

- `command.update` replaces the full definition. Copy the command from
  `project.get`, change only intended fields, and send the full command.
- `command.rename` handles reference updates; do not rename by delete/create.
- `command.inputs` replaces all saved inputs. Include the complete `values` map.
  To deliberately clear all saved inputs, use `"values":{}`.
- `list.update` replaces the ordered list. Include complete `entries`.
  To deliberately empty it, use `"entries":[]`.
- Missing `values`/`entries` are rejected for these replacement operations.
- `commands.json` is read-only, just as in the TUI. Check `command_sources`.
- List value edits do not automatically rewrite saved input copies.
  Include explicit updates for dependents the user wants changed.

## 5. Execute only when requested

Registration/editing does not require running the registered command.

For an execution request, first call `run.plan` with ordered steps:

```json
{
  "action":"run.plan",
  "project":"demo",
  "steps":[{"command":"hello"}]
}
```

If `data.missing_inputs` is nonempty, its `step` is a zero-based index.
Use `input.name` as the key in that step's `inputs` map, choose from
`choices[].value` when applicable, and call `run.plan` again.
`input.placeholder` is the human-readable token, not necessarily the map key.

When the plan has a token, inspect the expanded commands/workdirs. Send the
same project, steps, inputs and cwd with `action:"run.execute"` and
`plan_token` from `data.plan_token`. Check `data.results` and `data.completed`.
A failed step stops the sequence. Do not blindly repeat completed commands.

## If a request fails

| Code | Next action |
|---|---|
| `conflict` | Read again and re-evaluate the edit; or re-plan execution. |
| `referenced` | Inspect references, then explicitly update dependents. |
| `invalid_request` | Check examples and required fields from schema. |
| `validation_failed` | Fix reported definitions; use project.check. |
| `busy` | Close the TUI or wait for another API writer. Do not force-unlock. |
| `execution_failed` | Inspect per-step results; decide what to retry. |
| `recovery_required` | Use project.recover; retain external-conflict journals. |

TUI-open read/plan calls work; writes/execution require TUI to be closed.
Interactive commands need the TUI: the API closes stdin.
See [the full API reference](agent-api.md) for project deletion, timeouts,
output limits, recovery, and storage ownership.
