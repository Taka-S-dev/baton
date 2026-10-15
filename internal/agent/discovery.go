package agent

// Examples use the same envelope as stdin requests. Placeholder revisions must
// be taken from a preceding read, never guessed or copied from an old request.
func examples() map[string]any {
	return map[string]any{
		"inspect":                map[string]any{"action": "project.get", "project": "demo"},
		"preview_create_command": map[string]any{"action": "apply", "project": "demo", "revision": "<data.revision from project.get>", "dry_run": true, "operations": []any{map[string]any{"op": "command.create", "command": map[string]any{"name": "hello", "cmd": "echo hello"}}}},
		"clear_saved_inputs":     map[string]any{"action": "apply", "project": "demo", "revision": "<current revision>", "dry_run": true, "operations": []any{map[string]any{"op": "command.inputs", "name": "saved-command", "values": map[string]string{}}}},
		"plan":                   map[string]any{"action": "run.plan", "project": "demo", "steps": []any{map[string]any{"command": "hello"}}},
		"execute":                map[string]any{"action": "run.execute", "project": "demo", "steps": []any{map[string]any{"command": "hello"}}, "plan_token": "<data.plan_token from identical run.plan request>"},
	}
}
func requiredOperationFields() map[string][]string {
	return map[string][]string{
		"command.create":  {"command"},
		"command.update":  {"name", "command"},
		"command.rename":  {"name", "new_name"},
		"command.delete":  {"name"},
		"command.inputs":  {"name", "values"},
		"workflow.create": {"workflow"},
		"workflow.update": {"name", "workflow"},
		"workflow.rename": {"name", "new_name"},
		"workflow.delete": {"name"},
		"list.create":     {"name"},
		"list.update":     {"name", "entries"},
		"list.delete":     {"name"},
		"variable.create": {"name", "value"},
		"variable.update": {"name", "value"},
		"variable.delete": {"name"},
	}
}
func errorRecovery() map[string]string {
	return map[string]string{
		"conflict":          "Read project.get again and reconsider the diff. For execution, repeat run.plan with identical inputs and use its new token. Do not blindly resubmit writes.",
		"referenced":        "Inspect data.references from project.get. Update/delete the listed dependents first in the same apply batch.",
		"validation_failed": "Inspect the error and project.check diagnostics; correct definitions before retrying.",
		"busy":              "A TUI/API writer is active. Wait or close TUI before retrying; never remove a live process lock.",
		"invalid_request":   "Check schema examples and required_operation_fields. Updates replace definitions; use explicit {} or [] when clearing.",
		"not_found":         "Read project.list/project.get and use an existing name.",
		"execution_failed":  "Inspect data.results; execution stops at the first failure. Do not automatically repeat already-completed commands.",
		"recovery_required": "Use project.recover for an interrupted transaction; retain a journal with external conflicts for manual inspection.",
	}
}
