package model

// Command represents a single executable command definition.
type Command struct {
	Name  string            `json:"name"`
	Group string            `json:"group,omitempty"`
	Dir   string            `json:"dir,omitempty"`
	Cmd   string            `json:"cmd"`
	Shell string            `json:"shell,omitempty"`
	Vars  map[string]string `json:"vars,omitempty"`
}

// Config holds all commands for a project.
type Config struct {
	Commands []Command `json:"commands"`
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
