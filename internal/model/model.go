package model

import "encoding/json"

// Command represents a single executable command definition.
// A command is either concrete (Cmd is set) or template-derived
// (Template names a template whose slots are filled from Values;
// slots without a value are resolved interactively at run time).
type Command struct {
	Name     string            `json:"name"`
	Group    string            `json:"group,omitempty"`
	Dir      string            `json:"workdir,omitempty"`
	Cmd      string            `json:"cmd,omitempty"`
	Shell    string            `json:"shell,omitempty"`
	Slots    map[string]string `json:"slots,omitempty"`
	Template string            `json:"template,omitempty"`
	Values   map[string]string `json:"values,omitempty"`
}

// UnmarshalJSON accepts the legacy "vars" key as an alias for "slots".
func (c *Command) UnmarshalJSON(data []byte) error {
	type plain Command
	aux := struct {
		*plain
		LegacyVars map[string]string `json:"vars"`
	}{plain: (*plain)(c)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if c.Slots == nil {
		c.Slots = aux.LegacyVars
	}
	return nil
}

// Config holds all commands for a project.
// Base is the hand-written layer (commands.json / commands.tsv) and is
// never written back; Commands is the app-managed layer
// (commands.local.json). Whether an entry acts as a template is a
// per-row property (it has {slot} placeholders), not a property of the
// file it lives in.
type Config struct {
	Base     []Command `json:"-"`
	Commands []Command `json:"commands,omitempty"`
}

// AllCommands returns app-managed commands followed by the hand-written layer.
func (c Config) AllCommands() []Command {
	out := make([]Command, 0, len(c.Commands)+len(c.Base))
	out = append(out, c.Commands...)
	out = append(out, c.Base...)
	return out
}

// FindCommand looks up a command by name. App-managed commands
// (commands.local.json) take precedence over the hand-written layer.
func (c Config) FindCommand(name string) (Command, bool) {
	for _, cmd := range c.Commands {
		if cmd.Name == name {
			return cmd, true
		}
	}
	for _, cmd := range c.Base {
		if cmd.Name == name {
			return cmd, true
		}
	}
	return Command{}, false
}

// Workflow is a saved combination of commands with pre-set slot values.
type Workflow struct {
	Name     string                       `json:"name"`
	Commands []string                     `json:"commands"`
	Vars     map[string]map[string]string `json:"vars,omitempty"`
}

// Alias combines multiple commands into a single runnable item.
type Alias struct {
	Name  string                       `json:"name"`
	Steps []string                     `json:"steps"`
	Vars  map[string]map[string]string `json:"vars,omitempty"`
}

// ListEntry is a single entry in a selection list.
type ListEntry struct {
	Value string
	Label string
}

// RunItem is a resolved item ready for execution.
type RunItem struct {
	Name   string
	Cmd    *Command
	Alias  *Alias
	VarMap map[string]string
}

// IsAlias returns true if the RunItem wraps an alias.
func (r RunItem) IsAlias() bool { return r.Alias != nil }
