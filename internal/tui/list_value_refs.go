package tui

import (
	"fmt"
	"strings"

	"github.com/Taka-S-dev/baton/internal/slot"
)

// Match saved answers against the template's slot-to-list mapping.
// Equal text in an unrelated command or list is not a reference.
func (m Model) listValueRefs(listName, value string) []string {
	if value == "" {
		return nil
	}
	var refs []string
	for _, c := range m.config.AllCommands() {
		if c.Template == "" {
			continue
		}
		tpl, ok := m.config.FindCommand(c.Template)
		if !ok {
			continue
		}
		for _, s := range slot.GetSlots(tpl) {
			if s.ListName != listName {
				continue
			}
			fixed, exists := c.Values[s.Name]
			if override, ok := m.vars[c.Name+"."+s.Name]; ok {
				fixed, exists = override, true
			}
			if !exists {
				continue
			}
			match := fixed == value
			// Variadic answers are stored space-joined, without selection boundaries.
			// Only split when all list entries are single tokens; otherwise exact only.
			if !match && s.Variadic && len(strings.Fields(value)) == 1 {
				safe := true
				for _, entry := range m.lists[listName] {
					if len(strings.Fields(entry.Value)) != 1 {
						safe = false
						break
					}
				}
				if safe {
					for _, part := range strings.Fields(fixed) {
						if part == value {
							match = true
							break
						}
					}
				}
			}
			if match {
				refs = append(refs, fmt.Sprintf("command %q · %s", c.Name, s.Placeholder()))
			}
		}
	}
	return refs
}
