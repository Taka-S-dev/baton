package slot

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Taka-S-dev/baton/internal/model"
)

// Pattern matches {slotName} placeholders.
var Pattern = regexp.MustCompile(`\{(\w+)\}`)

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
// List name comes from cmd.Vars if defined, otherwise defaults to the slot name.
func GetSlots(cmd model.Command) []Def {
	seen := make(map[string]bool)
	var slots []Def
	add := func(name string) {
		if seen[name] {
			return
		}
		seen[name] = true
		listName := name
		if cmd.Vars != nil {
			if ln, ok := cmd.Vars[name]; ok {
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
	return os.WriteFile(filepath.Join(listsDir, name+".tsv"), []byte(strings.Join(lines, "\n")), 0644)
}
