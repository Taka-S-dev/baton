package projectops

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Taka-S-dev/baton/internal/config"
	mdl "github.com/Taka-S-dev/baton/internal/model"
	"github.com/Taka-S-dev/baton/internal/slot"
	"github.com/Taka-S-dev/baton/internal/store"
)

type Operation struct {
	Op       string            `json:"op"`
	Name     string            `json:"name,omitempty"`
	NewName  string            `json:"new_name,omitempty"`
	Command  *mdl.Command      `json:"command,omitempty"`
	Workflow *mdl.Workflow     `json:"workflow,omitempty"`
	Entries  []mdl.ListEntry   `json:"entries"`
	Value    *string           `json:"value,omitempty"`
	Values   map[string]string `json:"values"`
}

func ValidName(name string) error {
	if strings.TrimSpace(name) != name || name == "" || name == "." || name == ".." || strings.ContainsAny(name, "/\\<>:\"|?*\t\r\n\x00") || strings.HasSuffix(name, ".") {
		return fmt.Errorf("invalid name %q", name)
	}
	base := strings.ToUpper(strings.Split(name, ".")[0])
	if base == "CON" || base == "PRN" || base == "AUX" || base == "NUL" || regexp.MustCompile("^(COM|LPT)[0-9]$").MatchString(base) {
		return fmt.Errorf("reserved name %q", name)
	}
	return nil
}
func cell(s string) error {
	if strings.ContainsAny(s, "\t\r\n\x00") {
		return fmt.Errorf("tabs, newlines and NUL are not allowed in TSV values")
	}
	return nil
}
func validateCommand(c mdl.Command) error {
	if strings.TrimSpace(c.Name) == "" || c.Name == "*" {
		return fmt.Errorf("command name is required")
	}
	for _, v := range []string{c.Name, c.Cmd, c.Dir, c.Group, c.Shell, c.Template} {
		if err := cell(v); err != nil {
			return err
		}
	}
	if c.Shell != "" && c.Shell != "ps" {
		return fmt.Errorf("shell must be empty or ps")
	}
	if c.Template == "" && strings.TrimSpace(c.Cmd) == "" {
		return fmt.Errorf("cmd or template is required")
	}
	if c.Template == "" && len(c.Values) > 0 {
		return fmt.Errorf("saved inputs require a template")
	}
	for k, v := range c.Values {
		if k == "" || strings.ContainsAny(k, ".\t\r\n") {
			return fmt.Errorf("invalid saved input key %q", k)
		}
		if err := cell(v); err != nil {
			return err
		}
	}
	for k, v := range c.Slots {
		if !regexp.MustCompile(`^\w+$`).MatchString(k) || strings.ContainsAny(v, ",=\t\r\n") {
			return fmt.Errorf("invalid slot mapping")
		}
		if err := ValidName(v); err != nil {
			return err
		}
	}
	return nil
}
func saveCommand(dir string, p *config.Project, old *mdl.Command, c mdl.Command) error {
	if err := validateCommand(c); err != nil {
		return err
	}
	if old != nil && old.Source == "json" {
		return fmt.Errorf("commands.json is read-only; edit its definition by hand")
	}
	if old != nil && old.Source == "tsv" && c.Template != "" {
		return fmt.Errorf("cannot convert a TSV definition into a derived command; create a new saved command")
	}
	if c.Template != "" {
		tpl, ok := p.Config.FindCommand(c.Template)
		if !ok || tpl.Template != "" {
			return fmt.Errorf("unknown or derived template %q", c.Template)
		}
		allowed := map[string]bool{}
		for _, d := range slot.GetSlots(tpl) {
			allowed[d.Name] = true
		}
		for k := range c.Values {
			if !allowed[k] {
				return fmt.Errorf("%q is not a slot of %q", k, c.Template)
			}
		}
		var err error
		c, err = slot.MaterializeCommand(c, p.Config)
		if err != nil {
			return err
		}
	}
	if old == nil && c.Template == "" {
		_, err := config.AppendCommandTSV(dir, c)
		return err
	}
	if old != nil && old.Source == "tsv" {
		_, err := config.UpdateCommandTSV(dir, old.Name, c)
		if err == nil {
			for i, v := range p.Config.Base {
				if v.Name == old.Name {
					c.Source = "tsv"
					p.Config.Base[i] = c
					break
				}
			}
		}
		return err
	}
	found := false
	if old != nil {
		for i, v := range p.Config.Commands {
			if v.Name == old.Name {
				p.Config.Commands[i] = c
				found = true
				break
			}
		}
	}
	if !found {
		p.Config.Commands = append(p.Config.Commands, c)
	}
	var err error
	p.Vars, err = SaveConfig(dir, p.Config, p.Vars)
	return err
}

// Apply operates on a private staged project. The caller validates and commits the batch.
func Apply(dir string, o Operation) error {
	p, err := config.LoadProject(dir)
	if err != nil {
		return err
	}
	if p.Vars == nil {
		p.Vars = map[string]string{}
	}
	switch o.Op {
	case "command.create", "command.update":
		if o.Command == nil {
			return fmt.Errorf("command is required")
		}
		c := *o.Command
		target := o.Name
		if target == "" {
			target = c.Name
		}
		old, exists := p.Config.FindCommand(target)
		if o.Op == "command.create" {
			if _, ok := p.Config.FindCommand(c.Name); ok {
				return fmt.Errorf("command already exists: %s", c.Name)
			}
			return saveCommand(dir, &p, nil, c)
		}
		if !exists {
			return fmt.Errorf("command not found: %s", target)
		}
		if c.Name != target {
			return fmt.Errorf("use command.rename to change a name")
		}
		return saveCommand(dir, &p, &old, c)
	case "command.inputs":
		if o.Values == nil {
			return fmt.Errorf("values is required; use {} explicitly to clear saved inputs")
		}
		c, ok := p.Config.FindCommand(o.Name)
		if !ok || c.Template == "" {
			return fmt.Errorf("saved command not found: %s", o.Name)
		}
		old := c
		c.Values = o.Values
		return saveCommand(dir, &p, &old, c)
	case "command.rename":
		for _, other := range p.Config.Base {
			if other.Source == "json" && other.Template == o.Name {
				return fmt.Errorf("rename would require editing read-only commands.json reference %q", other.Name)
			}
		}
		c, ok := p.Config.FindCommand(o.Name)
		if !ok {
			return fmt.Errorf("command not found: %s", o.Name)
		}
		if o.NewName == "" {
			return fmt.Errorf("new_name is required")
		}
		if _, ok := p.Config.FindCommand(o.NewName); ok {
			return fmt.Errorf("command already exists: %s", o.NewName)
		}
		old := c
		c.Name = o.NewName
		// Rename references before saving the local layer, which prunes orphan inputs.
		RetargetCommandRefs(&p.Config, p.Workflows, p.Vars, o.Name, o.NewName)
		if err := saveCommand(dir, &p, &old, c); err != nil {
			return err
		}
		if _, err = SaveConfig(dir, p.Config, p.Vars); err != nil {
			return err
		}
		return store.SaveWorkflows(dir, p.Workflows)
	case "command.delete":
		c, ok := p.Config.FindCommand(o.Name)
		if !ok {
			return fmt.Errorf("command not found: %s", o.Name)
		}
		for _, wf := range p.Workflows {
			for _, n := range wf.Commands {
				if n == o.Name {
					return fmt.Errorf("referenced by workflow %q; update/delete it first", wf.Name)
				}
			}
		}
		for _, other := range p.Config.AllCommands() {
			if other.Template == o.Name {
				return fmt.Errorf("referenced by saved command %q", other.Name)
			}
		}
		if c.Source == "json" {
			return fmt.Errorf("commands.json is read-only")
		}
		if c.Source == "tsv" {
			if _, err := config.DeleteCommandsTSV(dir, []string{o.Name}); err != nil {
				return err
			}
		}
		for i, v := range p.Config.Commands {
			if v.Name == o.Name {
				p.Config.Commands = append(p.Config.Commands[:i], p.Config.Commands[i+1:]...)
				break
			}
		}
		slot.SetCommandValues(p.Vars, o.Name, nil)
		_, err = SaveConfig(dir, p.Config, p.Vars)
		return err
	case "workflow.create", "workflow.update", "workflow.rename", "workflow.delete":
		idx := -1
		for i, w := range p.Workflows {
			if w.Name == o.Name {
				idx = i
				break
			}
		}
		if o.Op == "workflow.create" || o.Op == "workflow.update" {
			if o.Workflow == nil {
				return fmt.Errorf("workflow is required")
			}
			w := *o.Workflow
			if err := ValidName(w.Name); err != nil {
				return err
			}
			if len(w.Commands) == 0 {
				return fmt.Errorf("workflow must contain commands")
			}
			for _, n := range w.Commands {
				if _, ok := p.Config.FindCommand(n); !ok {
					return fmt.Errorf("unknown command %q", n)
				}
			}
			if o.Op == "workflow.create" {
				for _, v := range p.Workflows {
					if v.Name == w.Name {
						return fmt.Errorf("workflow already exists")
					}
				}
				p.Workflows = append(p.Workflows, w)
			} else {
				if idx < 0 {
					return fmt.Errorf("workflow not found")
				}
				if w.Name != o.Name {
					return fmt.Errorf("use workflow.rename")
				}
				p.Workflows[idx] = w
			}
		} else {
			if idx < 0 {
				return fmt.Errorf("workflow not found")
			}
			if o.Op == "workflow.delete" {
				p.Workflows = append(p.Workflows[:idx], p.Workflows[idx+1:]...)
			} else {
				if err := ValidName(o.NewName); err != nil {
					return err
				}
				for _, v := range p.Workflows {
					if v.Name == o.NewName {
						return fmt.Errorf("workflow already exists")
					}
				}
				p.Workflows[idx].Name = o.NewName
			}
			if store.LoadLastWorkflow(dir) == o.Name {
				name := ""
				if o.Op == "workflow.rename" {
					name = o.NewName
				}
				store.SaveLastWorkflow(dir, name)
			}
		}
		return store.SaveWorkflows(dir, p.Workflows)
	case "list.create", "list.update", "list.delete":
		if o.Op == "list.update" && o.Entries == nil {
			return fmt.Errorf("entries is required; use [] explicitly to empty a list")
		}
		if err := ValidName(o.Name); err != nil {
			return err
		}
		_, exists := p.Lists[o.Name]
		if o.Op == "list.create" && exists {
			return fmt.Errorf("list already exists")
		}
		if o.Op != "list.create" && !exists {
			return fmt.Errorf("list not found")
		}
		if o.Op == "list.delete" {
			for _, c := range p.Config.AllCommands() {
				for _, d := range slot.GetSlots(c) {
					if d.ListName == o.Name {
						return fmt.Errorf("list referenced by command %q", c.Name)
					}
				}
			}
			return os.Remove(filepath.Join(dir, "lists", o.Name+".tsv"))
		}
		for _, e := range o.Entries {
			if err := cell(e.Value); err != nil {
				return err
			}
			if err := cell(e.Label); err != nil {
				return err
			}
			if strings.TrimSpace(e.Value) == "" {
				return fmt.Errorf("list value cannot be empty")
			}
		}
		return slot.SaveList(filepath.Join(dir, "lists"), o.Name, o.Entries)
	case "variable.create", "variable.update", "variable.delete":
		if !regexp.MustCompile(`^\w+$`).MatchString(o.Name) {
			return fmt.Errorf("variable name must match [A-Za-z0-9_]+")
		}
		_, exists := p.Vars[o.Name]
		if o.Op == "variable.create" && exists {
			return fmt.Errorf("variable already exists")
		}
		if o.Op != "variable.create" && !exists {
			return fmt.Errorf("variable not found")
		}
		if o.Op == "variable.delete" {
			delete(p.Vars, o.Name)
		} else {
			if o.Value == nil {
				return fmt.Errorf("value is required")
			}
			if err := cell(*o.Value); err != nil {
				return err
			}
			p.Vars[o.Name] = *o.Value
		}
		return slot.SaveVars(dir, p.Vars)
	default:
		return fmt.Errorf("unknown operation %q", o.Op)
	}
}
