package slot

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/Taka-S-dev/baton/internal/model"
)

// Pattern matches {slotName} placeholders.
var Pattern = regexp.MustCompile(`\{(\w+)\}`)

// VarPattern matches {$name} project-variable references. The $ prefix
// keeps them invisible to Pattern, so vars never capture interactive slots.
var VarPattern = regexp.MustCompile(`\{\$(\w+)\}`)

// Def defines a slot with its associated list name.
type Def struct {
	Name     string
	ListName string
}

// HasPlaceholders returns true if the command contains any {slot} placeholders.
func HasPlaceholders(cmd model.Command) bool {
	return Pattern.MatchString(cmd.Cmd) || Pattern.MatchString(cmd.Dir)
}

// GetSlots returns all unique slots to resolve for a command, in order.
// List name comes from cmd.Slots if defined, otherwise defaults to the slot name.
func GetSlots(cmd model.Command) []Def {
	seen := make(map[string]bool)
	var slots []Def
	add := func(name string) {
		if seen[name] {
			return
		}
		seen[name] = true
		listName := name
		if cmd.Slots != nil {
			if ln, ok := cmd.Slots[name]; ok {
				listName = ln
			}
		}
		slots = append(slots, Def{Name: name, ListName: listName})
	}
	for _, m := range Pattern.FindAllStringSubmatch(cmd.Cmd, -1) {
		add(m[1])
	}
	for _, m := range Pattern.FindAllStringSubmatch(cmd.Dir, -1) {
		add(m[1])
	}
	return slots
}

// Apply replaces all {slotName} occurrences in cmd with values from the map.
func Apply(cmd model.Command, values map[string]string) model.Command {
	result := cmd
	for k, v := range values {
		result.Cmd = strings.ReplaceAll(result.Cmd, "{"+k+"}", v)
		result.Dir = strings.ReplaceAll(result.Dir, "{"+k+"}", v)
	}
	return result
}

// ApplyVars replaces defined {$name} references in s with values from vars.
// Undefined references are left as-is (surfaced as a load warning), and the
// replacement is a single pass — a value containing {$other} stays literal.
func ApplyVars(s string, vars map[string]string) string {
	return VarPattern.ReplaceAllStringFunc(s, func(ref string) string {
		if v, ok := vars[ref[2:len(ref)-1]]; ok {
			return v
		}
		return ref
	})
}

// ApplyVarsToCommand applies ApplyVars to the command line and workdir.
func ApplyVarsToCommand(cmd model.Command, vars map[string]string) model.Command {
	cmd.Cmd = ApplyVars(cmd.Cmd, vars)
	cmd.Dir = ApplyVars(cmd.Dir, vars)
	return cmd
}

// UndefinedVars returns the {$name} references in s with no definition.
func UndefinedVars(s string, vars map[string]string) []string {
	var out []string
	for _, m := range VarPattern.FindAllStringSubmatch(s, -1) {
		if _, ok := vars[m[1]]; !ok {
			out = append(out, m[1])
		}
	}
	return out
}

// LoadVars loads project variables from vars.tsv in projectDir.
// The format is three columns — command / name / value — where the
// command column scopes a saved command's fixed slot value and "*"
// marks a global usable as {$name}. Internally both are kept in one
// map: globals under "name", scoped values under "command.name".
// The file is optional, and a broken line must not block the project
// from loading, so parse problems come back as warnings.
func LoadVars(projectDir string) (map[string]string, []string) {
	vars := make(map[string]string)
	var warnings []string

	data, err := os.ReadFile(filepath.Join(projectDir, "vars.tsv"))
	if err != nil {
		return vars, nil
	}
	set := func(key, value string, lineNo int) {
		if _, dup := vars[key]; dup {
			warnings = append(warnings, fmt.Sprintf("vars.tsv line %d: duplicate entry %s", lineNo, key))
			return
		}
		vars[key] = value
	}
	for i, line := range strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || (i == 0 && (strings.HasPrefix(line, "command\t") || strings.HasPrefix(line, "name\t"))) {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		switch {
		case len(parts) >= 3 && strings.TrimSpace(parts[2]) != "":
			cmd, name := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
			value := strings.TrimSpace(parts[2])
			if cmd == "*" || cmd == "" {
				set(name, value, i+1)
			} else {
				set(cmd+"."+name, value, i+1)
			}
		case len(parts) == 2 && strings.TrimSpace(parts[1]) != "":
			// Legacy two-column rows: "command.slot<TAB>value" or a
			// global "name<TAB>value". Rewritten in the current format
			// on the next save.
			set(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), i+1)
		default:
			warnings = append(warnings, fmt.Sprintf("vars.tsv line %d: missing value", i+1))
		}
	}
	return vars, warnings
}

// SaveVars writes vars to projectDir/vars.tsv — globals ("*" rows)
// first, then per-command values, each sorted by name. An empty map
// only writes when the file already exists (emptying it), so projects
// that never use vars never grow one.
func SaveVars(projectDir string, vars map[string]string) error {
	path := filepath.Join(projectDir, "vars.tsv")
	if len(vars) == 0 {
		if _, err := os.Stat(path); err != nil {
			return nil
		}
	}
	var globals, scoped []string
	for k := range vars {
		if strings.Contains(k, ".") {
			scoped = append(scoped, k)
		} else {
			globals = append(globals, k)
		}
	}
	sort.Strings(globals)
	sort.Strings(scoped)
	lines := []string{"command\tname\tvalue"}
	for _, k := range globals {
		lines = append(lines, "*\t"+k+"\t"+vars[k])
	}
	for _, k := range scoped {
		i := strings.LastIndex(k, ".")
		lines = append(lines, k[:i]+"\t"+k[i+1:]+"\t"+vars[k])
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(strings.Join(lines, "\n")+"\n"), 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Per-command fixed values are stored as "command.slot" names. The
// separator is the LAST dot: command names may contain dots, slot names
// ({\w+}) cannot.

// CommandValues extracts cmdName's fixed slot values from vars, or nil.
func CommandValues(vars map[string]string, cmdName string) map[string]string {
	prefix := cmdName + "."
	var out map[string]string
	for k, v := range vars {
		if strings.HasPrefix(k, prefix) && !strings.Contains(k[len(prefix):], ".") && k != prefix {
			if out == nil {
				out = make(map[string]string)
			}
			out[k[len(prefix):]] = v
		}
	}
	return out
}

// SetCommandValues replaces cmdName's fixed slot values in vars.
func SetCommandValues(vars map[string]string, cmdName string, values map[string]string) {
	prefix := cmdName + "."
	for k := range vars {
		if strings.HasPrefix(k, prefix) && !strings.Contains(k[len(prefix):], ".") {
			delete(vars, k)
		}
	}
	for k, v := range values {
		vars[prefix+k] = v
	}
}

// PruneCommandValues drops "command.slot" entries whose command no
// longer exists, so deletes and renames don't leave orphan lines.
func PruneCommandValues(vars map[string]string, keep func(cmdName string) bool) {
	for k := range vars {
		if i := strings.LastIndex(k, "."); i > 0 && !keep(k[:i]) {
			delete(vars, k)
		}
	}
}

// HighlightSlot replaces resolved slots with their values and highlights the
// current slot with cyan markers, for display in the context panel.
func HighlightSlot(text, currentSlot string, resolved map[string]string) string {
	return Pattern.ReplaceAllStringFunc(text, func(m string) string {
		name := m[1 : len(m)-1]
		if v, ok := resolved[name]; ok {
			return v
		}
		if name == currentSlot {
			return "\x1b[1;96m" + m + "\x1b[0m"
		}
		return m
	})
}

// LoadLists loads all .tsv files from listsDir.
func LoadLists(listsDir string) map[string][]model.ListEntry {
	result := make(map[string][]model.ListEntry)
	entries, err := os.ReadDir(listsDir)
	if err != nil {
		return result
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".tsv") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".tsv")
		data, err := os.ReadFile(filepath.Join(listsDir, e.Name()))
		if err != nil {
			continue
		}
		var list []model.ListEntry
		for _, line := range strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, "\t", 2)
			value := strings.TrimSpace(parts[0])
			label := ""
			if len(parts) > 1 {
				label = strings.TrimSpace(parts[1])
			}
			if value != "" {
				list = append(list, model.ListEntry{Value: value, Label: label})
			}
		}
		if len(list) > 0 {
			result[name] = list
		}
	}
	return result
}

// SaveList saves entries to listsDir/name.tsv.
func SaveList(listsDir, name string, entries []model.ListEntry) error {
	if err := os.MkdirAll(listsDir, 0755); err != nil {
		return err
	}
	var lines []string
	for _, e := range entries {
		if e.Label != "" {
			lines = append(lines, e.Value+"\t"+e.Label)
		} else {
			lines = append(lines, e.Value)
		}
	}
	path := filepath.Join(listsDir, name+".tsv")
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(strings.Join(lines, "\n")), 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// MaterializeCommand bakes a template-derived command: Cmd/Dir/Shell/Slots
// are recomputed from the template with stored Values applied, so template
// edits propagate on every load. Slots without a stored value remain as
// {placeholders} to be resolved interactively at run time.
// The previously baked Cmd acts as a fallback when the template is missing.
// Concrete commands (no Template) are returned unchanged.
func MaterializeCommand(cmd model.Command, config model.Config) (model.Command, error) {
	if cmd.Template == "" {
		return cmd, nil
	}
	tpl, found := config.FindCommand(cmd.Template)
	if !found || tpl.Template != "" {
		if cmd.Cmd != "" {
			// Keep the last baked command so the entry still runs.
			return cmd, fmt.Errorf("template not found: %s (using last saved command)", cmd.Template)
		}
		return cmd, fmt.Errorf("template not found: %s", cmd.Template)
	}
	resolved := Apply(tpl, cmd.Values)
	cmd.Cmd = resolved.Cmd
	cmd.Dir = resolved.Dir
	if cmd.Shell == "" {
		cmd.Shell = tpl.Shell
	}
	if cmd.Group == "" {
		cmd.Group = tpl.Group
	}
	cmd.Slots = tpl.Slots
	return cmd, nil
}
